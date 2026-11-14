package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/porte"
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
