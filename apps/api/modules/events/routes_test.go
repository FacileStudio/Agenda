package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var testDBCfg = testdb.Config{Prefix: "agenda_events_test", Migrate: schemas.Migrate}

type harness struct {
	t      *testing.T
	db     *gorm.DB
	router chi.Router
	userID string
	calID  int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	db, err := testdb.Open(url, testDBCfg)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	h := &harness{t: t, db: db}
	owner := schemas.User{Email: "owner@example.com", Name: "Owner"}
	db.Create(&owner)
	cal := schemas.Calendar{OwnerID: owner.ID, Slug: "main", Name: "Main"}
	db.Create(&cal)
	h.calID = cal.ID
	h.userID = itoa(owner.ID)
	h.router = chiWith(newController(NewService(db)), &h.userID)
	return h
}

func chiWith(c *controller, userID *string) chi.Router {
	router := chi.NewRouter()
	mountRoutes(router, c, asUser(userID))
	return router
}

func asUser(userID *string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := authcontext.WithIdentity(r.Context(), authcontext.Identity{UserID: *userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *harness) do(method, path string, body any) (int, EventResponse, []EventResponse) {
	h.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	var one EventResponse
	var many []EventResponse
	if json.Unmarshal(rec.Body.Bytes(), &one) != nil {
		json.Unmarshal(rec.Body.Bytes(), &many)
	}
	return rec.Code, one, many
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func (h *harness) create(body map[string]any) EventResponse {
	h.t.Helper()
	code, evt, _ := h.do(http.MethodPost, "/calendars/"+itoa(h.calID)+"/events/", body)
	if code != http.StatusCreated {
		h.t.Fatalf("create = %d", code)
	}
	return evt
}

func weekly(extra map[string]any) map[string]any {
	body := map[string]any{
		"title": "Standup", "start_at": "2026-11-02T08:00:00Z", "end_at": "2026-11-02T08:30:00Z",
		"recurrence_rule": "FREQ=DAILY;COUNT=5", "timezone": "Europe/Paris",
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func (h *harness) list() []string {
	h.t.Helper()
	path := "/calendars/" + itoa(h.calID) + "/events/?from=2026-11-01T00:00:00Z&to=2026-11-30T00:00:00Z"
	_, _, many := h.do(http.MethodGet, path, nil)
	var out []string
	for _, e := range many {
		out = append(out, e.StartAt.UTC().Format("02T15:04")+" "+e.Title)
	}
	return out
}

func (h *harness) raw(id int64) string {
	var evt schemas.Event
	h.db.First(&evt, id)
	return evt.RawICS
}

func mustEqual(t *testing.T, got []string, want string) {
	t.Helper()
	if strings.Join(got, "|") != want {
		t.Fatalf("got %q, want %q", strings.Join(got, "|"), want)
	}
}
