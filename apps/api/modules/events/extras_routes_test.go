package events

import (
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"net/http"
	"strings"
	"testing"
)

func TestCreateReturnsEveryField(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(map[string]any{
		"color":     "#ff8800",
		"attendees": []map[string]any{{"email": "Bob@Example.com", "name": "Bob", "role": "optional"}},
		"reminders": []map[string]any{{"minutes_before": 10, "method": "audio"}},
	}))
	if evt.Timezone != "Europe/Paris" || evt.Color != "#ff8800" {
		t.Fatalf("timezone/color = %q/%q", evt.Timezone, evt.Color)
	}
	if len(evt.Attendees) != 1 || evt.Attendees[0].Email != "bob@example.com" || evt.Attendees[0].Role != "optional" {
		t.Fatalf("attendees = %+v", evt.Attendees)
	}
	if len(evt.Reminders) != 1 || evt.Reminders[0].MinutesBefore != 10 {
		t.Fatalf("reminders = %+v", evt.Reminders)
	}
	_, got, _ := h.do(http.MethodGet, "/events/"+itoa(evt.ID), nil)
	if len(got.Attendees) != 1 || got.Timezone != "Europe/Paris" {
		t.Fatalf("show lost data: %+v", got)
	}
}

func TestCreateRejectsInvalidValues(t *testing.T) {
	h := newHarness(t)
	for name, extra := range map[string]map[string]any{
		"status":   {"status": "maybe"},
		"timezone": {"timezone": "Mars/Olympus"},
		"color":    {"color": "red"},
		"attendee": {"attendees": []map[string]any{{"email": "nope"}}},
		"reminder": {"reminders": []map[string]any{{"minutes_before": -5}}},
	} {
		code, _, _ := h.do(http.MethodPost, "/calendars/"+itoa(h.calID)+"/events/", weekly(extra))
		if code != http.StatusBadRequest {
			t.Errorf("%s: code = %d, want 400", name, code)
		}
	}
}

func TestUpdateKeepsOmittedFieldsAndZone(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(map[string]any{
		"color": "#112233", "attendees": []map[string]any{{"email": "a@example.com"}},
		"reminders": []map[string]any{{"minutes_before": 5}},
	}))
	body := weekly(map[string]any{"title": "Renamed"})
	delete(body, "timezone")
	code, got, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID), body)
	if code != http.StatusOK || got.Title != "Renamed" {
		t.Fatalf("update = %d %+v", code, got)
	}
	if got.Timezone != "Europe/Paris" || got.Color != "#112233" || len(got.Attendees) != 1 || len(got.Reminders) != 1 {
		t.Fatalf("update dropped stored fields: %+v", got)
	}
	if !strings.Contains(h.raw(evt.ID), "DTSTART;TZID=Europe/Paris:") {
		t.Fatalf("series was re-anchored:\n%s", h.raw(evt.ID))
	}
}

func TestUpdateClearsWhenEmpty(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(map[string]any{
		"color": "#112233", "attendees": []map[string]any{{"email": "a@example.com"}},
		"reminders": []map[string]any{{"minutes_before": 5}},
	}))
	body := weekly(map[string]any{"color": "", "attendees": []any{}, "reminders": []any{}})
	_, got, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID), body)
	if got.Color != "" || len(got.Attendees) != 0 || len(got.Reminders) != 0 {
		t.Fatalf("empty values did not clear: %+v", got)
	}
}

func TestUpdateRejectsInvalidStatus(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	code, _, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID), weekly(map[string]any{"status": "bogus"}))
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	h := newHarness(t)
	for i := 0; i < 2; i++ {
		if err := schemas.Migrate(h.db); err != nil {
			t.Fatalf("migrate run %d: %v", i, err)
		}
	}
}

func TestAPIEditKeepsWhatACalDAVClientWrote(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	client := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//client//EN\r\nBEGIN:VEVENT\r\nUID:x@agenda\r\n" +
		"DTSTAMP:20261101T000000Z\r\nDTSTART;TZID=Europe/Paris:20261102T090000\r\nDTEND;TZID=Europe/Paris:20261102T093000\r\n" +
		"RRULE:FREQ=DAILY;COUNT=5\r\nSUMMARY:Standup\r\nEXDATE;TZID=Europe/Paris:20261104T090000\r\n" +
		"ATTENDEE;CN=Eve;PARTSTAT=DECLINED:mailto:eve@example.com\r\n" +
		"BEGIN:VALARM\r\nACTION:DISPLAY\r\nTRIGGER:-PT30M\r\nDESCRIPTION:x\r\nEND:VALARM\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	h.db.Model(&schemas.Event{}).Where("id = ?", evt.ID).Update("raw_ics", client)

	_, got, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID), weekly(map[string]any{"title": "Edited"}))
	if len(got.Exceptions) != 1 || len(got.Attendees) != 1 || got.Attendees[0].Response != "declined" || got.Reminders[0].MinutesBefore != 30 {
		t.Fatalf("API edit lost client data: %+v", got)
	}
	mustEqual(t, h.list(), "02T08:00 Edited|03T08:00 Edited|05T08:00 Edited|06T08:00 Edited")
}

func TestAttendeesCarryTheSuitePicture(t *testing.T) {
	h := newHarness(t)
	h.db.Create(&schemas.User{Email: "lea@example.com", Name: "Lea", OIDCPictureURL: "https://sso.test/avatars/lea"})
	evt := h.create(weekly(map[string]any{"attendees": []map[string]any{
		{"email": "Lea@Example.com", "avatar_url": "https://ignored.test/x"},
		{"email": "guest@example.com"},
	}}))
	_, got, _ := h.do(http.MethodGet, "/events/"+itoa(evt.ID), nil)
	if len(got.Attendees) != 2 || got.Attendees[0].AvatarURL != "https://sso.test/avatars/lea" || got.Attendees[1].AvatarURL != "" {
		t.Fatalf("attendees = %+v", got.Attendees)
	}
	if strings.Contains(h.raw(evt.ID), "ignored.test") {
		t.Fatalf("a request avatar_url reached the ICS:\n%s", h.raw(evt.ID))
	}
}
