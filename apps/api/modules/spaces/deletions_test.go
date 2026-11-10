package spaces

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
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
	avatar      string
}

func mustCreate(t *testing.T, db *gorm.DB, row any) {
	t.Helper()
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func seedAccess(t *testing.T, db *gorm.DB, userID int64) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO porte_sessions (token_hash, user_id, label) VALUES ('h-' || (?::bigint)::text, ?, 'caldav')`,
		`INSERT INTO mcp_actions (id, user_id, kind, target, digest, expires_at)
		 VALUES ('a-' || (?::bigint)::text, (?::bigint)::text, 'delete_event', '{}', 'd', now() + interval '1 hour')`,
	} {
		if err := db.Exec(statement, userID, userID).Error; err != nil {
			t.Fatalf("seed access: %v", err)
		}
	}
}

func seedAvatar(t *testing.T, db *gorm.DB, storage string, userID int64) string {
	t.Helper()
	path := filepath.Join(storage, "avatars", "user-gone.png")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("avatar dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatalf("avatar: %v", err)
	}
	if err := db.Exec(`UPDATE users SET avatar_upload_path = 'avatars/user-gone.png' WHERE id = ?`, userID).Error; err != nil {
		t.Fatalf("avatar path: %v", err)
	}
	return path
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
	seedAccess(t, db, gone)
	deletions, avatar := erasingDeletions(t, db, gone)
	return erasureFixture{gone: gone, other: other, deletions: deletions, avatar: avatar}
}

func erasingDeletions(t *testing.T, db *gorm.DB, gone int64) (*Deletions, string) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	storage := t.TempDir()
	erasers := Erasers{Account: auth.NewStore(sqlDB).EraseAccount}
	deletions := NewDeletions(db, directory.Client{Issuer: testIssuer}, erasers, storage,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	return deletions, seedAvatar(t, db, storage, gone)
}

func assertNothingReferences(t *testing.T, db *gorm.DB, f erasureFixture) {
	t.Helper()
	for _, query := range []string{
		`SELECT count(*) FROM users WHERE id = ?`,
		`SELECT count(*) FROM calendars WHERE owner_id = ?`,
		`SELECT count(*) FROM event_attendees WHERE user_id = ?`,
		`SELECT count(*) FROM calendar_members WHERE user_id = ?`,
		`SELECT count(*) FROM space_members WHERE user_id = ?`,
		`SELECT count(*) FROM events WHERE created_by_id = ?`,
		`SELECT count(*) FROM mcp_actions WHERE user_id = (?::bigint)::text`,
		`SELECT count(*) FROM porte_sessions WHERE user_id = ?`,
		`SELECT count(*) FROM porte_identities WHERE user_id = ?`,
	} {
		if got := count(t, db, query, f.gone); got != 0 {
			t.Errorf("%s = %d, want 0", query, got)
		}
	}
	if _, err := os.Stat(f.avatar); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the uploaded avatar survived: %v", err)
	}
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

	assertNothingReferences(t, db, f)
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
