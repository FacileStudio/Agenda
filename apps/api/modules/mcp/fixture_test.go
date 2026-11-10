package mcp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	"github.com/FacileStudio/tronc/testdb"
	"github.com/go-chi/chi/v5"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// subject is the Registre sub of the test person.
const subject = "person-sub"

// testDBCfg is this package's scratch schema.
var testDBCfg = testdb.Config{Prefix: "agenda_mcp_test", Migrate: schemas.Migrate}

// identities maps the test subject to userID and nothing else.
type identities struct {
	porte.IdentityStore
	userID int64
}

// Find resolves only the test subject.
func (s identities) Find(_ context.Context, provider, sub string) (porte.StoredIdentity, error) {
	if sub != subject {
		return porte.StoredIdentity{}, porte.ErrNotFound
	}
	return porte.StoredIdentity{UserID: s.userID, Provider: provider, Subject: sub}, nil
}

// fixture is a running MCP endpoint over Agenda's real services.
type fixture struct {
	db         *gorm.DB
	srv        *httptest.Server
	issuer     *mcptest.Issuer
	service    *Service
	events     *events.Service
	userID     int64
	calendarID int64
	resource   string
	perMinute  int
}

// openDB opens a clean scratch database, or skips the test without one.
func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		t.Skip(testdb.SkipReason("docker compose up -d db   # or any PostgreSQL 16"))
	}
	db, err := testdb.Open(url, testDBCfg)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	if err := db.Exec("TRUNCATE mcp_actions").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

// newFixture starts the endpoint for one person with a personal calendar.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	return newFixtureOn(t, nil, nil)
}

// newFixtureOn is newFixture on a given listener, nil for any free port,
// with front wrapping the routes when it is set.
func newFixtureOn(t *testing.T, listener net.Listener, front func(*fixture, http.Handler) http.Handler) *fixture {
	t.Helper()
	f := &fixture{db: openDB(t), issuer: mcptest.NewIssuer(t)}
	if listener != nil {
		f.perMinute = 1_000_000
	}
	f.seed(t)
	var handler http.Handler
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	if listener != nil {
		f.srv.Listener = listener
	}
	f.srv.Start()
	t.Cleanup(f.srv.Close)
	f.resource = f.srv.URL + "/mcp"
	f.service = f.build(t)
	router := chi.NewRouter()
	RegisterRoutes(router, f.service)
	handler = router
	if front != nil {
		handler = front(f, router)
	}
	return f
}

// seed creates the person and their personal calendar.
func (f *fixture) seed(t *testing.T) {
	t.Helper()
	user := schemas.User{Email: "owner@example.test", Name: "Owner"}
	if err := f.db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	cal := schemas.Calendar{OwnerID: user.ID, Slug: "personal", Name: "Personal", IsPersonal: true}
	if err := f.db.Create(&cal).Error; err != nil {
		t.Fatal(err)
	}
	f.userID, f.calendarID = user.ID, cal.ID
}

// build makes the MCP service behind the fake Registre.
func (f *fixture) build(t *testing.T) *Service {
	t.Helper()
	verifier, err := mcpauth.Verifier(context.Background(), mcpauth.Config{
		Issuer: f.issuer.URL, Resource: f.resource, ScopePrefix: "agenda", Identities: identities{userID: f.userID},
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := f.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	f.events = events.NewService(f.db)
	service, err := NewService(calendars.NewService(f.db), f.events, Config{
		Resource: f.resource, Issuer: f.issuer.URL, Version: "test", StateKey: []byte("state-key"), Verifier: verifier,
		Pending: mcpkit.PostgresPending{DB: sqlDB}, PerMinute: f.perMinute,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

// token is a token for this endpoint with scopes, or both when none is
// given.
func (f *fixture) token(t testing.TB, scopes ...string) string {
	if len(scopes) == 0 {
		scopes = []string{"agenda.read", "agenda.write"}
	}
	return f.issuer.Token(t, f.resource, subject, scopes...)
}

// session opens a client session with every scope and no elicitation.
func (f *fixture) session(t *testing.T) *sdk.ClientSession {
	t.Helper()
	return mcptest.Connect(t, f.resource, f.token(t), mcptest.ClientOptions{})
}

// event creates an event in the personal calendar through the service.
func (f *fixture) event(t testing.TB, req events.CreateEventRequest) events.EventResponse {
	t.Helper()
	created, err := f.events.CreateEvent(t.Context(), f.userID, f.calendarID, &req)
	if err != nil {
		t.Fatal(err)
	}
	return *created
}

// exists reports whether event id is still there.
func (f *fixture) exists(id int64) bool {
	var count int64
	f.db.Model(&schemas.Event{}).Where("id = ?", id).Count(&count)
	return count == 1
}

// itoa formats an id.
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
