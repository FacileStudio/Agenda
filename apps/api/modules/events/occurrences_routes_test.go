package events

import (
	"net/http"
	"strings"
	"testing"
)

const series = "02T08:00 Standup|03T08:00 Standup|04T08:00 Standup|05T08:00 Standup|06T08:00 Standup"

func TestCancelOccurrenceExcludesItFromTheList(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	mustEqual(t, h.list(), series)

	code, got, _ := h.do(http.MethodDelete, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-04", nil)
	if code != http.StatusOK || len(got.Exceptions) != 1 || !got.Exceptions[0].Cancelled || got.Exceptions[0].Date != "2026-11-04" {
		t.Fatalf("cancel = %d %+v", code, got.Exceptions)
	}
	mustEqual(t, h.list(), "02T08:00 Standup|03T08:00 Standup|05T08:00 Standup|06T08:00 Standup")
	if !strings.Contains(h.raw(evt.ID), "EXDATE;TZID=Europe/Paris:20261104T090000") {
		t.Fatalf("ICS lacks the EXDATE:\n%s", h.raw(evt.ID))
	}
	code, _, _ = h.do(http.MethodDelete, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-04", nil)
	if strings.Contains(h.raw(evt.ID), "SEQUENCE;") {
		t.Fatalf("SEQUENCE carries a type parameter:\n%s", h.raw(evt.ID))
	}
	if code != http.StatusOK || strings.Count(h.raw(evt.ID), "EXDATE") != 1 {
		t.Fatalf("a repeated cancel was not idempotent: %d", code)
	}
}

func TestOverrideOccurrenceChangesOnlyThatOccurrence(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	path := "/events/" + itoa(evt.ID) + "/occurrences/2026-11-03"
	code, got, _ := h.do(http.MethodPut, path, map[string]any{"title": "Moved", "start_at": "2026-11-03T14:00:00Z"})
	if code != http.StatusOK || len(got.Exceptions) != 1 || got.Exceptions[0].Title != "Moved" {
		t.Fatalf("override = %d %+v", code, got.Exceptions)
	}
	mustEqual(t, h.list(), "02T08:00 Standup|03T14:00 Moved|04T08:00 Standup|05T08:00 Standup|06T08:00 Standup")
	raw := h.raw(evt.ID)
	if !strings.Contains(raw, "RECURRENCE-ID;TZID=Europe/Paris:20261103T090000") || !strings.Contains(raw, "SUMMARY:Moved") {
		t.Fatalf("ICS lacks the override:\n%s", raw)
	}
	code, got, _ = h.do(http.MethodPut, path, map[string]any{"location": "Room 2"})
	if code != http.StatusOK || len(got.Exceptions) != 1 || got.Exceptions[0].Location != "Room 2" || got.Exceptions[0].Title != "Moved" {
		t.Fatalf("second override did not merge: %d %+v", code, got.Exceptions)
	}
}

func TestRestoreOccurrenceBringsItBack(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	base := "/events/" + itoa(evt.ID) + "/occurrences/"
	h.do(http.MethodDelete, base+"2026-11-04", nil)
	h.do(http.MethodPut, base+"2026-11-05", map[string]any{"title": "Odd"})
	h.do(http.MethodPost, base+"2026-11-04/restore", nil)
	code, got, _ := h.do(http.MethodPost, base+"2026-11-05/restore", nil)
	if code != http.StatusOK || len(got.Exceptions) != 0 {
		t.Fatalf("restore = %d %+v", code, got.Exceptions)
	}
	mustEqual(t, h.list(), series)
	if strings.Contains(h.raw(evt.ID), "EXDATE") || strings.Contains(h.raw(evt.ID), "RECURRENCE-ID") {
		t.Fatalf("restore left exception data:\n%s", h.raw(evt.ID))
	}
}

func TestOccurrenceRoutesRejectBadTargets(t *testing.T) {
	h := newHarness(t)
	single := h.create(map[string]any{"title": "Once", "start_at": "2026-11-02T08:00:00Z", "end_at": "2026-11-02T09:00:00Z"})
	evt := h.create(weekly(nil))
	cases := map[string]string{
		"/events/" + itoa(single.ID) + "/occurrences/2026-11-02": "non-recurring",
		"/events/" + itoa(evt.ID) + "/occurrences/2026-12-25":    "no such date",
		"/events/" + itoa(evt.ID) + "/occurrences/tomorrow":      "bad date",
	}
	for path, name := range cases {
		if code, _, _ := h.do(http.MethodDelete, path, nil); code != http.StatusBadRequest {
			t.Errorf("%s: code = %d, want 400", name, code)
		}
	}
	h.do(http.MethodDelete, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-03", nil)
	code, _, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-03", map[string]any{"title": "x"})
	if code != http.StatusBadRequest {
		t.Errorf("overriding a cancelled occurrence = %d, want 400", code)
	}
	code, _, _ = h.do(http.MethodPut, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-04", map[string]any{"status": "cancelled"})
	if code != http.StatusBadRequest {
		t.Errorf("status cancelled on an override = %d, want 400", code)
	}
}

func TestOccurrenceRoutesNeedWriteAccess(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	stranger := "999999"
	reader := newController(NewService(h.db))
	other := h.router
	h.router = chiWith(reader, &stranger)
	code, _, _ := h.do(http.MethodDelete, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-03", nil)
	if code != http.StatusForbidden {
		t.Fatalf("stranger cancel = %d, want 403", code)
	}
	h.router = other
}

func TestScheduleEditDropsExceptionsOtherEditsKeepThem(t *testing.T) {
	h := newHarness(t)
	evt := h.create(weekly(nil))
	h.do(http.MethodDelete, "/events/"+itoa(evt.ID)+"/occurrences/2026-11-03", nil)

	_, got, _ := h.do(http.MethodPut, "/events/"+itoa(evt.ID), weekly(map[string]any{"title": "Daily"}))
	if len(got.Exceptions) != 1 {
		t.Fatalf("a title edit dropped the exception: %+v", got.Exceptions)
	}
	_, got, _ = h.do(http.MethodPut, "/events/"+itoa(evt.ID), weekly(map[string]any{"start_at": "2026-11-02T09:00:00Z", "end_at": "2026-11-02T09:30:00Z"}))
	if len(got.Exceptions) != 0 {
		t.Fatalf("a schedule edit kept the exception: %+v", got.Exceptions)
	}
}

func TestExpandedListReportsOccurrenceStart(t *testing.T) {
	h := newHarness(t)
	h.create(weekly(nil))
	path := "/calendars/" + itoa(h.calID) + "/events/?from=2026-11-01T00:00:00Z&to=2026-11-30T00:00:00Z"
	_, _, many := h.do(http.MethodGet, path, nil)
	if len(many) != 5 || many[1].OccurrenceStart == nil || many[1].OccurrenceStart.UTC().Format("2006-01-02") != "2026-11-03" {
		t.Fatalf("occurrence_start missing: %+v", many)
	}
}
