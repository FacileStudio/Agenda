package live_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/live"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const bootstrap = "docker compose up -d db   # or any PostgreSQL 16"

const (
	aliceID    = 910001
	bobID      = 910002
	calendarID = 910003
)

type stream struct {
	changes <-chan live.Change
	filter  *live.Filter
}

func openStreams(t *testing.T) (*gorm.DB, stream, stream) {
	t.Helper()
	url, ok := testdb.URL()
	if !ok {
		testdb.Announce(bootstrap)
		t.Skip(testdb.SkipReason(bootstrap))
	}
	cfg := testdb.Config{Prefix: "agenda_live_test", Migrate: schemas.Migrate}
	db, err := testdb.Open(url, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := testdb.Truncate(db, cfg); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	hub := live.NewHub()
	aliceChanges, stopAlice := hub.Subscribe()
	bobChanges, stopBob := hub.Subscribe()
	t.Cleanup(stopAlice)
	t.Cleanup(stopBob)
	go live.Listen(ctx, url, hub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	access := live.CalendarAccess(db)
	return db, stream{aliceChanges, live.NewFilter(aliceID, access)}, stream{bobChanges, live.NewFilter(bobID, access)}
}

// next returns the first frame s may see for which match holds, failing
// after the timeout.
func (s stream) next(t *testing.T, match func(live.Frame) bool) live.Frame {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case change := <-s.changes:
			frame, allowed, err := s.filter.Frame(context.Background(), change)
			if err != nil {
				t.Fatalf("frame: %v", err)
			}
			if allowed && match(frame) {
				return frame
			}
		case <-deadline:
			t.Fatal("no matching frame")
		}
	}
}

// none fails if any change already queued on s gives it a frame match accepts.
func (s stream) none(t *testing.T, match func(live.Frame) bool) {
	t.Helper()
	for {
		select {
		case change := <-s.changes:
			frame, allowed, err := s.filter.Frame(context.Background(), change)
			if err == nil && allowed && match(frame) {
				t.Fatalf("unexpected frame %+v", frame)
			}
		default:
			return
		}
	}
}

// waitListening pings the channel until the listener relays a ping, so the
// test's own writes cannot happen before LISTEN.
func waitListening(t *testing.T, db *gorm.DB, s stream) {
	t.Helper()
	for range 100 {
		db.Exec(`SELECT pg_notify(?, 'e:0')`, live.Channel)
		select {
		case change := <-s.changes:
			if change.Kind == live.KindEvent && change.ID == 0 {
				return
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("the listener never started")
}

func inCalendar(f live.Frame) bool { return f.CalendarID == calendarID && !f.Calendar }
func isActivity(f live.Frame) bool { return f.Activity }

func TestTriggersReachOnlyUsersWhoCanReadTheCalendar(t *testing.T) {
	db, alice, bob := openStreams(t)
	waitListening(t, db, alice)
	for _, user := range []schemas.User{{ID: aliceID, Email: "alice@live.test"}, {ID: bobID, Email: "bob@live.test"}} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	db.Create(&schemas.Calendar{ID: calendarID, OwnerID: aliceID, Slug: "work", Name: "Work"})

	db.Exec(`INSERT INTO events (calendar_id, uid, title, start_at, end_at) VALUES (?, 'raw-1', 'Raw SQL', now(), now())`, calendarID)
	alice.next(t, inCalendar)
	bob.none(t, inCalendar)

	db.Create(&schemas.CalendarMember{CalendarID: calendarID, UserID: bobID, Role: "reader"})
	bob.next(t, func(f live.Frame) bool { return f.CalendarID == calendarID && f.Calendar })
	db.Exec(`UPDATE events SET title = 'Renamed' WHERE uid = 'raw-1'`)
	bob.next(t, inCalendar)

	db.Create(&schemas.Activity{UserID: aliceID, Source: "echo", ExternalID: "c1", StartAt: time.Now()})
	alice.next(t, isActivity)
	bob.none(t, isActivity)
}

func TestParseChangeRejectsUnknownPayloads(t *testing.T) {
	for _, payload := range []string{"", "e:", "x:1", "e:abc", "ee:1", "r:1"} {
		if _, ok := live.ParseChange(payload); ok {
			t.Errorf("ParseChange(%q) accepted", payload)
		}
	}
	if c, ok := live.ParseChange("c:42"); !ok || c.Kind != live.KindCalendar || c.ID != 42 {
		t.Errorf("ParseChange(c:42) = %+v, %v", c, ok)
	}
}
