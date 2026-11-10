package mcp

import (
	"fmt"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestConformance runs tronc's conformance kit on the endpoint, with
// delete event as the sensitive tool.
func TestConformance(t *testing.T) {
	f := newFixture(t)
	mcptest.Run(t, mcptest.Config{
		URL:     f.resource,
		Token:   f.token,
		Confirm: []mcptest.ConfirmCase{{Tool: toolDeleteEvent, Setup: f.freshEvent}},
	})
}

// freshEvent adds an event and reports 1 once it is gone.
func (f *fixture) freshEvent(t testing.TB) (map[string]any, func() int) {
	start := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	e := f.event(t, events.CreateEventRequest{Title: fmt.Sprintf("Review %s", t.Name()), StartAt: start, EndAt: start.Add(time.Hour)})
	return map[string]any{"event_id": e.ID}, func() int {
		if f.exists(e.ID) {
			return 0
		}
		return 1
	}
}
