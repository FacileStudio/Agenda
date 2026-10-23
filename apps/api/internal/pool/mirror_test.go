package pool

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	antenne "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_pool_test", Migrate: schemas.Migrate}
)

type fixture struct {
	db     *gorm.DB
	mirror *Mirror
	events *events.Service
	alice  schemas.User
	bob    schemas.User
	ctx    context.Context
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	testDBOnce.Do(func() { testDB, testDBErr = testdb.Open(url, testDBCfg) })
	if testDBErr != nil {
		t.Fatalf("test database: %v", testDBErr)
	}
	if err := testdb.Truncate(testDB, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}

	f := &fixture{db: testDB, ctx: context.Background()}
	f.events = events.NewService(testDB)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	f.mirror = NewMirror(testDB, f.events, calendars.NewService(testDB), logger)
	f.alice = schemas.User{Email: "alice@example.com", Name: "Alice"}
	f.bob = schemas.User{Email: "bob@example.com", Name: "Bob"}
	for _, u := range []*schemas.User{&f.alice, &f.bob} {
		if err := testDB.Create(u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	return f
}

var clock atomic.Int64

func task(facileID, title, due, email string) json.RawMessage {
	return taskEvent("Opus", facileID, title, due, email, 1_700_000_000_000+clock.Add(1))
}

func taskEvent(app, facileID, title, due, email string, version int64) json.RawMessage {
	payload := map[string]any{"facile_id": facileID, "title": title}
	if due != "" {
		payload["due_date"] = due
	}
	if email != "" {
		payload["actor_email"] = email
	}
	raw, _ := json.Marshal(map[string]any{
		"version": 1, "app": app, "facile_id": facileID, "payload": payload,
		"idempotency_key": facileID + ":" + strconv.FormatInt(version, 10),
	})
	return raw
}

func (f *fixture) apply(t *testing.T, channel string, raw json.RawMessage) {
	t.Helper()
	if err := f.mirror.Apply(f.ctx, channel, raw); err != nil {
		t.Fatalf("apply %s: %v", channel, err)
	}
}

func (f *fixture) mirrored(t *testing.T) []schemas.Event {
	t.Helper()
	var evts []schemas.Event
	if err := f.db.Where("source = ?", SourceOpus).Order("id").Find(&evts).Error; err != nil {
		t.Fatalf("list mirrored: %v", err)
	}
	return evts
}

func (f *fixture) calendarOf(t *testing.T, id int64) schemas.Calendar {
	t.Helper()
	var cal schemas.Calendar
	if err := f.db.First(&cal, id).Error; err != nil {
		t.Fatalf("load calendar: %v", err)
	}
	return cal
}

func TestUpsertCreatesAllDayEventInOpusCalendar(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04T00:00:00Z", f.alice.Email))

	evts := f.mirrored(t)
	if len(evts) != 1 {
		t.Fatalf("mirrored events = %d, want 1", len(evts))
	}
	evt := evts[0]
	if evt.Title != "Ship it" || !evt.IsAllDay || evt.ExternalID != "t1" {
		t.Fatalf("unexpected event %+v", evt)
	}
	if got := evt.StartAt.UTC().Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("start = %s, want 2026-11-04", got)
	}
	if !evt.EndAt.Equal(evt.StartAt.Add(allDayEndOffset)) {
		t.Fatalf("end = %s, want start + %s", evt.EndAt, allDayEndOffset)
	}
	if !strings.Contains(evt.RawICS, "DTSTART;VALUE=DATE:20261104") || !strings.Contains(evt.RawICS, "DTEND;VALUE=DATE:20261105") || !strings.Contains(evt.RawICS, "SUMMARY:Ship it") {
		t.Fatalf("raw ics not regenerated:\n%s", evt.RawICS)
	}
	cal := f.calendarOf(t, evt.CalendarID)
	if cal.Name != "Opus tasks" || cal.OwnerID != f.alice.ID || cal.SyncToken == "" {
		t.Fatalf("unexpected calendar %+v", cal)
	}
}

func TestUpsertUpdatesTitleAndDateAndBumpsSyncToken(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	before := f.calendarOf(t, f.mirrored(t)[0].CalendarID).SyncToken

	f.apply(t, "task.updated", task("t1", "Ship it now", "2026-11-09", f.alice.Email))

	evts := f.mirrored(t)
	if len(evts) != 1 {
		t.Fatalf("mirrored events = %d, want 1", len(evts))
	}
	if evts[0].Title != "Ship it now" || evts[0].StartAt.UTC().Format("2006-01-02") != "2026-11-09" {
		t.Fatalf("event not updated: %+v", evts[0])
	}
	if f.calendarOf(t, evts[0].CalendarID).SyncToken == before {
		t.Fatalf("sync token was not bumped")
	}
}

func TestReplayIsIdempotent(t *testing.T) {
	f := newFixture(t)
	raw := task("t1", "Ship it", "2026-11-04", f.alice.Email)
	f.apply(t, "task.created", raw)
	f.apply(t, "task.created", raw)
	f.apply(t, "task.updated", raw)

	if n := len(f.mirrored(t)); n != 1 {
		t.Fatalf("mirrored events = %d, want 1", n)
	}
	var calendarCount int64
	f.db.Model(&schemas.Calendar{}).Where("name = ?", "Opus tasks").Count(&calendarCount)
	if calendarCount != 1 {
		t.Fatalf("Opus tasks calendars = %d, want 1", calendarCount)
	}
}

func TestClearingDueDateDeletesEvent(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	calID := f.mirrored(t)[0].CalendarID
	before := f.calendarOf(t, calID).SyncToken

	f.apply(t, "task.updated", task("t1", "Ship it", "", f.alice.Email))

	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("mirrored events = %d, want 0", n)
	}
	if f.calendarOf(t, calID).SyncToken == before {
		t.Fatalf("sync token was not bumped on delete")
	}
}

