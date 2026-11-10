package calendars

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	"github.com/FacileStudio/Agenda/apps/api/internal/bus/bustest"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"gorm.io/gorm"
)

func publishingService(t *testing.T) (*gorm.DB, *Service, int64) {
	t.Helper()
	db := openTestDatabase(t)
	owner := schemas.User{Email: "owner@example.com", Name: "Owner"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	return db, NewService(db).WithOutbox(bus.New(true)), owner.ID
}

func TestCalendarWritesQueueOneRowEach(t *testing.T) {
	db, svc, userID := publishingService(t)
	ctx := context.Background()
	var created *CalendarResponse

	bustest.AssertQueued(t, db, func() error {
		return svc.EnsurePersonalCalendar(ctx, userID)
	}, "calendar.created")
	bustest.AssertQueued(t, db, func() error {
		_, err := svc.EnsureCalendar(ctx, userID, "opus-tasks", "Opus tasks", "#000000")
		return err
	}, "calendar.created")
	bustest.AssertQueued(t, db, func() (err error) {
		created, err = svc.CreateCalendar(ctx, userID, &CreateCalendarRequest{Name: "Work"})
		return err
	}, "calendar.created")
	bustest.AssertQueued(t, db, func() error {
		_, err := svc.UpdateCalendar(ctx, userID, created.ID, &UpdateCalendarRequest{Name: "Work stuff"})
		return err
	}, "calendar.updated")

	if err := db.Create(&schemas.Event{CalendarID: created.ID, UID: "uid-1", Title: "Inside"}).Error; err != nil {
		t.Fatalf("seed event: %v", err)
	}
	bustest.AssertQueued(t, db, func() error {
		return svc.DeleteCalendar(ctx, userID, created.ID)
	}, "calendar_event.deleted", "calendar.deleted")
}

func TestCalendarPayloadNamesTheOwner(t *testing.T) {
	db, svc, userID := publishingService(t)
	if _, err := svc.CreateCalendar(context.Background(), userID, &CreateCalendarRequest{Name: "Work", Description: "private"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	rows := bustest.Rows(t, db)
	var event envelope.Event[envelope.Calendar]
	if err := json.Unmarshal(rows[0].Payload, &event); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if event.Payload.Name != "Work" || event.Payload.OwnerEmail != "owner@example.com" || event.Payload.ActorEmail != "owner@example.com" {
		t.Fatalf("payload %+v", event.Payload)
	}
}

func TestCalendarWritesRollBackWithTheirEnqueue(t *testing.T) {
	db, svc, userID := publishingService(t)
	ctx := context.Background()
	kept, err := svc.CreateCalendar(ctx, userID, &CreateCalendarRequest{Name: "Kept"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	calendars := func() int64 { return bustest.Count(t, db, "calendars", "owner_id = ?", userID) }

	bustest.AssertRollsBack(t, db, func() error {
		_, err := svc.CreateCalendar(ctx, userID, &CreateCalendarRequest{Name: "Lost"})
		return err
	}, func() bool { return calendars() == 1 })
	bustest.AssertRollsBack(t, db, func() error {
		_, err := svc.UpdateCalendar(ctx, userID, kept.ID, &UpdateCalendarRequest{Name: "Renamed"})
		return err
	}, func() bool { return bustest.Count(t, db, "calendars", "id = ? AND name = ?", kept.ID, "Kept") == 1 })
	bustest.AssertRollsBack(t, db, func() error {
		return svc.DeleteCalendar(ctx, userID, kept.ID)
	}, func() bool { return calendars() == 1 })
}
