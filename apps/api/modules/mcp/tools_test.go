package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestListExpandsAndSearches checks a weekly event comes back per
// occurrence, and that query keeps only matching events.
func TestListExpandsAndSearches(t *testing.T) {
	f := newFixture(t)
	start := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	f.event(t, events.CreateEventRequest{Title: "Standup", StartAt: start, EndAt: start.Add(15 * time.Minute), RecurrenceRule: "FREQ=WEEKLY", Timezone: "UTC"})
	f.event(t, events.CreateEventRequest{Title: "Dentist", Location: "Rue de Rivoli", StartAt: start.Add(26 * time.Hour), EndAt: start.Add(27 * time.Hour)})
	var out listOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolListEvents, map[string]any{"from": "2026-11-01", "to": "2026-11-30"}), &out)
	if out.Total != 6 || out.Events[0].Title != "Standup" || out.Events[0].OccurrenceStart == "" {
		t.Fatalf("want 5 standups and a dentist, earliest first, got %+v", out)
	}
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolListEvents, map[string]any{"from": "2026-11-01", "to": "2026-11-30", "query": "rivoli"}), &out)
	if out.Total != 1 || out.Events[0].Title != "Dentist" {
		t.Fatalf("query must find the dentist only, got %+v", out)
	}
}

// TestListPagesWithoutRepeats walks 7 events in pages of 3.
func TestListPagesWithoutRepeats(t *testing.T) {
	f := newFixture(t)
	start := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	for i := range 7 {
		f.event(t, events.CreateEventRequest{Title: "Call", StartAt: start.Add(time.Duration(i) * time.Hour), EndAt: start.Add(time.Duration(i)*time.Hour + 30*time.Minute)})
	}
	seen := map[int64]bool{}
	cursor := ""
	for pages := 0; pages < 5; pages++ {
		var out listOutput
		mcptest.Decode(t, mcptest.Call(t, f.session(t), toolListEvents, map[string]any{"from": "2026-11-02", "to": "2026-11-02", "limit": 3, "cursor": cursor}), &out)
		for _, e := range out.Events {
			if seen[e.EventID] {
				t.Fatalf("event %d came twice", e.EventID)
			}
			seen[e.EventID] = true
		}
		if cursor = out.NextCursor; cursor == "" {
			break
		}
	}
	if len(seen) != 7 {
		t.Fatalf("want 7 events over the pages, got %d", len(seen))
	}
}

// TestCreateThenUpdateKeepsLength checks a create lands in the personal
// calendar and a moved start keeps the event's length.
func TestCreateThenUpdateKeepsLength(t *testing.T) {
	f := newFixture(t)
	var created writeOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolCreateEvent, map[string]any{
		"title": "Lunch", "start": "2026-11-12T12:00:00+01:00", "end": "2026-11-12T13:30:00+01:00", "timezone": "Europe/Paris",
		"attendees": []string{"Client@Example.test"},
	}), &created)
	if created.Event.CalendarID != f.calendarID || created.Event.Start != "2026-11-12T12:00:00+01:00" || len(created.Event.Attendees) != 1 {
		t.Fatalf("want the event in the personal calendar at noon Paris time, got %+v", created.Event)
	}
	var updated writeOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolUpdateEvent, map[string]any{
		"event_id": created.Event.EventID, "start": "2026-11-13T09:00:00+01:00",
	}), &updated)
	if updated.Event.End != "2026-11-13T10:30:00+01:00" || updated.Event.Title != "Lunch" || len(updated.Event.Attendees) != 1 {
		t.Fatalf("a moved start keeps the length and the other fields, got %+v", updated.Event)
	}
}

// TestAllDayEventIsDates checks an all-day create reads back as dates.
func TestAllDayEventIsDates(t *testing.T) {
	f := newFixture(t)
	var created writeOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolCreateEvent, map[string]any{
		"title": "Offsite", "start": "2026-11-20", "end": "2026-11-21", "all_day": true, "timezone": "Europe/Paris",
	}), &created)
	if !created.Event.AllDay || created.Event.Start != "2026-11-20" || created.Event.End != "2026-11-21" {
		t.Fatalf("want a two-day all-day event, got %+v", created.Event)
	}
}

// TestFreeSlotsSkipBusyTime checks the slots go around an event inside
// working hours, and skip the weekend.
func TestFreeSlotsSkipBusyTime(t *testing.T) {
	f := newFixture(t)
	paris, _ := time.LoadLocation("Europe/Paris")
	busy := time.Date(2026, 11, 13, 11, 0, 0, 0, paris)
	f.event(t, events.CreateEventRequest{Title: "Busy", StartAt: busy, EndAt: busy.Add(2 * time.Hour)})
	var out slotsOutput
	mcptest.Decode(t, mcptest.Call(t, f.session(t), toolFreeSlots, map[string]any{
		"from": "2026-11-13", "to": "2026-11-15", "duration_minutes": 60, "timezone": "Europe/Paris",
	}), &out)
	want := []freeSlot{
		{Start: "2026-11-13T09:00:00+01:00", End: "2026-11-13T11:00:00+01:00", Minutes: 120},
		{Start: "2026-11-13T13:00:00+01:00", End: "2026-11-13T18:00:00+01:00", Minutes: 300},
	}
	if len(out.Slots) != 2 || out.Slots[0] != want[0] || out.Slots[1] != want[1] {
		t.Fatalf("want %+v, got %+v", want, out.Slots)
	}
}

// TestReaderCannotBeAskedToDelete checks a reader's delete is refused before
// anyone is asked, and nothing is pending.
func TestReaderCannotBeAskedToDelete(t *testing.T) {
	f := newFixture(t)
	owner := schemas.User{Email: "other@example.test", Name: "Other"}
	f.db.Create(&owner)
	shared := schemas.Calendar{OwnerID: owner.ID, Slug: "team", Name: "Team"}
	f.db.Create(&shared)
	f.db.Create(&schemas.CalendarMember{CalendarID: shared.ID, UserID: f.userID, Role: "reader"})
	start := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	e, err := f.events.CreateEvent(t.Context(), owner.ID, shared.ID, &events.CreateEventRequest{Title: "Theirs", StartAt: start, EndAt: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	result := mcptest.Call(t, f.session(t), toolDeleteEvent, map[string]any{"event_id": e.ID})
	open, _ := f.service.tools.confirm.Open(t.Context(), itoa(f.userID))
	if !result.IsError || !strings.Contains(mcptest.Text(result), "only read") || len(open) != 0 || !f.exists(e.ID) {
		t.Fatalf("want a refusal and nothing pending, got %s and %d pending", mcptest.Text(result), len(open))
	}
}
