package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"

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
	{http.MethodPost, "/api/auth/register"},
	{http.MethodPost, "/api/auth/login"},
	{http.MethodPost, "/api/auth/logout"},
	{http.MethodPost, "/api/auth/sync-profile"},
	{http.MethodGet, "/api/auth/oidc"},
	{http.MethodGet, "/api/auth/oidc/callback"},
	{http.MethodGet, "/api/users/"},
	{http.MethodGet, "/api/users/me"},
	{http.MethodPatch, "/api/users/me"},
	{http.MethodPost, "/api/users/me/avatar"},
	{http.MethodDelete, "/api/users/me/avatar"},
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
	{http.MethodGet, "/api/spaces/"},
	{http.MethodPost, "/api/spaces/"},
	{http.MethodGet, "/api/spaces/1/"},
	{http.MethodPut, "/api/spaces/1/"},
	{http.MethodDelete, "/api/spaces/1/"},
	{http.MethodPost, "/api/spaces/1/leave"},
	{http.MethodGet, "/api/spaces/1/members"},
	{http.MethodPost, "/api/spaces/1/members"},
	{http.MethodDelete, "/api/spaces/1/members/2"},
	{http.MethodPut, "/api/spaces/1/members/2/role"},
	{http.MethodGet, "/api/settings/"},
	{http.MethodPut, "/api/settings/"},
	{http.MethodGet, "/api/health"},
	{http.MethodGet, "/api/ready"},
	{http.MethodGet, "/health"},
	{http.MethodGet, "/ready"},
	{http.MethodGet, "/files/avatars/user-1.png"},
	{http.MethodGet, "/.well-known/caldav"},
	{http.MethodGet, "/dav/"},
	{http.MethodGet, "/auth/oidc/callback"},
}

// stubIssuer serves just enough OpenID discovery for newOIDCHandler to succeed,
// so the router under test carries the /auth/oidc routes SSO_ONLY depends on.
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

func testRouter(t *testing.T) chi.Router {
	t.Helper()
	router := chi.NewRouter()
	mountRoutes(router, mounts{env: env.Config{
		StorageDir: t.TempDir(),
		OIDC: &env.OIDCConfig{
			Issuer:       stubIssuer(t),
			ClientID:     "agenda",
			ClientSecret: "secret",
			RedirectURL:  "https://agenda.example/auth/oidc/callback",
		},
	}})
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
