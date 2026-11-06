package spaces

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

func count(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

type erasureFixture struct {
	gone, other int64
	deletions   *Deletions
}

func mustCreate(t *testing.T, db *gorm.DB, row any) {
	t.Helper()
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func seedErasure(t *testing.T, db *gorm.DB) erasureFixture {
	t.Helper()
	gone := ssoUser(t, db, "gone@facile.studio", "sub-gone")
	other := ssoUser(t, db, "other@facile.studio", "sub-other")
	space := spaceWith(t, db, "Studio", map[int64]string{gone: "member", other: "owner"})
	if err := schemas.SyncSpaceCalendars(db); err != nil {
		t.Fatalf("space calendars: %v", err)
	}
	personal := schemas.Calendar{OwnerID: gone, Name: "Perso", IsPersonal: true}
	kept := schemas.Calendar{OwnerID: other, Name: "Autre", IsPersonal: true}
	mustCreate(t, db, &personal)
	mustCreate(t, db, &kept)
	var shared schemas.Calendar
	if err := db.Where("space_id = ?", space).First(&shared).Error; err != nil {
		t.Fatalf("space calendar: %v", err)
	}
	mustCreate(t, db, &schemas.Event{CalendarID: personal.ID, UID: "a", Title: "private", CreatedByID: &gone})
	mustCreate(t, db, &schemas.Event{CalendarID: shared.ID, UID: "b", Title: "team", CreatedByID: &gone})
	theirs := schemas.Event{CalendarID: kept.ID, UID: "c", Title: "mine", CreatedByID: &other}
	mustCreate(t, db, &theirs)
	mustCreate(t, db, &schemas.EventAttendee{EventID: theirs.ID, UserID: &gone, Email: "gone@facile.studio"})
	mustCreate(t, db, &schemas.CalendarMember{CalendarID: kept.ID, UserID: gone, Role: "reader"})
	deletions := NewDeletions(db, directory.Client{Issuer: testIssuer}, nil, t.TempDir(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	return erasureFixture{gone: gone, other: other, deletions: deletions}
}

// A deleted subject loses the account, its personal calendar with the events
// and attendees in it, its shares and its memberships. An event it wrote in a
// space calendar stays, with no author, and other people's data is untouched.
func TestErasePersonDeletesPersonalDataAndKeepsSharedEvents(t *testing.T) {
	db := openTestDatabase(t)
	f := seedErasure(t, db)
	for range 2 {
		if err := f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone"); err != nil {
			t.Fatalf("erase: %v", err)
		}
	}

	checks := map[string]int64{
		`SELECT count(*) FROM users WHERE id = ?`:                 0,
		`SELECT count(*) FROM calendars WHERE owner_id = ?`:       0,
		`SELECT count(*) FROM event_attendees WHERE user_id = ?`:  0,
		`SELECT count(*) FROM calendar_members WHERE user_id = ?`: 0,
		`SELECT count(*) FROM space_members WHERE user_id = ?`:    0,
	}
	for query, want := range checks {
		if got := count(t, db, query, f.gone); got != want {
			t.Errorf("%s = %d, want %d", query, got, want)
		}
	}
	for title, want := range map[string]int64{"private": 0, "team": 1, "mine": 1} {
		if got := count(t, db, `SELECT count(*) FROM events WHERE title = ?`, title); got != want {
			t.Errorf("events titled %q = %d, want %d", title, got, want)
		}
	}
	if n := count(t, db, `SELECT count(*) FROM events WHERE title = 'team' AND created_by_id IS NULL`); n != 1 {
		t.Errorf("the space event kept an author")
	}
	if n := count(t, db, `SELECT count(*) FROM users WHERE id = ?`, f.other); n != 1 {
		t.Errorf("another account was touched")
	}
}

func TestDeletionStateSurvivesARestart(t *testing.T) {
	db := openTestDatabase(t)
	state := deletionState{orm: db}

	if saved, err := state.Load(context.Background()); err != nil || saved != nil {
		t.Fatalf("empty Load = %q, %v; want nil", saved, err)
	}
	for _, blob := range []string{"one", "two"} {
		if err := state.Save(context.Background(), []byte(blob)); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	if saved, err := state.Load(context.Background()); err != nil || string(saved) != "two" {
		t.Fatalf("Load = %q, %v; want two", saved, err)
	}
}
