package pool

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
)

func placedTask(facileID string, extra map[string]any) json.RawMessage {
	payload := map[string]any{"facile_id": facileID, "title": "Ship it", "due_date": "2026-11-04", "actor_email": "alice@example.com"}
	for key, value := range extra {
		payload[key] = value
	}
	raw, _ := json.Marshal(map[string]any{
		"version": 1, "app": "Opus", "facile_id": facileID, "payload": payload,
		"idempotency_key": facileID + ":" + strconv.FormatInt(1_700_000_000_000+clock.Add(1), 10),
	})
	return raw
}

func (f *fixture) spaceCalendar(t *testing.T, registreID string) int64 {
	t.Helper()
	space := schemas.Space{Name: "Studio", RegistreSpaceID: &registreID}
	if err := f.db.Create(&space).Error; err != nil {
		t.Fatalf("create space: %v", err)
	}
	if err := schemas.SyncSpaceCalendars(f.db); err != nil {
		t.Fatalf("sync space calendars: %v", err)
	}
	var cal schemas.Calendar
	if err := f.db.Where("space_id = ?", space.ID).First(&cal).Error; err != nil {
		t.Fatalf("space calendar: %v", err)
	}
	return cal.ID
}

func (f *fixture) mirroredTask(t *testing.T, facileID string) schemas.Event {
	t.Helper()
	var evt schemas.Event
	if err := f.db.Where("source = ? AND external_id = ?", SourceOpus, facileID).First(&evt).Error; err != nil {
		t.Fatalf("mirrored %s: %v", facileID, err)
	}
	return evt
}

func TestSpaceTaskLandsInTheSpaceCalendar(t *testing.T) {
	f := newFixture(t)
	spaceCal := f.spaceCalendar(t, "reg-space-1")
	link := "https://opus.facile.studio/projects/p1?task=t1"

	f.apply(t, "task.created", placedTask("t1", map[string]any{"registre_space_id": "reg-space-1", "url": link}))
	f.apply(t, "task.created", placedTask("t2", map[string]any{"registre_space_id": "reg-space-1", "actor_email": "nobody@example.com"}))

	evt := f.mirroredTask(t, "t1")
	if evt.CalendarID != spaceCal || evt.SpaceID == nil || evt.ExternalURL != link {
		t.Fatalf("space task = calendar %d space %v url %q, want calendar %d and the link", evt.CalendarID, evt.SpaceID, evt.ExternalURL, spaceCal)
	}
	if f.mirroredTask(t, "t2").CalendarID != spaceCal {
		t.Fatal("a space task without an Agenda assignee left the space calendar")
	}
}

func TestPersonalOrUnknownSpaceTaskLandsInThePersonalCalendar(t *testing.T) {
	f := newFixture(t)
	spaceCal := f.spaceCalendar(t, "reg-space-1")

	f.apply(t, "task.created", placedTask("t1", map[string]any{"registre_space_id": "reg-space-1", "personal": true}))
	f.apply(t, "task.created", placedTask("t2", map[string]any{"registre_space_id": "reg-unknown", "url": "javascript:alert(1)"}))

	for _, id := range []string{"t1", "t2"} {
		evt := f.mirroredTask(t, id)
		cal := f.calendarOf(t, evt.CalendarID)
		if evt.CalendarID == spaceCal || !cal.IsPersonal || cal.OwnerID != f.alice.ID {
			t.Fatalf("%s landed in %+v, want Alice's personal calendar", id, cal)
		}
	}
	if url := f.mirroredTask(t, "t2").ExternalURL; url != "" {
		t.Fatalf("unsafe link kept: %q", url)
	}
}

func TestTaskMovesWhenItJoinsASpace(t *testing.T) {
	f := newFixture(t)
	spaceCal := f.spaceCalendar(t, "reg-space-1")
	f.apply(t, "task.created", placedTask("t1", nil))
	f.apply(t, "task.updated", placedTask("t1", map[string]any{"registre_space_id": "reg-space-1"}))

	if got := f.mirroredTask(t, "t1").CalendarID; got != spaceCal {
		t.Fatalf("calendar = %d, want the space calendar %d", got, spaceCal)
	}
}
