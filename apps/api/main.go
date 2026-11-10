package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/crypto"
	"github.com/FacileStudio/Agenda/apps/api/internal/database"
	docs "github.com/FacileStudio/Agenda/apps/api/internal/documentation"
	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/internal/pool"
	"github.com/FacileStudio/Agenda/apps/api/migrations"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/modules/caldav"
	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/modules/ideas"
	"github.com/FacileStudio/Agenda/apps/api/modules/mcp"
	"github.com/FacileStudio/Agenda/apps/api/modules/settings"
	"github.com/FacileStudio/Agenda/apps/api/modules/spaces"
	"github.com/FacileStudio/Agenda/apps/api/modules/users"
	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/spa"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	porteKit "github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func main() {
	if healthcheck.Handle(os.Args) {
		return
	}
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

// run wires the app together and serves until a shutdown signal arrives,
// returning a process exit code.
//
// Behind Traefik and Cloudflare, RemoteAddr is only the visitor if both are
// trusted: Traefik replaces the forwarded chain rather than extending it, so
// the visitor survives in Cf-Connecting-Ip alone. TRUSTED_PROXIES=private,
// cloudflare fills all three.
func run() int {
	appEnv, err := env.Load()
	appLogger := logger.New(logger.Config{})
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return 1
	}
	var journalClient *journal.Client
	appLogger = logger.New(logger.Config{
		Level: appEnv.LogLevel,
		Wrap: func(handler slog.Handler) slog.Handler {
			if appEnv.JournalURL == "" || appEnv.JournalToken == "" {
				return handler
			}
			journalClient = journal.New(journal.Config{URL: appEnv.JournalURL, Token: appEnv.JournalToken})
			return journal.NewHandler(journalClient, handler)
		},
	})
	if journalClient != nil {
		defer journalClient.Close()
	}

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return 1
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	if handled, err := migrate.Command(context.Background(), os.Args, migrations.Config(sqlDB, appLogger)); handled {
		if err != nil {
			appLogger.Error("migrate", slog.Any("error", err))
			return 1
		}
		return 0
	}

	if err := schemas.MigrateWithIssuer(context.Background(), db, appEnv.IssuerForMigration(), appLogger); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return 1
	}

	if len(appEnv.EncryptionKey) > 0 {
		if err := crypto.MigrateOIDCTokens(db, appEnv.EncryptionKey, appLogger); err != nil {
			appLogger.Warn("OIDC token migration failed", slog.Any("error", err))
		}
	}

	if err := os.MkdirAll(filepath.Join(appEnv.StorageDir, "avatars"), 0o755); err != nil {
		appLogger.Error("failed to prepare storage", slog.Any("error", err))
		return 1
	}

	porteAuth, err := buildAuth(context.Background(), sqlDB, appEnv, appLogger)
	if err != nil {
		appLogger.Error("failed to build authentication", slog.Any("error", err))
		return 1
	}
	sessions, kit, identities := porteAuth.sessions, porteAuth.oidc, porteAuth.identities

	authService := auth.NewService(sessions, porteAuth.users)
	calendarService := calendars.NewService(db)
	eventService := events.NewService(db)
	spaceService := spaces.NewService(db)
	userService := users.NewService(db, authService)
	settingsService := settings.NewService(db)
	mcpService, err := newMCP(context.Background(), sqlDB, appEnv, calendarService, eventService, identities, kit, appLogger)
	if err != nil {
		appLogger.Error("failed to build the MCP server", slog.Any("error", err))
		return 1
	}

	registreSync, deletions, stopSync := startDirectory(db, porteAuth.users, appEnv, appLogger)
	defer stopSync()

	ideasService, err := newIdeas(context.Background(), db, appEnv, identities, registreSync, appLogger)
	if err != nil {
		appLogger.Error("failed to set up ideas", slog.Any("error", err))
		return 1
	}

	if appEnv.Antenne != nil {
		poolClient := pool.NewClient(*appEnv.Antenne, pool.NewMirror(db, eventService, calendarService, appLogger), appLogger)
		poolClient.Start(context.Background())
		defer poolClient.Stop()
	}

	clientDir := spa.DirFromEnv()
	router := httpx.NewRouter(httpx.Config{
		TrustedProxies: appEnv.TrustedProxies,
		CDNProxies:     appEnv.CDNProxies,
		CDNHeader:      appEnv.CDNHeader,
		Logger:         appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins:   appEnv.CORSAllowedOrigins,
			AllowCredentials: true,
		},
		Security: securityConfig(clientDir, appLogger),
	})
	router.Use(middleware.MaxBodySize(4 << 20))

	if err := mountRoutes(router, mounts{
		env:         appEnv,
		db:          db,
		sqlDB:       sqlDB,
		sessions:    sessions,
		kit:         kit,
		auth:        authService,
		calendars:   calendarService,
		events:      eventService,
		spaces:      spaceService,
		users:       userService,
		settings:    settingsService,
		directory:   registreSync,
		deletions:   deletions,
		mcp:         mcpService,
		ideas:       ideasService,
		invitations: invitationsConfig(appEnv, identities, appLogger),
	}); err != nil {
		appLogger.Error("failed to mount routes", slog.Any("error", err))
		return 1
	}

	if spa.Available(clientDir) {
		router.Handle("/*", middleware.Gzip(spa.Handler(spa.Config{Dir: clientDir})))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	addr := ":" + strconv.Itoa(appEnv.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger.Info("server starting", slog.String("addr", addr))
	select {
	case err := <-serverErrCh:
		if !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
			return 1
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return 1
		}
		appLogger.Info("server stopped")
	}

	return 0
}

