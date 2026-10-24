package events

import (
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
)

func TestExpandOccurrencesKeepsWallClockAcrossDST(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	evt := &schemas.Event{
		UID:            "weekly@agenda",
		Title:          "Standup",
		StartAt:        time.Date(2026, 10, 19, 9, 0, 0, 0, paris),
		EndAt:          time.Date(2026, 10, 19, 9, 30, 0, 0, paris),
		RecurrenceRule: "FREQ=WEEKLY;COUNT=4",
	}
	evt.RawICS = buildRawICS(evt, paris)
	if !strings.Contains(evt.RawICS, "DTSTART;TZID=Europe/Paris:20261019T090000") {
		t.Fatalf("raw ICS lacks a zoned DTSTART:\n%s", evt.RawICS)
	}

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	occ := expandOccurrences(evt, time.UTC, from, to)
	if len(occ) != 4 {
		t.Fatalf("got %d occurrences, want 4", len(occ))
	}
	for _, o := range occ {
		local := o.StartAt.In(paris)
		if local.Hour() != 9 || local.Minute() != 0 {
			t.Fatalf("occurrence starts at %s, want 09:00 Paris", local)
		}
		if o.EndAt.Sub(o.StartAt) != 30*time.Minute {
			t.Fatalf("occurrence lasts %s, want 30m", o.EndAt.Sub(o.StartAt))
		}
	}
}

func TestExpandOccurrencesHonoursExdateAndOverrides(t *testing.T) {
	raw := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//test//EN",
		"BEGIN:VEVENT",
		"UID:daily@client",
		"DTSTAMP:20261001T000000Z",
		"DTSTART:20261001T080000Z",
		"DTEND:20261001T090000Z",
		"RRULE:FREQ=DAILY;COUNT=5",
		"EXDATE:20261002T080000Z",
		"SUMMARY:Daily",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:daily@client",
		"DTSTAMP:20261001T000000Z",
		"RECURRENCE-ID:20261003T080000Z",
		"DTSTART:20261003T140000Z",
		"DTEND:20261003T150000Z",
		"SUMMARY:Moved",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:daily@client",
		"DTSTAMP:20261001T000000Z",
		"RECURRENCE-ID:20261004T080000Z",
		"DTSTART:20261004T080000Z",
		"STATUS:CANCELLED",
		"END:VEVENT",
		"END:VCALENDAR",
		"",
	}, "\r\n")
	evt := &schemas.Event{
		Title:          "Daily",
		StartAt:        time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
		EndAt:          time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		RecurrenceRule: "FREQ=DAILY;COUNT=5",
		RawICS:         raw,
	}

	occ := expandOccurrences(evt, time.UTC, evt.StartAt, evt.StartAt.AddDate(0, 0, 10))
	var got []string
	for _, o := range occ {
		got = append(got, o.StartAt.Format("02 15:04 ")+o.Title)
	}
	want := "01 08:00 Daily|03 14:00 Moved|05 08:00 Daily"
	if strings.Join(got, "|") != want {
		t.Fatalf("occurrences = %q, want %q", strings.Join(got, "|"), want)
	}
}

func TestNormalizeRecurrenceRule(t *testing.T) {
	if got, err := normalizeRecurrenceRule(" RRULE:freq=weekly;byday=mo,we "); err != nil || got != "FREQ=WEEKLY;BYDAY=MO,WE" {
		t.Fatalf("normalize = %q, %v", got, err)
	}
	if _, err := normalizeRecurrenceRule("FREQ=SOMETIMES"); err == nil {
		t.Fatal("an invalid rule was accepted")
	}
}
