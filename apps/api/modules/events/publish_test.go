package events

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	"github.com/FacileStudio/Agenda/apps/api/internal/bus/bustest"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
)

func publishing(t *testing.T) (*harness, *Service, int64) {
	t.Helper()
	h := newHarness(t)
	var owner schemas.User
	h.db.Where("email = ?", "owner@example.com").First(&owner)
	return h, NewService(h.db).WithOutbox(bus.New(true)), owner.ID
}

func sampleEvent(title string) *CreateEventRequest {
	start := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	return &CreateEventRequest{
		Title: title, Description: "private notes", Location: "12 rue privée",
		StartAt: start, EndAt: start.Add(time.Hour),
	}
}

func (h *harness) events() int64 {
	return bustest.Count(h.t, h.db, "events", "calendar_id = ?", h.calID)
}

func TestEventWritesQueueOneRowEach(t *testing.T) {
	h, svc, userID := publishing(t)
	ctx := context.Background()
	var created *EventResponse

	bustest.AssertQueued(t, h.db, func() (err error) {
		created, err = svc.CreateEvent(ctx, userID, h.calID, sampleEvent("Standup"))
		return err
	}, "calendar_event.created")

	bustest.AssertQueued(t, h.db, func() error {
		_, err := svc.UpdateEvent(ctx, userID, created.ID, &UpdateEventRequest{
			Title: "Standup moved", StartAt: created.StartAt.Add(time.Hour), EndAt: created.EndAt.Add(time.Hour),
		})
		return err
	}, "calendar_event.updated")

	bustest.AssertQueued(t, h.db, func() error {
		return svc.DeleteEvent(ctx, userID, created.ID)
	}, "calendar_event.deleted")
}

func TestOccurrenceEditQueuesAnUpdate(t *testing.T) {
	h, svc, userID := publishing(t)
	ctx := context.Background()
	req := sampleEvent("Daily")
	req.RecurrenceRule = "FREQ=DAILY;COUNT=3"
	created, err := svc.CreateEvent(ctx, userID, h.calID, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	bustest.AssertQueued(t, h.db, func() error {
		_, err := svc.CancelOccurrence(ctx, userID, created.ID, "2026-11-13")
		return err
	}, "calendar_event.updated")
}

func TestEventWritesRollBackWithTheirEnqueue(t *testing.T) {
	h, svc, userID := publishing(t)
	ctx := context.Background()
	kept, err := svc.CreateEvent(ctx, userID, h.calID, sampleEvent("Kept"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	bustest.AssertRollsBack(t, h.db, func() error {
		_, err := svc.CreateEvent(ctx, userID, h.calID, sampleEvent("Lost"))
		return err
	}, func() bool { return h.events() == 1 })

	bustest.AssertRollsBack(t, h.db, func() error {
		_, err := svc.UpdateEvent(ctx, userID, kept.ID, &UpdateEventRequest{Title: "Renamed", StartAt: kept.StartAt, EndAt: kept.EndAt})
		return err
	}, func() bool { return bustest.Count(t, h.db, "events", "id = ? AND title = ?", kept.ID, "Kept") == 1 })

	bustest.AssertRollsBack(t, h.db, func() error {
		return svc.DeleteEvent(ctx, userID, kept.ID)
	}, func() bool { return h.events() == 1 })
}

func TestMirroredAndUnpublishedWritesQueueNothing(t *testing.T) {
	h, svc, userID := publishing(t)
	ctx := context.Background()
	bustest.AssertQueued(t, h.db, func() error {
		return svc.UpsertExternal(ctx, h.calID, "opus", "task-1", sampleEvent("Opus task"))
	})
	bustest.AssertQueued(t, h.db, func() error {
		_, err := NewService(h.db).CreateEvent(ctx, userID, h.calID, sampleEvent("Off"))
		return err
	})
}

// TestEventPayloadCarriesOnlyTheTypesFields checks a queued event decodes
// into envelope.Event[envelope.CalendarEvent] with no unknown field, and that
// neither the description nor the location reaches the bus.
func TestEventPayloadCarriesOnlyTheTypesFields(t *testing.T) {
	h, svc, userID := publishing(t)
	created, err := svc.CreateEvent(context.Background(), userID, h.calID, sampleEvent("Standup"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rows := bustest.Rows(t, h.db)
	if len(rows) != 1 {
		t.Fatalf("queued %d rows, want 1", len(rows))
	}
	for _, private := range []string{"private notes", "rue priv"} {
		if strings.Contains(string(rows[0].Payload), private) {
			t.Fatalf("the payload carries %q: %s", private, rows[0].Payload)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(rows[0].Payload))
	decoder.DisallowUnknownFields()
	var event envelope.Event[envelope.CalendarEvent]
	if err := decoder.Decode(&event); err != nil {
		t.Fatalf("the payload has a field beyond the type's: %v\n%s", err, rows[0].Payload)
	}
	want := envelope.CalendarEvent{
		FacileID: itoa(created.ID), CalendarFacileID: itoa(h.calID), Title: "Standup",
		StartsAt: "2026-11-12T09:00:00Z", EndsAt: "2026-11-12T10:00:00Z", ActorEmail: "owner@example.com",
	}
	if event.Payload != want || event.App != envelope.AppAgenda || event.Object != envelope.ObjectCalendarEvent ||
		event.Action != envelope.ActionCreated || event.FacileID != want.FacileID || event.IdempotencyKey == "" {
		t.Fatalf("event %+v, want payload %+v", event, want)
	}
}