type mounts struct {
	env       env.Config
	db        *gorm.DB
	sqlDB     *sql.DB
	sessions  *session.Manager
	kit       *oidc.Kit
	auth      *auth.Service
	calendars *calendars.Service
	events    *events.Service
	spaces    *spaces.Service
	users     *users.Service
	settings  *settings.Service
	directory *spaces.Directory
	deletions *spaces.Deletions
	mcp       *mcp.Service
	ideas     *ideas.Service

	invitations *porteKit.InvitationsConfig
}

// startDirectory follows Registre's spaces and its GDPR deletions when SSO is
// configured, signing its pulls with the app's own OIDC client. It returns at
// once: the import, the sync and the deletion poll run in the background, so a
// Registre that is down or not yet serving the directory never holds up boot.
// Without SSO there is nothing to follow, nil receivers and a stop that does
// nothing. All suite apps run one replica, so this is the one deletion poller.
func startDirectory(
	db *gorm.DB, users *porteKit.Store, appEnv env.Config, appLogger *slog.Logger,
) (*spaces.Directory, *spaces.Deletions, context.CancelFunc) {
	if appEnv.OIDC == nil {
		return nil, nil, func() {}
	}
	client := directory.Client{
		Issuer:       appEnv.OIDC.Issuer,
		ClientID:     appEnv.OIDC.ClientID,
		ClientSecret: appEnv.OIDC.ClientSecret,
	}
	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("deletion receiver not started", slog.Any("error", err))
		return nil, nil, func() {}
	}
	sync := spaces.NewDirectory(db, client, appLogger)
	deletions := spaces.NewDeletions(db, client, spaces.Erasers{
		Account: users.EraseAccount, Identity: portepg.New(sqlDB).EraseIdentity,
	}, appEnv.StorageDir, appLogger)
	ctx, stop := context.WithCancel(context.Background())
	go sync.Run(ctx)
	go deletions.Run(ctx)
	return sync, deletions, stop
}

// porteAuth is porte as Agenda wires it: one session manager, the OIDC kit,
// the identity store and the kit's user store over Agenda's users table.
type porteAuth struct {
	sessions   *session.Manager
	oidc       *oidc.Kit
	identities porte.IdentityStore
	users      *porteKit.Store
}

// buildAuth constructs porte over the identity tables. Sign-in goes through
// Registre only; there is no local password kit. Discovery runs here, so an
// unreachable or half-configured issuer fails at boot rather than on
// somebody's first login.
func buildAuth(ctx context.Context, sqlDB *sql.DB, appEnv env.Config, appLogger *slog.Logger) (porteAuth, error) {
	store, err := newPorteStore(sqlDB)
	if err != nil {
		return porteAuth{}, err
	}
	resealInBackground(ctx, store, appLogger)
	users := auth.NewStore(sqlDB)
	cfg := appEnv.Porte()

	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: appLogger})
	if err != nil {
		return porteAuth{}, err
	}
	kit, err := oidc.New(ctx, cfg, oidc.Deps{
		Users:      users,
		Identities: store.Identities(),
		Sessions:   sessions,
		Codes:      store.LoginCodes(),
		Logger:     appLogger,
	})
	if err != nil {
		return porteAuth{}, err
	}
	return porteAuth{sessions: sessions, oidc: kit, identities: store.Identities(), users: users}, nil
}

