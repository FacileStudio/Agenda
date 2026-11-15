package schemas

import (
	"testing"

	"gorm.io/gorm"
)

// seedOpusTaskCalendars gives Alice a personal calendar and an Opus tasks one
// with a task, and Bob, who has no personal calendar, a shared Opus tasks one
// with a task and an empty one.
func seedOpusTaskCalendars(db *gorm.DB) (alice, bob User, personal Calendar) {
	alice = User{Email: "alice@example.com", Name: "Alice"}
	bob = User{Email: "bob@example.com", Name: "Bob"}
	db.Create(&alice)
	db.Create(&bob)
	personal = Calendar{OwnerID: alice.ID, Slug: "personal-1", Name: "Personal", IsPersonal: true, SyncToken: "old"}
	db.Create(&personal)
	aliceTasks := Calendar{OwnerID: alice.ID, Slug: "opus-tasks-1", Name: "Opus tasks"}
	bobTasks := Calendar{OwnerID: bob.ID, Slug: "opus-tasks-2", Name: "Opus tasks"}
	for _, cal := range []*Calendar{&aliceTasks, &bobTasks, {OwnerID: bob.ID, Slug: "opus-tasks-x", Name: "Opus tasks"}} {
		db.Create(cal)
	}
	db.Create(&Event{CalendarID: aliceTasks.ID, UID: "a1", Title: "Ship", Source: "opus", ExternalID: "t1"})
	db.Create(&Event{CalendarID: bobTasks.ID, UID: "b1", Title: "Review", Source: "opus", ExternalID: "t2"})
	db.Create(&CalendarMember{CalendarID: bobTasks.ID, UserID: alice.ID, Role: "reader"})
	return alice, bob, personal
}

func TestOpusTaskCalendarsMoveIntoThePersonalCalendar(t *testing.T) {
	db := openTestDatabase(t)
	_, bob, personal := seedOpusTaskCalendars(db)

	for range 2 {
		if err := moveOpusTaskCalendars(db); err != nil {
			t.Fatalf("move: %v", err)
		}
	}

	var left int64
	db.Model(&Calendar{}).Where("slug LIKE ?", OpusTasksSlugPattern).Count(&left)
	if left != 0 {
		t.Fatalf("%d Opus tasks calendars left", left)
	}
	var ship, review Event
	db.Where("external_id = ?", "t1").First(&ship)
	db.Where("external_id = ?", "t2").First(&review)
	if ship.CalendarID != personal.ID {
		t.Fatalf("Alice's task is in calendar %d, want her personal calendar %d", ship.CalendarID, personal.ID)
	}
	var bobCal Calendar
	db.First(&bobCal, review.CalendarID)
	if bobCal.OwnerID != bob.ID || !bobCal.IsPersonal || bobCal.SpaceID != nil {
		t.Fatalf("Bob's task is in %+v, want a new personal calendar of his", bobCal)
	}
	assertOpusCleanup(t, db, personal.ID)
}

// assertOpusCleanup checks the receiving calendar was bumped and no share of a
// deleted calendar is left.
func assertOpusCleanup(t *testing.T, db *gorm.DB, personalID int64) {
	t.Helper()
	var reloaded Calendar
	db.First(&reloaded, personalID)
	if reloaded.SyncToken == "old" {
		t.Fatal("the receiving calendar's sync token was not bumped")
	}
	var shares int64
	db.Model(&CalendarMember{}).Count(&shares)
	if shares != 0 {
		t.Fatalf("%d shares of a deleted calendar left", shares)
	}
}
