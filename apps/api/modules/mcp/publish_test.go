package mcp

import (
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus/bustest"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestToolWritesQueueOneRowEach checks the MCP create, update and confirmed
// delete each queue exactly one change, through the events service's
// transaction.
func TestToolWritesQueueOneRowEach(t *testing.T) {
	f := newFixture(t)
	var created writeOutput
	bustest.AssertQueued(t, f.db, func() error {
		mcptest.Decode(t, mcptest.Call(t, f.session(t), toolCreateEvent, map[string]any{
			"title": "Lunch", "start": "2026-11-12T12:00:00+01:00", "end": "2026-11-12T13:30:00+01:00", "timezone": "Europe/Paris",
		}), &created)
		return nil
	}, "calendar_event.created")
	bustest.AssertQueued(t, f.db, func() error {
		mcptest.Call(t, f.session(t), toolUpdateEvent, map[string]any{"event_id": created.Event.EventID, "start": "2026-11-13T09:00:00+01:00"})
		return nil
	}, "calendar_event.updated")

	eventID, id := pendingDelete(t, f)
	bustest.AssertQueued(t, f.db, func() error {
		if got := onPage(f, "confirm", id, ""); got.Code != 200 || f.exists(eventID) {
			t.Fatalf("want the event deleted, got %d %s", got.Code, got.Body)
		}
		return nil
	}, "calendar_event.deleted")
}
