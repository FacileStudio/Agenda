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
	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/modules/caldav"
	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/modules/settings"
	"github.com/FacileStudio/Agenda/apps/api/modules/spaces"
	"github.com/FacileStudio/Agenda/apps/api/modules/users"
	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"

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

	if err := schemas.Migrate(db); err != nil {
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

	authService := auth.NewService(db, appEnv.StorageDir, appLogger, appEnv.EncryptionKey)
	calendarService := calendars.NewService(db)
	eventService := events.NewService(db)
	spaceService := spaces.NewService(db)
	userService := users.NewService(db, appEnv.StorageDir)
	settingsService := settings.NewService(db)

	router := httpx.NewRouter(httpx.Config{
		Logger: appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins:   appEnv.CORSAllowedOrigins,
			AllowCredentials: true,
		},
	})
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.MaxBodySize(4 << 20)) // 4 MB

	mountRoutes(router, mounts{
		env:       appEnv,
		db:        db,
		sqlDB:     sqlDB,
		auth:      authService,
		calendars: calendarService,
		events:    eventService,
		spaces:    spaceService,
		users:     userService,
		settings:  settingsService,
	})

	clientDir := spa.DirFromEnv()
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
	auth      *auth.Service
	calendars *calendars.Service
	events    *events.Service
	spaces    *spaces.Service
	users     *users.Service
	settings  *settings.Service
}

// mountRoutes claims every URL the app answers. The whole API lives under /api
// so an unknown API path 404s instead of falling through to the SPA catch-all;
// CalDAV, the avatar file server and the legacy OIDC callback stay at the root
// because their URLs are held by external clients, stored rows or Authentik.
func mountRoutes(router chi.Router, m mounts) {
	health.Mount(router, health.DB(m.sqlDB))
	router.Handle("/files/*", http.StripPrefix("/files/", http.FileServer(http.Dir(m.env.StorageDir))))

	caldav.RegisterRoutes(router, m.db)
	if m.env.OIDC != nil {
		router.Get("/auth/oidc/callback", redirectUnderAPI)
	}

	router.Route("/api", func(r chi.Router) {
		auth.RegisterRoutes(r, m.auth, m.env)
		calendars.RegisterRoutes(r, m.calendars, m.auth)
		events.RegisterRoutes(r, m.events, m.auth)
		spaces.RegisterRoutes(r, m.spaces, m.auth)
		users.RegisterRoutes(r, m.users, m.auth)
		settings.RegisterRoutes(r, m.settings, m.auth)
	})
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
