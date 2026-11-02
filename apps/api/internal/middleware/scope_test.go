package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

type fakeJWT map[string]porte.Identity

func (f fakeJWT) VerifyJWT(_ context.Context, raw string) (porte.Identity, error) {
	identity, ok := f[raw]
	if !ok {
		return porte.Identity{}, http.ErrNoCookie
	}
	return identity, nil
}

type scopedAuth struct {
	*session.Manager
}

func (scopedAuth) IdentityForUser(_ context.Context, userID int64) (string, string, error) {
	return "1", "user@example.com", nil
}

type noSessions struct{ porte.SessionStore }

func scopedRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := env.Config{OIDC: &env.OIDCConfig{
		Issuer:          "https://sso.example.com",
		ClientID:        "agenda",
		ClientSecret:    "secret",
		RedirectURL:     "https://agenda.example.com/api/auth/oidc/callback",
		MachineAudience: "agenda",
		ScopePrefix:     "agenda",
	}}.Porte()
	manager, err := session.New(cfg, session.Deps{Sessions: noSessions{}})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	manager.WithJWT(fakeJWT{
		"a.b.read":  {UserID: 1, Scopes: []string{"agenda.read"}, Actor: "bulle"},
		"a.b.write": {UserID: 1, Scopes: []string{"agenda.write"}},
		"a.b.other": {UserID: 1, Scopes: []string{"courrier.read"}},
		"a.b.none":  {UserID: 1},
	})
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router := chi.NewRouter()
	router.Route("/calendars", func(r chi.Router) {
		r.Use(RequireAuth(scopedAuth{manager}))
		r.Get("/", ok)
		r.Get("/{id}/events", ok)
		r.Post("/{id}/events", ok)
	})
	return router
}

func TestBearerScopesGateAgendaRoutes(t *testing.T) {
	router := scopedRouter(t)
	cases := []struct {
		name   string
		token  string
		method string
		path   string
		want   int
	}{
		{"read lists calendars", "a.b.read", http.MethodGet, "/calendars/", http.StatusOK},
		{"read lists events", "a.b.read", http.MethodGet, "/calendars/1/events?from=today", http.StatusOK},
		{"read cannot create", "a.b.read", http.MethodPost, "/calendars/1/events", http.StatusForbidden},
		{"write can create", "a.b.write", http.MethodPost, "/calendars/1/events", http.StatusOK},
		{"another app's scope is refused", "a.b.other", http.MethodGet, "/calendars/", http.StatusForbidden},
		{"no scope claim is refused", "a.b.none", http.MethodGet, "/calendars/", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("Authorization", "Bearer "+tc.token)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.want {
				t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, recorder.Code, tc.want)
			}
		})
	}
}
