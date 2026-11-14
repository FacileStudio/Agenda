package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/porte"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	"github.com/go-chi/chi/v5"
)

// pendingDelete asks to delete a fresh event from a client that cannot
// confirm, and returns the event and the action id.
func pendingDelete(t *testing.T, f *fixture) (int64, string) {
	t.Helper()
	args, _ := f.freshEvent(t)
	var out actionOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolDeleteEvent, args), &out)
	return args["event_id"].(int64), out.ConfirmURL[strings.LastIndex(out.ConfirmURL, "=")+1:]
}

// onPage calls a confirm page route as the signed-in person, or as a
// delegated token when actor is set. An empty route lists the actions.
func onPage(f *fixture, route, id, actor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if route != "" {
		req = httptest.NewRequest(http.MethodPost, "/"+id+"/"+route, nil)
	}
	return asIdentity(f, req, porte.Identity{UserID: f.userID, SessionID: 1, BrowserSession: true, Actor: actor})
}

// asIdentity serves req on the confirm page as identity.
func asIdentity(f *fixture, req *http.Request, identity porte.Identity) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	f.service.page.Routes(router)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req.WithContext(porte.WithIdentity(req.Context(), identity)))
	return recorder
}

// TestConfirmPageNeedsBrowserSession checks a CLI session and a bearer token
// of the same person cannot confirm: only their own browser session can.
func TestConfirmPageNeedsBrowserSession(t *testing.T) {
	f := newFixture(t)
	eventID, id := pendingDelete(t, f)
	cli := httptest.NewRequest(http.MethodPost, "/"+id+"/confirm", nil)
	if got := asIdentity(f, cli, porte.Identity{UserID: f.userID, SessionID: 1}); got.Code != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a CLI session must not confirm, got %d", got.Code)
	}
	bearer := httptest.NewRequest(http.MethodPost, "/"+id+"/confirm", nil)
	bearer.Header.Set("Authorization", "Bearer token")
	identity := porte.Identity{UserID: f.userID, SessionID: 1, BrowserSession: true}
	if got := asIdentity(f, bearer, identity); got.Code != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a bearer token must not confirm, got %d", got.Code)
	}
}

// TestConfirmPageDeletesOnce checks the person confirms in Agenda, the event
// goes once, a second confirmation finds nothing, and an agent cannot
// confirm its own request.
func TestConfirmPageDeletesOnce(t *testing.T) {
	f := newFixture(t)
	eventID, id := pendingDelete(t, f)
	if got := onPage(f, "confirm", id, "bulle"); got.Code != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a delegated token must not confirm, got %d", got.Code)
	}
	list := onPage(f, "", "", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"state":"ready"`) ||
		!strings.Contains(list.Body.String(), `"event":{"id":`) {
		t.Fatalf("want one ready action on the page, got %d %s", list.Code, list.Body)
	}
	if got := onPage(f, "confirm", id, ""); got.Code != http.StatusOK || f.exists(eventID) {
		t.Fatalf("want the event deleted, got %d %s", got.Code, got.Body)
	}
	if again := onPage(f, "confirm", id, ""); again.Code != http.StatusNotFound {
		t.Fatalf("a confirmation is single use, got %d", again.Code)
	}
}

// TestEditedEventIsNotDeleted checks an edit after the request closes it
// instead of deleting a version the person never saw.
func TestEditedEventIsNotDeleted(t *testing.T) {
	f := newFixture(t)
	eventID, id := pendingDelete(t, f)
	current, _ := f.events.GetEvent(t.Context(), f.userID, eventID)
	if _, err := f.events.UpdateEvent(t.Context(), f.userID, eventID, &events.UpdateEventRequest{
		Title: "Moved", StartAt: current.StartAt, EndAt: current.EndAt,
	}); err != nil {
		t.Fatal(err)
	}
	if got := onPage(f, "confirm", id, ""); got.Code != http.StatusPreconditionFailed || !f.exists(eventID) {
		t.Fatalf("want 412 and the event kept, got %d", got.Code)
	}
}

// TestPendingRoutesThroughRequireAuth mounts the confirm page the way main
// does, behind Agenda's real RequireAuth, and checks a browser cookie needs
// the CSRF header to confirm, and a CLI token sent as that cookie cannot.
func TestPendingRoutesThroughRequireAuth(t *testing.T) {
	f := newFixture(t)
	eventID, id := pendingDelete(t, f)
	sqlDB, err := f.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := session.New(porte.Config{}, session.Deps{Sessions: portepg.New(sqlDB).Sessions()})
	if err != nil {
		t.Fatal(err)
	}
	users, err := auth.NewStore(t.Context(), sqlDB, nil)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	RegisterPendingRoutes(router, f.service, auth.NewService(sessions, users))
	issued := httptest.NewRecorder()
	if _, _, err := sessions.IssueCookie(t.Context(), issued, httptest.NewRequest(http.MethodGet, "/", nil), f.userID); err != nil {
		t.Fatal(err)
	}
	browser := issued.Result().Cookies()[0]
	cli, _, err := sessions.IssueFor(t.Context(), porte.Session{UserID: f.userID})
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path string, cookie *http.Cookie, csrf bool) int {
		req := httptest.NewRequest(method, "/mcp/pending"+path, nil)
		req.AddCookie(cookie)
		if csrf {
			req.Header.Set(porte.CSRFHeaderName, "1")
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder.Code
	}
	confirm := "/" + id + "/confirm"
	if got := send(http.MethodPost, confirm, browser, false); got != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a cookie POST without %s must be refused, got %d", porte.CSRFHeaderName, got)
	}
	if got := send(http.MethodPost, confirm, &http.Cookie{Name: browser.Name, Value: cli}, true); got != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a CLI token sent as the cookie must not confirm, got %d", got)
	}
	if got := send(http.MethodGet, "", browser, false); got != http.StatusOK {
		t.Fatalf("the browser session lists its actions, got %d", got)
	}
	if got := send(http.MethodPost, confirm, browser, true); got != http.StatusOK || f.exists(eventID) {
		t.Fatalf("the browser session with the header confirms, got %d", got)
	}
}
