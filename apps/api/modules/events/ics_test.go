package events

import (
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
)

func recurringEvent() *schemas.Event {
	paris, _ := time.LoadLocation("Europe/Paris")
	evt := &schemas.Event{
		UID:            "series@agenda",
		Title:          "Standup",
		StartAt:        time.Date(2026, 11, 2, 9, 0, 0, 0, paris),
		EndAt:          time.Date(2026, 11, 2, 9, 30, 0, 0, paris),
		RecurrenceRule: "FREQ=DAILY;COUNT=5",
		Timezone:       "Europe/Paris",
		Status:         "confirmed",
	}
	evt.RawICS = buildRawICS(evt, paris)
	return evt
}

func TestDecorateRoundTripsAttendeesAndReminders(t *testing.T) {
	evt := recurringEvent()
	x := extras{
		attendees: []Attendee{{Email: "bob@example.com", Name: "Bob", Role: "optional", Response: "accepted"}},
		reminders: []Reminder{{MinutesBefore: 15, Method: "display"}, {MinutesBefore: 0, Method: "audio"}},
	}
	evt.RawICS = mustDecorate(t, evt.RawICS, "", x, false)
	for _, want := range []string{"ATTENDEE;", "ROLE=OPT-PARTICIPANT", "PARTSTAT=ACCEPTED", "BEGIN:VALARM", "TRIGGER:-PT15M"} {
		if !strings.Contains(evt.RawICS, want) {
			t.Fatalf("ICS lacks %q:\n%s", want, evt.RawICS)
		}
	}
	d := readDetails(evt)
	if len(d.attendees) != 1 || d.attendees[0] != x.attendees[0] {
		t.Fatalf("attendees = %+v", d.attendees)
	}
	if len(d.reminders) != 2 || d.reminders[0].MinutesBefore != 15 || d.reminders[1].Method != "audio" {
		t.Fatalf("reminders = %+v", d.reminders)
	}
	if d.timezone != "Europe/Paris" {
		t.Fatalf("timezone = %q", d.timezone)
	}
}

func TestDecorateKeepsPreviousExtrasWhenUnset(t *testing.T) {
	evt := recurringEvent()
	set := extras{attendees: []Attendee{{Email: "a@example.com"}}, reminders: []Reminder{{MinutesBefore: 5}}}
	prev := mustDecorate(t, evt.RawICS, "", extras{attendees: normalized(t, set.attendees), reminders: set.reminders}, false)

	kept := mustDecorate(t, buildRawICS(evt, requestLocation("Europe/Paris")), prev, extras{}, true)
	d := readDetails(&schemas.Event{RawICS: kept})
	if len(d.attendees) != 1 || len(d.reminders) != 1 {
		t.Fatalf("unset extras were dropped: %+v", d)
	}
	cleared := mustDecorate(t, buildRawICS(evt, requestLocation("Europe/Paris")), prev, extras{attendees: []Attendee{}, reminders: []Reminder{}}, true)
	d = readDetails(&schemas.Event{RawICS: cleared})
	if len(d.attendees) != 0 || len(d.reminders) != 0 {
		t.Fatalf("empty extras did not clear: %+v", d)
	}
}

func normalized(t *testing.T, in []Attendee) []Attendee {
	t.Helper()
	out, err := normalizeAttendees(in)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestValidationRejectsBadValues(t *testing.T) {
	bad := []Attendee{{Email: "not-an-email"}}
	if _, err := normalizeAttendees(bad); err == nil {
		t.Error("a bad attendee email was accepted")
	}
	if _, err := normalizeAttendees([]Attendee{{Email: "a@b.co", Role: "boss"}}); err == nil {
		t.Error("a bad attendee role was accepted")
	}
	if _, err := normalizeReminders([]Reminder{{MinutesBefore: 5, Method: "carrier-pigeon"}}); err == nil {
		t.Error("a bad reminder method was accepted")
	}
	if _, err := normalizeReminders([]Reminder{{MinutesBefore: -1}}); err == nil {
		t.Error("a negative reminder was accepted")
	}
	if err := validateStatus("maybe"); err == nil {
		t.Error("an unknown status was accepted")
	}
	if err := validateTimezone("Mars/Olympus"); err == nil {
		t.Error("an unknown timezone was accepted")
	}
	color := "red"
	if err := validateColor(&color); err == nil {
		t.Error("a non-hex color was accepted")
	}
}

func TestNormalizeAttendeesDeduplicatesAndLowercases(t *testing.T) {
	out, err := normalizeAttendees([]Attendee{{Email: "Bob@Example.com"}, {Email: "bob@example.com", Name: "Dup"}})
	if err != nil || len(out) != 1 || out[0].Email != "bob@example.com" || out[0].Role != "required" || out[0].Response != "needs-action" {
		t.Fatalf("normalize = %+v, %v", out, err)
	}
}

func mustDecorate(t *testing.T, raw, prev string, x extras, keep bool) string {
	t.Helper()
	out, err := decorate(raw, prev, x, keep)
	if err != nil {
		t.Fatalf("decorate: %v", err)
	}
	return out
}