func TestDeletedTaskDeletesEvent(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	f.apply(t, "task.deleted", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	f.apply(t, "task.deleted", task("t1", "Ship it", "2026-11-04", f.alice.Email))

	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("mirrored events = %d, want 0", n)
	}
}

func TestReassignMovesEventToNewAssigneeCalendar(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	aliceCal := f.mirrored(t)[0].CalendarID

	f.apply(t, "task.actor_changed", task("t1", "Ship it", "2026-11-04", "BOB@example.com"))

	evts := f.mirrored(t)
	if len(evts) != 1 {
		t.Fatalf("mirrored events = %d, want 1", len(evts))
	}
	if evts[0].CalendarID == aliceCal {
		t.Fatalf("event stayed in the old assignee's calendar")
	}
	if f.calendarOf(t, evts[0].CalendarID).OwnerID != f.bob.ID {
		t.Fatalf("event is not in Bob's calendar")
	}
	var left int64
	f.db.Model(&schemas.Event{}).Where("calendar_id = ?", aliceCal).Count(&left)
	if left != 0 {
		t.Fatalf("old calendar still holds %d events", left)
	}
}

func TestUnknownOrMissingAssigneeIsSkipped(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", "nobody@example.com"))
	f.apply(t, "task.created", task("t2", "Ship it", "2026-11-04", ""))
	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("mirrored events = %d, want 0", n)
	}

	f.apply(t, "task.created", task("t3", "Ship it", "2026-11-04", f.alice.Email))
	f.apply(t, "task.updated", task("t3", "Ship it", "2026-11-04", ""))
	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("unassigned task kept its event: %d", n)
	}
}

func TestDueDateAcceptsDateOnlyAndRejectsGarbage(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	if n := len(f.mirrored(t)); n != 1 {
		t.Fatalf("date-only due date mirrored %d events, want 1", n)
	}
	f.apply(t, "task.updated", task("t1", "Ship it", "next tuesday", f.alice.Email))
	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("unparsable due date kept %d events, want 0", n)
	}
}

func TestBarePayloadWithoutEnvelopeIsIgnored(t *testing.T) {
	f := newFixture(t)
	raw := json.RawMessage(`{"facile_id":"t9","title":"Bare","due_date":"2026-12-01T00:00:00Z","actor_email":"alice@example.com"}`)
	f.apply(t, "task.created", raw)
	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("mirrored events = %d, want 0", n)
	}
}

func TestSablierEventsNeverTouchMirroredEvents(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	future := int64(4_000_000_000_000)

	f.apply(t, "task.updated", taskEvent("Sablier", "t1", "Renamed", "", "", future))
	f.apply(t, "task.deleted", taskEvent("Sablier", "t1", "Renamed", "", "", future+1))
	f.apply(t, "task.created", taskEvent("Sablier", "t2", "Other", "2026-11-05", f.alice.Email, future))

	evts := f.mirrored(t)
	if len(evts) != 1 || evts[0].Title != "Ship it" || evts[0].ExternalID != "t1" {
		t.Fatalf("sablier events changed the mirror: %+v", evts)
	}
}

func TestStatusAndActorChannelsWithoutDueDateKeepEvent(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	f.apply(t, "task.status_changed", task("t1", "Ship it", "", f.alice.Email))
	f.apply(t, "task.actor_changed", task("t1", "Ship it", "", f.alice.Email))
	if n := len(f.mirrored(t)); n != 1 {
		t.Fatalf("mirrored events = %d, want 1", n)
	}
}

