package calendars

import (
	"context"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var testDBCfg = testdb.Config{Prefix: "agenda_calendars_test", Migrate: schemas.Migrate}

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	db, err := testdb.Open(url, testDBCfg)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	return db
}

func names(cals []CalendarResponse) map[string]bool {
	out := map[string]bool{}
	for _, c := range cals {
		out[c.Name] = true
	}
	return out
}

func TestPersonalListingHidesOnlyCalendarsOfSpacesTheUserBelongsTo(t *testing.T) {
	db := openTestDatabase(t)
	svc := NewService(db)
	ctx := context.Background()
	alice := schemas.User{Email: "alice@example.com", Name: "Alice"}
	bob := schemas.User{Email: "bob@example.com", Name: "Bob"}
	db.Create(&alice)
	db.Create(&bob)
	space := schemas.Space{Name: "Studio"}
	db.Create(&space)
	db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: alice.ID, Role: "owner"})
	zero := int64(0)
	gone := int64(9999)
	db.Create(&schemas.Calendar{OwnerID: alice.ID, Slug: "legacy", Name: "Legacy"})
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &space.ID, Slug: "in-space", Name: "InSpace"})
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &zero, Slug: "zero", Name: "Zero"})
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &gone, Slug: "gone", Name: "Gone"})
	shared := schemas.Calendar{OwnerID: bob.ID, SpaceID: &space.ID, Slug: "bob-space", Name: "BobSpace"}
	db.Create(&shared)
	db.Create(&schemas.CalendarMember{CalendarID: shared.ID, UserID: alice.ID, Role: "writer"})

	personal, err := svc.ListCalendars(ctx, alice.ID, nil)
	if err != nil {
		t.Fatalf("list personal: %v", err)
	}
	got := names(personal)
	for _, want := range []string{"Legacy", "Zero", "Gone", "Personal"} {
		if !got[want] {
			t.Errorf("personal listing misses %s: %v", want, got)
		}
	}
	if got["InSpace"] || got["BobSpace"] {
		t.Errorf("personal listing leaks space calendars: %v", got)
	}

	inSpace, err := svc.ListCalendars(ctx, alice.ID, &space.ID)
	if err != nil {
		t.Fatalf("list space: %v", err)
	}
	if got := names(inSpace); !got["InSpace"] || !got["BobSpace"] || len(got) != 2 {
		t.Errorf("space listing = %v", got)
	}
}

func TestCreateCalendarTreatsSpaceZeroAsPersonal(t *testing.T) {
	db := openTestDatabase(t)
	svc := NewService(db)
	alice := schemas.User{Email: "alice@example.com", Name: "Alice"}
	db.Create(&alice)
	zero := int64(0)
	resp, err := svc.CreateCalendar(context.Background(), alice.ID, &CreateCalendarRequest{Name: "Zero", SpaceID: &zero})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var cal schemas.Calendar
	db.First(&cal, resp.ID)
	if cal.SpaceID != nil {
		t.Fatalf("space_id = %d, want NULL", *cal.SpaceID)
	}
}

func TestDeleteCalendarRemovesItsEvents(t *testing.T) {
	db := openTestDatabase(t)
	svc := NewService(db)
	alice := schemas.User{Email: "alice@example.com", Name: "Alice"}
	db.Create(&alice)
	id, err := svc.EnsureCalendar(context.Background(), alice.ID, "opus-tasks-1", "Opus tasks", "#fff")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	evt := schemas.Event{CalendarID: id, UID: "u1", Title: "Task", Source: "opus", ExternalID: "t1"}
	db.Create(&evt)
	db.Create(&schemas.EventAttendee{EventID: evt.ID, Email: "x@example.com"})

	if err := svc.DeleteCalendar(context.Background(), alice.ID, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var events, attendees int64
	db.Model(&schemas.Event{}).Count(&events)
	db.Model(&schemas.EventAttendee{}).Count(&attendees)
	if events != 0 || attendees != 0 {
		t.Fatalf("orphans left: %d events, %d attendees", events, attendees)
	}
}
