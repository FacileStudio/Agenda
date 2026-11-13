package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/mcp"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/modules/ideas"
	"github.com/FacileStudio/Agenda/apps/api/modules/spaces"
	"github.com/FacileStudio/porte/directory"
	porteKit "github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

// clientCalls lists every URL apps/client asks for, plus the endpoints held by
// callers outside this repo. The SPA catch-all answers 200 on any extension-less
// path, so a route that quietly stops existing is invisible in the browser: this
// test is the only thing standing between a moved prefix and a dead feature.
var clientCalls = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/auth/config"},
	{http.MethodGet, "/api/me/invitations"},
	{http.MethodPost, "/api/auth/logout"},
	{http.MethodPost, "/api/auth/sync-profile"},
	{http.MethodGet, "/api/auth/oidc"},
	{http.MethodGet, "/api/auth/oidc/callback"},
	{http.MethodGet, "/api/users/"},
	{http.MethodGet, "/api/users/me"},
	{http.MethodPatch, "/api/users/me"},
	{http.MethodGet, "/api/users/me/api-token"},
	{http.MethodPost, "/api/users/me/api-token"},
	{http.MethodDelete, "/api/users/me/api-token"},
	{http.MethodGet, "/api/users/7"},
	{http.MethodGet, "/api/calendars/"},
	{http.MethodPost, "/api/calendars/"},
	{http.MethodGet, "/api/calendars/1/"},
	{http.MethodPut, "/api/calendars/1/"},
	{http.MethodDelete, "/api/calendars/1/"},
	{http.MethodGet, "/api/calendars/1/members"},
	{http.MethodPost, "/api/calendars/1/members"},
	{http.MethodDelete, "/api/calendars/1/members/2"},
	{http.MethodGet, "/api/calendars/1/events/"},
	{http.MethodPost, "/api/calendars/1/events/"},
	{http.MethodGet, "/api/events/3/"},
	{http.MethodPut, "/api/events/3/"},
	{http.MethodDelete, "/api/events/3/"},
	{http.MethodPut, "/api/events/3/occurrences/2026-11-05"},
	{http.MethodDelete, "/api/events/3/occurrences/2026-11-05"},
	{http.MethodPost, "/api/events/3/occurrences/2026-11-05/restore"},
	{http.MethodGet, "/api/spaces/"},
	{http.MethodGet, "/api/spaces/1/"},
	{http.MethodGet, "/api/spaces/1/members"},
	{http.MethodPost, directory.SyncPath},
	{http.MethodPost, directory.DeletionsPath},
	{http.MethodGet, "/api/settings/"},
	{http.MethodPut, "/api/settings/"},
	{http.MethodGet, "/api/ideas"},
	{http.MethodPost, "/api/ideas/generate"},
	{http.MethodPost, "/api/ideas/i1/run"},
	{http.MethodPost, "/api/ideas/i1/dismiss"},
	{http.MethodGet, "/api/ideas/runs/r1"},
	{http.MethodPost, "/api/ideas/runs/r1/approvals/c1"},
	{http.MethodGet, "/api/health"},
	{http.MethodGet, "/api/ready"},
	{http.MethodGet, "/health"},
	{http.MethodGet, "/ready"},
	{http.MethodGet, "/version"},
	{http.MethodGet, "/files/avatars/user-1.png"},
	{http.MethodGet, "/.well-known/caldav"},
	{http.MethodGet, "/dav/"},
	{http.MethodGet, "/auth/oidc/callback"},
}

