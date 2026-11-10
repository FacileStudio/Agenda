package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
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

// onPage calls a confirm page handler as the signed-in person, or as a
// delegated token when actor is set.
func onPage(f *fixture, handler http.HandlerFunc, id, actor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/mcp/pending/"+id+"/confirm", nil)
	ctx := authcontext.WithIdentity(req.Context(), authcontext.Identity{UserID: itoa(f.userID)})
	ctx = porte.WithIdentity(ctx, porte.Identity{UserID: f.userID, Actor: actor})
	routes := chi.NewRouteContext()
	routes.URLParams.Add("id", id)
	recorder := httptest.NewRecorder()
	handler(recorder, req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, routes)))
	return recorder
}

// TestConfirmPageDeletesOnce checks the person confirms in Agenda, the event
// goes once, a second confirmation finds nothing, and an agent cannot
// confirm its own request.
func TestConfirmPageDeletesOnce(t *testing.T) {
	f := newFixture(t)
	eventID, id := pendingDelete(t, f)
	if got := onPage(f, f.service.handleConfirm, id, "bulle"); got.Code != http.StatusForbidden || !f.exists(eventID) {
		t.Fatalf("a delegated token must not confirm, got %d", got.Code)
	}
	list := onPage(f, f.service.handleList, "", "")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"state":"ready"`) {
		t.Fatalf("want one ready action on the page, got %d %s", list.Code, list.Body)
	}
	if got := onPage(f, f.service.handleConfirm, id, ""); got.Code != http.StatusOK || f.exists(eventID) {
		t.Fatalf("want the event deleted, got %d %s", got.Code, got.Body)
	}
	if again := onPage(f, f.service.handleConfirm, id, ""); again.Code != http.StatusNotFound {
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
	if got := onPage(f, f.service.handleConfirm, id, ""); got.Code != http.StatusPreconditionFailed || !f.exists(eventID) {
		t.Fatalf("want 412 and the event kept, got %d", got.Code)
	}
}
