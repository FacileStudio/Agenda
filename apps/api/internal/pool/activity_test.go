package pool

import (
	"encoding/json"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
)

func activityEvent(app, facileID string, version int64, payload map[string]any) json.RawMessage {
	payload["facile_id"] = facileID
	raw, _ := json.Marshal(map[string]any{
		"version": 1, "app": app, "facile_id": facileID, "payload": payload,
		"idempotency_key": facileID + ":" + strconv.FormatInt(version, 10),
		"timestamp":       time.UnixMilli(1_700_000_000_000 + version).UTC().Format(time.RFC3339Nano),
	})
	return raw
}

func session(email string, stopped any, version int64) json.RawMessage {
	return activityEvent("Sablier", "s1", version, map[string]any{
		"user_email": email, "started_at": "2026-11-16T09:00:00Z", "stopped_at": stopped,
		"description": "Mockups", "task_facile_id": "t1", "project_facile_id": "p1",
	})
}

func call(emails []string, ended any, version int64) json.RawMessage {
	return activityEvent("Echo", "c1", version, map[string]any{
		"room_slug": "standup", "room_name": "Standup", "started_at": "2026-11-16T10:00:00Z",
		"ended_at": ended, "participant_emails": emails, "url": "https://echo.facile.studio/standup",
	})
}

func (f *fixture) activities(t *testing.T, source string) []schemas.Activity {
	t.Helper()
	var rows []schemas.Activity
	if err := f.db.Where("source = ?", source).Order("user_id").Find(&rows).Error; err != nil {
		t.Fatalf("list activities: %v", err)
	}
	return rows
}

func (f *fixture) record(t *testing.T, channel string, raw json.RawMessage) {
	t.Helper()
	recorder := NewActivities(f.db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := recorder.Apply(f.ctx, channel, raw); err != nil {
		t.Fatalf("apply %s: %v", channel, err)
	}
}

func TestSablierSessionLifecycleAndVersionGuard(t *testing.T) {
	f := newFixture(t)
	f.record(t, "time_entry.created", session(f.alice.Email, nil, 1000))
	rows := f.activities(t, SourceSablier)
	if len(rows) != 1 || rows[0].UserID != f.alice.ID || rows[0].EndAt != nil || rows[0].Title != "Mockups" {
		t.Fatalf("running session = %+v", rows)
	}

	f.record(t, "time_entry.updated", session(f.alice.Email, "2026-11-16T10:30:00Z", 2000))
	f.record(t, "time_entry.updated", session(f.alice.Email, nil, 1500))
	rows = f.activities(t, SourceSablier)
	if len(rows) != 1 || rows[0].EndAt == nil || rows[0].EndAt.UTC().Format("15:04") != "10:30" {
		t.Fatalf("stopped session after a stale replay = %+v", rows)
	}

	f.record(t, "time_entry.deleted", session(f.alice.Email, "2026-11-16T10:30:00Z", 3000))
	f.record(t, "time_entry.created", session(f.alice.Email, nil, 1000))
	if rows = f.activities(t, SourceSablier); len(rows) != 0 {
		t.Fatalf("deleted session came back: %+v", rows)
	}
}

func TestEchoCallFollowsItsParticipants(t *testing.T) {
	f := newFixture(t)
	f.record(t, "call.started", call([]string{f.alice.Email, "guest@elsewhere.test"}, nil, 1000))
	if rows := f.activities(t, SourceEcho); len(rows) != 1 || rows[0].UserID != f.alice.ID || rows[0].URL == "" {
		t.Fatalf("started call = %+v", rows)
	}

	f.record(t, "call.updated", call([]string{"BOB@example.com"}, nil, 2000))
	rows := f.activities(t, SourceEcho)
	if len(rows) != 1 || rows[0].UserID != f.bob.ID {
		t.Fatalf("after Alice left = %+v, want only Bob", rows)
	}

	f.record(t, "call.ended", call([]string{f.alice.Email, f.bob.Email}, "2026-11-16T10:45:00Z", 3000))
	f.record(t, "call.updated", call([]string{f.alice.Email}, nil, 2500))
	rows = f.activities(t, SourceEcho)
	if len(rows) != 2 || rows[0].EndAt == nil || rows[1].EndAt == nil {
		t.Fatalf("ended call = %+v, want two finished rows", rows)
	}
}

func TestActivityFromAnotherAppIsIgnored(t *testing.T) {
	f := newFixture(t)
	f.record(t, "call.started", activityEvent("Opus", "c9", 1000, map[string]any{
		"started_at": "2026-11-16T10:00:00Z", "participant_emails": []string{f.alice.Email},
	}))
	if rows := f.activities(t, SourceEcho); len(rows) != 0 {
		t.Fatalf("foreign call stored: %+v", rows)
	}
}

func echoEnvelope(key, timestamp string, emails []string, ended any) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"version": 1, "app": "Echo", "facile_id": "c2", "idempotency_key": key, "timestamp": timestamp,
		"payload": map[string]any{"facile_id": "c2", "room_name": "Review", "started_at": "2026-11-16T10:00:00Z",
			"ended_at": ended, "participant_emails": emails},
	})
	return raw
}

func TestLateEchoJoinNeverReopensAnEndedCall(t *testing.T) {
	f := newFixture(t)
	f.record(t, "call.started", echoEnvelope("echo:call:c2:started", "2026-11-16T10:00:00.120Z", []string{f.alice.Email}, nil))
	f.record(t, "call.updated", echoEnvelope("echo:call:c2:joined:7", "2026-11-16T10:05:00Z", []string{f.alice.Email, f.bob.Email}, nil))
	if rows := f.activities(t, SourceEcho); len(rows) != 2 {
		t.Fatalf("after the join = %+v, want two rows: a user-id key suffix must not read as a stale version", rows)
	}
	f.record(t, "call.ended", echoEnvelope("echo:call:c2:ended", "2026-11-16T10:30:00Z", []string{f.alice.Email, f.bob.Email}, "2026-11-16T10:30:00Z"))
	f.record(t, "call.updated", echoEnvelope("echo:call:c2:joined:9", "2026-11-16T10:30:02Z", []string{f.alice.Email, f.bob.Email}, nil))

	rows := f.activities(t, SourceEcho)
	if len(rows) != 2 || rows[0].EndAt == nil || rows[1].EndAt == nil {
		t.Fatalf("late join reopened the call: %+v", rows)
	}
}
