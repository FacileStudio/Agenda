package pool

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

const opusTaskEnvelope = `{"version":1,"app":"Opus","object":"task","action":"%ACTION%","facile_id":"cm9task0000000000000001","payload":{"id":42,"facile_id":"cm9task0000000000000001","project_id":3,"project_facile_id":"cm9proj","workspace_id":1,"title":"Ship it","name":"Ship it","status":"todo","priority":"high"%DUE%,"actor_id":7,"actor_email":"alice@example.com"},"timestamp":"2026-10-23T09:30:00Z","idempotency_key":"cm9task0000000000000001:%VERSION%"}`

func opusEnvelope(action string, withDue bool, version int64) json.RawMessage {
	due := ""
	if withDue {
		due = `,"due_date":"2026-11-04T00:00:00Z"`
	}
	return json.RawMessage(strings.NewReplacer("%ACTION%", action, "%DUE%", due, "%VERSION%", strconv.FormatInt(version, 10)).Replace(opusTaskEnvelope))
}

func TestOpusEnvelopeCreatesAllDayEventKeyedByFacileID(t *testing.T) {
	f := newFixture(t)
	f.apply(t, "task.created", opusEnvelope("created", true, 1000))
	evts := f.mirrored(t)
	if len(evts) != 1 || evts[0].ExternalID != "cm9task0000000000000001" || !evts[0].IsAllDay {
		t.Fatalf("created = %+v, want one all-day event keyed by the task facile_id", evts)
	}
	if got := evts[0].StartAt.Format("2006-01-02"); got != "2026-11-04" {
		t.Fatalf("start = %s, want 2026-11-04", got)
	}
	if cal := f.calendarOf(t, evts[0].CalendarID); cal.Name != calendarName {
		t.Fatalf("calendar = %q, want %q", cal.Name, calendarName)
	}
}

func TestOpusEnvelopeLifecycle(t *testing.T) {
	steps := []struct {
		name    string
		channel string
		action  string
		withDue bool
		version int64
		want    int
	}{
		{"create", "task.created", "created", true, 1000, 1},
		{"replayed create", "task.created", "created", true, 1000, 1},
		{"stale update", "task.updated", "updated", false, 900, 1},
		{"cleared due date", "task.updated", "updated", false, 1100, 0},
		{"restored due date", "task.updated", "updated", true, 1200, 1},
		{"delete", "task.deleted", "deleted", true, 1300, 0},
		{"stale update after delete", "task.updated", "updated", true, 1250, 0},
		{"replayed create after delete", "task.created", "created", true, 1000, 0},
	}
	f := newFixture(t)
	for _, step := range steps {
		f.apply(t, step.channel, opusEnvelope(step.action, step.withDue, step.version))
		if got := len(f.mirrored(t)); got != step.want {
			t.Fatalf("%s: %d mirrored events, want %d", step.name, got, step.want)
		}
	}
}
