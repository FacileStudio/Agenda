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

func TestEverySpaceOwnsOneCalendarItsMembersShare(t *testing.T) {
	db := openTestDatabase(t)
	svc := NewService(db)
	ctx := context.Background()
	alice := schemas.User{Email: "alice@example.com", Name: "Alice"}
	bob := schemas.User{Email: "bob@example.com", Name: "Bob"}
	carol := schemas.User{Email: "carol@example.com", Name: "Carol"}
	db.Create(&alice)
	db.Create(&bob)
	db.Create(&carol)
	space := schemas.Space{Name: "Studio"}
	db.Create(&space)
	db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: alice.ID, Role: "owner"})
	db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: bob.ID, Role: "member"})
	gone := int64(9999)
	db.Exec("DROP INDEX idx_calendars_one_per_space")
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &space.ID, Slug: "first", Name: "First"})
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &space.ID, Slug: "second", Name: "Second"})
	db.Create(&schemas.Calendar{OwnerID: alice.ID, SpaceID: &gone, Slug: "gone", Name: "Gone"})
	if err := schemas.SyncSpaceCalendars(db); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := schemas.SyncSpaceCalendars(db); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	for user, want := range map[int64]string{alice.ID: "admin", bob.ID: "writer"} {
		cals, err := svc.ListCalendars(ctx, user, &space.ID)
		if err != nil || len(cals) != 1 || cals[0].Name != "Studio" || cals[0].Role != want {
			t.Fatalf("space listing for %d = %+v, %v; want one Studio as %s", user, cals, err, want)
		}
	}
	if cals, _ := svc.ListCalendars(ctx, carol.ID, &space.ID); len(cals) != 0 {
		t.Fatalf("outsider sees the space calendar: %+v", cals)
	}
	got := map[string]bool{}
	none := int64(0)
	personal, _ := svc.ListCalendars(ctx, alice.ID, &none)
	for _, c := range personal {
		got[c.Name] = true
	}
	if !got["Second"] || !got["Gone"] || !got["Personal"] || got["Studio"] {
		t.Fatalf("personal listing = %v", got)
	}
	if all, _ := svc.ListCalendars(ctx, alice.ID, nil); len(all) != len(personal)+1 {
		t.Fatalf("unfiltered listing = %+v, want personal calendars plus Studio", all)
	}
	members, err := svc.ListMembers(ctx, bob.ID, personalOrSpace(t, svc, bob.ID, space.ID))
	if err != nil || len(members) != 2 {
		t.Fatalf("space calendar members = %+v, %v", members, err)
	}
}

func personalOrSpace(t *testing.T, svc *Service, userID, spaceID int64) int64 {
	t.Helper()
	cals, err := svc.ListCalendars(context.Background(), userID, &spaceID)
	if err != nil || len(cals) != 1 {
		t.Fatalf("space calendar: %+v, %v", cals, err)
	}
	return cals[0].ID
}

func TestSpaceCalendarRefusesSharesDeletionAndSiblings(t *testing.T) {
	db := openTestDatabase(t)
	svc := NewService(db)
	ctx := context.Background()
	alice := schemas.User{Email: "alice@example.com", Name: "Alice"}
	db.Create(&alice)
	space := schemas.Space{Name: "Studio"}
	db.Create(&space)
	db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: alice.ID, Role: "owner"})
	if err := schemas.SyncSpaceCalendars(db); err != nil {
		t.Fatalf("sync: %v", err)
	}
	id := personalOrSpace(t, svc, alice.ID, space.ID)
	if err := svc.ShareCalendar(ctx, alice.ID, id, &ShareCalendarRequest{Email: "x@example.com", Role: "reader"}); err == nil {
		t.Error("share on a space calendar succeeded")
	}
	if err := svc.DeleteCalendar(ctx, alice.ID, id); err == nil {
		t.Error("delete of a space calendar succeeded")
	}
	if _, err := svc.CreateCalendar(ctx, alice.ID, &CreateCalendarRequest{Name: "Extra", SpaceID: &space.ID}); err == nil {
		t.Error("second calendar in a space was created")
	}
	updated, err := svc.UpdateCalendar(ctx, alice.ID, id, &UpdateCalendarRequest{Name: "Renamed", Color: "#000000"})
	if err != nil || updated.Name != "Studio" || updated.Color != "#000000" {
		t.Errorf("update = %+v, %v; want name kept, color changed", updated, err)
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