// mountRoutes claims every URL the app answers. The whole API lives under /api
// so an unknown API path 404s instead of falling through to the SPA catch-all;
// CalDAV, the avatar file server and the legacy OIDC callback stay at the root
// because their URLs are held by external clients, stored rows or Authentik.
func mountRoutes(router chi.Router, m mounts) error {
	health.Mount(router, health.DB(m.sqlDB))
	router.Get("/version", versionHandler)
	apiref.Mount(router, referenceConfig())
	router.Handle("/files/*", http.StripPrefix("/files/", http.FileServer(http.Dir(m.env.StorageDir))))

	caldav.RegisterRoutes(router, m.db, m.auth)
	if m.mcp != nil {
		mcp.RegisterRoutes(router, m.mcp)
	}
	if m.env.OIDC != nil {
		router.Get("/auth/oidc/callback", redirectUnderAPI)
	}

	var mountErr error
	router.Route("/api", func(r chi.Router) {
		mountErr = auth.RegisterRoutes(r, m.sessions, m.kit)
		calendars.RegisterRoutes(r, m.calendars, m.auth)
		events.RegisterRoutes(r, m.events, m.auth)
		spaces.RegisterRoutes(r, m.spaces, m.auth)
		if m.directory != nil && m.deletions != nil {
			spaces.RegisterSync(r, m.directory)
			spaces.RegisterDeletions(r, m.deletions)
		}
		users.RegisterRoutes(r, m.users, m.auth)
		settings.RegisterRoutes(r, m.settings, m.auth)
		if m.mcp != nil {
			mcp.RegisterPendingRoutes(r, m.mcp, m.auth)
		}
		if mountErr == nil {
			mountErr = mountOptional(r, m)
		}
	})
	return mountErr
}

// referenceConfig describes the API reference served at /docs. Registry paths
// are written relative to /api, the one server every documented route hangs off.
func referenceConfig() apiref.Config {
	return apiref.Config{
		Title:       "Agenda API",
		Description: "Self-hosted calendar for creative studios, with CalDAV sync.",
		Servers:     []string{"/api"},
		Registry: docs.Response{Modules: []docs.Module{
			auth.Documentation,
			calendars.Documentation,
			events.Documentation,
			spaces.Documentation,
			users.Documentation,
			settings.Documentation,
			mcp.Documentation,
			ideas.Documentation,
		}},
	}
}

// redirectUnderAPI forwards a legacy root-level API path to its /api twin. The
// OIDC callback URL is registered in Authentik and pinned by OIDC_REDIRECT_URL,
// so it cannot move with the rest of the API.
func redirectUnderAPI(w http.ResponseWriter, r *http.Request) {
	target := "/api" + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// externalConnect lists the origins the built client calls besides itself:
// Ardoise's public plan catalog and the icon API behind iconify-icon.
var externalConnect = []string{"https://api-ardoise.facile.studio", "https://api.iconify.design"}

// securityConfig extends tronc's default security headers with what the built
// client needs: the inline scripts adapter-static writes into its pages, which
// script-src 'self' would otherwise block, and the external origins above. A
// client directory that cannot be hashed is logged and leaves the default
// script policy, which only reports.
func securityConfig(clientDir string, appLogger *slog.Logger) troncmiddleware.SecurityConfig {
	sources := map[string][]string{"connect-src": externalConnect}
	if spa.Available(clientDir) {
		hashes, err := spa.InlineScriptHashes(clientDir)
		if err != nil {
			appLogger.Warn("failed to hash the client's inline scripts", slog.Any("error", err))
		} else {
			sources["script-src"] = hashes
		}
	}
	return troncmiddleware.SecurityConfig{CSPSources: sources}
}