func TestStaleUpdateDoesNotOverwriteNewerState(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.updated", taskEvent("Opus", "t1", "New", "2026-11-09", f.alice.Email, 2000))
	f.apply(t, "task.updated", taskEvent("Opus", "t1", "Old", "2026-11-04", f.alice.Email, 1000))

	evts := f.mirrored(t)
	if len(evts) != 1 || evts[0].Title != "New" || evts[0].StartAt.UTC().Format("2006-01-02") != "2026-11-09" {
		t.Fatalf("stale event overwrote newer state: %+v", evts)
	}
}

func TestDeletedTaskIsNotResurrectedByReplayedCreate(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", taskEvent("Opus", "t1", "Ship it", "2026-11-04", f.alice.Email, 1000))
	f.apply(t, "task.deleted", taskEvent("Opus", "t1", "Ship it", "2026-11-04", f.alice.Email, 2000))
	f.apply(t, "task.created", taskEvent("Opus", "t1", "Ship it", "2026-11-04", f.alice.Email, 1000))

	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("replayed create resurrected the task: %d events", n)
	}
}

func TestLegacyNumericIdsAreIgnored(t *testing.T) {
	f := newFixture(t)
	payload := map[string]any{"id": 42, "facile_id": "42", "title": "Old", "due_date": "2026-11-04T00:00:00Z", "actor_email": f.alice.Email}
	raw, _ := json.Marshal(map[string]any{"app": "Opus", "facile_id": "42", "payload": payload})
	f.apply(t, "task.created", raw)
	if n := len(f.mirrored(t)); n != 0 {
		t.Fatalf("legacy numeric id mirrored %d events, want 0", n)
	}
}

func TestDueDateKeepsWrittenCalendarDate(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04T00:00:00+01:00", f.alice.Email))
	if got := f.mirrored(t)[0].StartAt.UTC().Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("start = %s, want 2026-11-04", got)
	}
}

func TestHandlerRecoversFromPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler(&Mirror{logger: logger}, "task.created", logger)
	h(task("t1", "Ship it", "2026-11-04", "alice@example.com"), antenne.EventMeta{})
}

func TestMirroredEventsAreReadOnlyThroughService(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", task("t1", "Ship it", "2026-11-04", f.alice.Email))
	evt := f.mirrored(t)[0]

	_, err := f.events.UpdateEvent(f.ctx, f.alice.ID, evt.ID, &events.UpdateEventRequest{
		Title: "Hacked", StartAt: evt.StartAt, EndAt: evt.EndAt, IsAllDay: true,
	})
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("update error = %v, want read-only", err)
	}
	other, err := f.events.UpdateEvent(f.ctx, f.alice.ID, evt.ID, &events.UpdateEventRequest{
		CalendarID: evt.CalendarID + 1, Title: "Moved", StartAt: evt.StartAt, EndAt: evt.EndAt,
	})
	if err == nil || other != nil {
		t.Fatalf("move was not rejected: %v", err)
	}
	err = f.events.DeleteEvent(f.ctx, f.alice.ID, evt.ID)
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("delete error = %v, want read-only", err)
	}
	if n := len(f.mirrored(t)); n != 1 {
		t.Fatalf("mirrored event was removed")
	}

	got, err := f.events.GetEvent(f.ctx, f.alice.ID, evt.ID)
	if err != nil || got.Source != SourceOpus || got.ExternalID != "t1" {
		t.Fatalf("source marker missing from response: %+v, %v", got, err)
	}
}

func TestUnassignedTaskFallsBackToOwnerEmail(t *testing.T) {
	f := newFixture(t)
	raw, _ := json.Marshal(map[string]any{
		"version": 1, "app": "Opus", "facile_id": "t9",
		"payload":         map[string]any{"facile_id": "t9", "title": "Unassigned", "due_date": "2026-11-04T00:00:00Z", "owner_email": f.bob.Email},
		"idempotency_key": "t9:" + strconv.FormatInt(1_700_000_000_000+clock.Add(1), 10),
	})
	f.apply(t, "task.updated", raw)

	evts := f.mirrored(t)
	if len(evts) != 1 {
		t.Fatalf("mirrored events = %d, want 1", len(evts))
	}
	if cal := f.calendarOf(t, evts[0].CalendarID); cal.OwnerID != f.bob.ID {
		t.Fatalf("calendar owner = %d, want %d", cal.OwnerID, f.bob.ID)
	}
}
