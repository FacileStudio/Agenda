package ideas

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/plans"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const (
	testIssuer = "https://sso.test"
	bootstrap  = "docker compose up -d db   # or any PostgreSQL 16"
)

var testDBCfg = testdb.Config{Prefix: "agenda_ideas_test", Migrate: schemas.Migrate}

type stubAuth struct{}

func (stubAuth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.Header.Get("X-Test-User"), 10, 64)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		ctx := porte.WithIdentity(r.Context(), porte.Identity{UserID: id})
		next.ServeHTTP(w, r.WithContext(kit.WithIdentity(ctx, kit.Identity{UserID: id})))
	})
}

type harness struct {
	router chi.Router
	bulle  *httptest.Server
	spaces chan string
	calls  atomic.Int32
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(bootstrap)
		t.Skip(testdb.SkipReason(bootstrap))
	}
	db, err := testdb.Open(url, testDBCfg)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	return db
}

func newHarness(t *testing.T, db *gorm.DB, gate Plans) *harness {
	t.Helper()
	h := &harness{spaces: make(chan string, 8)}
	h.bulle = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.calls.Add(1)
		h.spaces <- r.URL.Query().Get("space_id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ideas":[],"generating":false,"last_batch_at":null}`)
	}))
	t.Cleanup(h.bulle.Close)
	service := NewService(db, Config{
		URL:        h.bulle.URL,
		UpgradeURL: testIssuer + "/spaces",
		Token:      func(context.Context) (string, error) { return "token", nil },
		Plans:      gate,
	})
	h.router = chi.NewRouter()
	h.router.Route("/api", func(api chi.Router) {
		if err := RegisterRoutes(api, service, stubAuth{}); err != nil {
			t.Fatal(err)
		}
	})
	return h
}

func (h *harness) do(method, path, user string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader("{}"))
	if user != "" {
		request.Header.Set("X-Test-User", user)
	}
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func seedSpace(t *testing.T, db *gorm.DB) (member int64, other int64, spaceID int64) {
	t.Helper()
	member = newUser(t, db, "member@test")
	other = newUser(t, db, "other@test")
	registre := "spc_registre"
	space := schemas.Space{Name: "Team", RegistreSpaceID: &registre}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: member, Role: "member"}).Error; err != nil {
		t.Fatal(err)
	}
	return member, other, space.ID
}

func newUser(t *testing.T, db *gorm.DB, email string) int64 {
	t.Helper()
	user := schemas.User{Email: email, Name: email}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user.ID
}

func TestIdeasAreBehindAuth(t *testing.T) {
	h := newHarness(t, nil, nil)
	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/ideas"},
		{http.MethodPost, "/api/ideas/generate"},
		{http.MethodPost, "/api/ideas/i1/run"},
		{http.MethodGet, "/api/ideas/runs/r1"},
		{http.MethodPost, "/api/ideas/runs/r1/approvals/c1"},
	}
	for _, c := range cases {
		if code := h.do(c.method, c.path, "").Code; code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session = %d, want 401", c.method, c.path, code)
		}
	}
	if h.calls.Load() != 0 {
		t.Fatalf("Bulle was called %d times without a session", h.calls.Load())
	}
}

func TestIdeasPassTheRegistreSpace(t *testing.T) {
	db := openDB(t)
	member, other, spaceID := seedSpace(t, db)
	h := newHarness(t, db, nil)
	path := "/api/ideas?space_id=" + strconv.FormatInt(spaceID, 10)

	if code := h.do(http.MethodGet, path, strconv.FormatInt(member, 10)).Code; code != http.StatusOK {
		t.Fatalf("member space = %d, want 200", code)
	}
	if got := <-h.spaces; got != "spc_registre" {
		t.Errorf("Bulle got space %q, want spc_registre", got)
	}
	if code := h.do(http.MethodGet, "/api/ideas", strconv.FormatInt(member, 10)).Code; code != http.StatusOK {
		t.Fatalf("personal = %d, want 200", code)
	}
	if got := <-h.spaces; got != "" {
		t.Errorf("personal sent space %q, want none", got)
	}
	if code := h.do(http.MethodGet, path, strconv.FormatInt(other, 10)).Code; code != http.StatusNotFound {
		t.Errorf("non-member = %d, want 404", code)
	}
	if h.calls.Load() != 2 {
		t.Errorf("Bulle calls = %d, want 2", h.calls.Load())
	}
}

func TestAFreePlanGetsNoAIAndNoBulleCall(t *testing.T) {
	db := openDB(t)
	free, paid, _ := seedSpace(t, db)
	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, 'sub-free'), (?, ?, 'sub-paid')`,
		free, testIssuer, paid, testIssuer).Error; err != nil {
		t.Fatal(err)
	}
	gate := plans.New(db, testIssuer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	snapshot := directory.Snapshot{Version: 1, EntitlementsRevision: 1, People: []directory.Person{
		{Subject: "sub-paid", Entitlements: map[string]json.RawMessage{plans.AIAccess: json.RawMessage("true")}},
	}}
	if err := gate.Apply(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, db, gate)

	list := h.do(http.MethodGet, "/api/ideas", strconv.FormatInt(free, 10))
	var page struct {
		AIAccess   bool   `json:"ai_access"`
		UpgradeURL string `json:"upgrade_url"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if list.Code != http.StatusOK || page.AIAccess || page.UpgradeURL != testIssuer+"/spaces" {
		t.Errorf("free list = %d %+v, want 200, ai_access false and the upgrade url", list.Code, page)
	}
	if code := h.do(http.MethodPost, "/api/ideas/generate", strconv.FormatInt(free, 10)).Code; code != http.StatusPaymentRequired {
		t.Errorf("free generate = %d, want 402", code)
	}
	if h.calls.Load() != 0 {
		t.Fatalf("Bulle was called %d times for a free plan", h.calls.Load())
	}
	if code := h.do(http.MethodPost, "/api/ideas/generate", strconv.FormatInt(paid, 10)).Code; code != http.StatusAccepted {
		t.Errorf("paid generate = %d, want 202", code)
	}
	if h.calls.Load() != 1 {
		t.Errorf("Bulle calls for a paid plan = %d, want 1", h.calls.Load())
	}
}

func TestASpacePlanDecidesForThatSpace(t *testing.T) {
	db := openDB(t)
	member, _, spaceID := seedSpace(t, db)
	gate := plans.New(db, testIssuer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	snapshot := directory.Snapshot{Version: 1, EntitlementsRevision: 1, Spaces: []directory.Space{
		{ID: "spc_registre", Entitlements: map[string]json.RawMessage{plans.AIAccess: json.RawMessage("true")}},
	}}
	if err := gate.Apply(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, db, gate)
	user := strconv.FormatInt(member, 10)

	if code := h.do(http.MethodPost, "/api/ideas/generate?space_id="+strconv.FormatInt(spaceID, 10), user).Code; code != http.StatusAccepted {
		t.Errorf("paid space generate = %d, want 202", code)
	}
	if code := h.do(http.MethodPost, "/api/ideas/generate", user).Code; code != http.StatusPaymentRequired {
		t.Errorf("personal generate on a free person = %d, want 402", code)
	}
}

func TestEntitlementsSurviveARestart(t *testing.T) {
	db := openDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	first := plans.New(db, testIssuer, logger)
	if err := first.Apply(context.Background(), directory.Snapshot{Version: 1, EntitlementsRevision: 4}); err != nil {
		t.Fatal(err)
	}
	second := plans.New(db, testIssuer, logger)
	if second.Active() {
		t.Fatal("a fresh gate must start inactive")
	}
	if err := second.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	allowed, err := second.HasAI(context.Background(), 9, "")
	if err != nil || allowed {
		t.Fatalf("a user with no entitlement = %v, %v, want false after a restart", allowed, err)
	}
}