// stubIssuer serves just enough OpenID discovery for newOIDCHandler to succeed,
// so the router under test carries the /auth/oidc routes.
func stubIssuer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(nil)
	t.Cleanup(server.Close)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"issuer": "` + server.URL + `",
			"authorization_endpoint": "` + server.URL + `/authorize",
			"token_endpoint": "` + server.URL + `/token",
			"jwks_uri": "` + server.URL + `/jwks"
		}`))
	})
	return server.URL
}

// testRouter builds the full router under test from an in-memory configuration.
//
// The porte pieces are built over a nil database and an empty user store,
// which is what the rest of this harness does with the services: these cases
// stop at routing, and an unauthenticated request is refused before anything
// is read.
func testRouter(t *testing.T) chi.Router {
	t.Helper()
	appEnv := env.Config{
		StorageDir: t.TempDir(),
		OIDC: &env.OIDCConfig{
			Issuer:       stubIssuer(t),
			ClientID:     "agenda",
			ClientSecret: "secret",
			RedirectURL:  "https://agenda.example/auth/oidc/callback",
			SuccessURL:   "https://agenda.example/",
		},
	}
	appLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	store := portepg.New(nil)
	sessions, err := session.New(appEnv.Porte(), session.Deps{Sessions: store.Sessions(), Logger: appLogger})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	kit, err := oidc.New(context.Background(), appEnv.Porte(), oidc.Deps{
		Users:      &porteKit.Store{},
		Identities: store.Identities(),
		Codes:      store.LoginCodes(),
		Sessions:   sessions,
		Logger:     appLogger,
	})
	if err != nil {
		t.Fatalf("oidc.New: %v", err)
	}
	authService := auth.NewService(sessions, &porteKit.Store{})

	router := chi.NewRouter()
	if err := mountRoutes(router, mounts{
		env:       appEnv,
		sessions:  sessions,
		kit:       kit,
		auth:      authService,
		directory: spaces.NewDirectory(nil, directory.Client{Issuer: appEnv.OIDC.Issuer}, appLogger),
		deletions: spaces.NewDeletions(nil, directory.Client{Issuer: appEnv.OIDC.Issuer}, spaces.Erasers{}, appEnv.StorageDir, appLogger),
		mcp:       testMCP(t),
		invitations: &porteKit.InvitationsConfig{
			Directory:  directory.Client{Issuer: appEnv.OIDC.Issuer, ClientID: "agenda", ClientSecret: "secret"},
			Identities: store.Identities(),
		},
		ideas: ideas.NewService(nil, ideas.Config{
			URL:   "http://bulle.invalid",
			Token: func(context.Context) (string, error) { return "", nil },
		}),
	}); err != nil {
		t.Fatalf("mountRoutes: %v", err)
	}
	return router
}

func TestEveryCalledURLIsRouted(t *testing.T) {
	router := testRouter(t)

	for _, call := range clientCalls {
		routeContext := chi.NewRouteContext()
		if !router.Match(routeContext, call.method, call.path) {
			t.Errorf("%s %s matches no route", call.method, call.path)
		}
	}
}

func TestNoLocalSignInRouteIsMounted(t *testing.T) {
	router := testRouter(t)

	for _, path := range []string{"/api/auth/register", "/api/auth/login", "/auth/register", "/auth/login"} {
		for _, method := range []string{http.MethodPost, http.MethodGet} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
			if recorder.Code != http.StatusNotFound {
				t.Errorf("%s %s = %d, want 404; sign-in goes through Registre only", method, path, recorder.Code)
			}
		}
	}
}

func TestVersionReportsTheBuild(t *testing.T) {
	router := testRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/version", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), version) {
		t.Errorf("GET /version = %d %q, want 200 carrying %q", recorder.Code, recorder.Body.String(), version)
	}
}

func TestUnknownAPIPathsDoNotReachTheSPA(t *testing.T) {
	router := testRouter(t)
	router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/typo", nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("GET /api/typo = %d, want 404; the SPA is swallowing unknown API paths", recorder.Code)
	}
}

func TestHealthSurvivesTheAPIMount(t *testing.T) {
	router := testRouter(t)

	for _, path := range []string{"/health", "/api/health"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200; the /api mount is shadowing it", path, recorder.Code)
		}
	}
}

func TestLegacyOIDCCallbackRedirectsUnderAPI(t *testing.T) {
	recorder := httptest.NewRecorder()
	redirectUnderAPI(recorder, httptest.NewRequest(http.MethodGet, "/auth/oidc/callback?code=abc&state=xyz", nil))

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/api/auth/oidc/callback?code=abc&state=xyz" {
		t.Errorf("Location = %q; the authorization code must survive the hop", got)
	}
}

func TestRegistreNudgeIsAcceptedWithoutCredentials(t *testing.T) {
	router := testRouter(t)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, directory.SyncPath, nil))
	if recorder.Code != http.StatusAccepted {
		t.Errorf("POST %s = %d, want 202; Registre's nudge carries no credentials", directory.SyncPath, recorder.Code)
	}
}

func TestSecurityConfigAllowsTheClientsInlineScripts(t *testing.T) {
	dir := t.TempDir()
	page := "<html><body><script>window.boot = 1</script></body></html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o600); err != nil {
		t.Fatalf("write page: %v", err)
	}

	config := securityConfig(dir, slog.New(slog.NewTextHandler(io.Discard, nil)))

	scripts := config.CSPSources["script-src"]
	if len(scripts) != 1 || !strings.HasPrefix(scripts[0], "'sha256-") {
		t.Errorf("script-src = %v, want the page's inline script hash", scripts)
	}
	if !slices.Contains(config.CSPSources["connect-src"], "https://api-ardoise.facile.studio") {
		t.Errorf("connect-src = %v, want the Ardoise catalog origin", config.CSPSources["connect-src"])
	}
}

func TestVersionIsTheReleaseVersion(t *testing.T) {
	if version == "dev" {
		t.Skip("version is set by -ldflags at build time")
	}
	raw, err := os.ReadFile("../client/package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	versionHandler(recorder, httptest.NewRequest(http.MethodGet, "/version", nil))
	want := `{"version":"v` + pkg.Version + `"}`
	if strings.TrimSpace(recorder.Body.String()) != want {
		t.Errorf("GET /version = %q, want %q", recorder.Body.String(), want)
	}
}

// testMCP is the MCP service over nil services and an in-memory store,
// enough to mount its routes.
func testMCP(t *testing.T) *mcp.Service {
	t.Helper()
	refuse := func(context.Context, string, *http.Request) (*sdkauth.TokenInfo, error) {
		return nil, sdkauth.ErrInvalidToken
	}
	service, err := mcp.NewService(nil, nil, mcp.Config{
		Resource: "https://agenda.example/mcp", Issuer: "https://sso.example", StateKey: []byte("k"),
		Verifier: refuse, Pending: mcptest.NewPending(),
	})
	if err != nil {
		t.Fatalf("mcp.NewService: %v", err)
	}
	return service
}
