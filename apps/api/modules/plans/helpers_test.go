package plans

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte/kit"
	tronc "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var testDBCfg = testdb.Config{Prefix: "agenda_plans_test", Migrate: schemas.Migrate}

type harness struct {
	t        *testing.T
	db       *gorm.DB
	events   *events.Service
	service  *Service
	owner    schemas.User
	other    schemas.User
	calendar schemas.Calendar
	foreign  schemas.Calendar
	now      time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	url, ok := testdb.URL()
	if !ok {
		t.Skip(testdb.SkipReason("docker compose up -d db   # or any PostgreSQL 16"))
	}
	db, err := testdb.Open(url, testDBCfg)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	h := &harness{t: t, db: db, now: clock}
	h.events = events.NewService(db)
	h.service = NewService(db, h.events)
	h.service.now = func() time.Time { return h.now }
	h.owner = schemas.User{Email: "owner@example.test", Name: "Owner"}
	h.other = schemas.User{Email: "other@example.test", Name: "Other"}
	for _, u := range []*schemas.User{&h.owner, &h.other} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	h.calendar = schemas.Calendar{OwnerID: h.owner.ID, Slug: "personal", Name: "Personal", IsPersonal: true}
	h.foreign = schemas.Calendar{OwnerID: h.other.ID, Slug: "theirs", Name: "Theirs"}
	for _, c := range []*schemas.Calendar{&h.calendar, &h.foreign} {
		if err := db.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	return h
}

func (h *harness) block(startHour, endHour int) BlockInput {
	b := block(startHour, endHour)
	b.CalendarID = h.calendar.ID
	b.TaskRef = "task-" + strconv.Itoa(startHour)
	b.TaskURL = "https://opus.example/tasks/" + strconv.Itoa(startHour)
	b.Reason = "Due Friday"
	return b
}

func (h *harness) propose(blocks ...BlockInput) *PlanResponse {
	h.t.Helper()
	plan, err := h.service.Propose(context.Background(), h.owner.ID, proposal(blocks...))
	if err != nil {
		h.t.Fatalf("propose: %v", err)
	}
	return plan
}

func (h *harness) busy(startHour, endHour int) {
	h.t.Helper()
	b := block(startHour, endHour)
	_, err := h.events.CreateEvent(context.Background(), h.owner.ID, h.calendar.ID, &events.CreateEventRequest{
		Title: "Meeting", StartAt: b.StartAt, EndAt: b.EndAt,
	})
	if err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) eventCount() int64 {
	var n int64
	h.db.Model(&schemas.Event{}).Count(&n)
	return n
}

func status(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return tronc.Status(err)
}

func routerAs(service *Service, userID *int64) chi.Router {
	router := chi.NewRouter()
	guard := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(kit.WithIdentity(r.Context(), kit.Identity{UserID: *userID})))
		})
	}
	mountRoutes(router, &controller{service: service}, guard)
	return router
}

func serve(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
