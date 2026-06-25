package caldav

import (
	"encoding/xml"
	"testing"
	"time"

	"api/schemas"
)

// Apple's MKCALENDAR body uses namespaced elements; confirm displayname and
// color are extracted and the #RRGGBBAA color is trimmed to #RRGGBB.
func TestMkcalendarBodyParse(t *testing.T) {
	body := `<?xml version="1.0" encoding="utf-8"?>
<B:mkcalendar xmlns:B="urn:ietf:params:xml:ns:caldav">
  <A:set xmlns:A="DAV:">
    <A:prop>
      <A:displayname>Personal</A:displayname>
      <D:calendar-color xmlns:D="http://apple.com/ns/ical/">#1D9BF6FF</D:calendar-color>
    </A:prop>
  </A:set>
</B:mkcalendar>`
	var m mkcalendarBody
	if err := xml.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.DisplayName != "Personal" {
		t.Fatalf("displayname = %q, want Personal", m.DisplayName)
	}
	if got := normalizeColor(m.Color); got != "#1D9BF6" {
		t.Fatalf("normalizeColor = %q, want #1D9BF6", got)
	}
}

// macOS Reminders (remindd) creates a VTODO-only collection via MKCALENDAR;
// Agenda is events-only, so such a request must be detected and rejected.
func TestMkcalendarTasksOnlyDetection(t *testing.T) {
	tasks := `<?xml version="1.0" encoding="utf-8"?>
<B:mkcalendar xmlns:B="urn:ietf:params:xml:ns:caldav">
  <A:set xmlns:A="DAV:">
    <A:prop>
      <A:displayname>DEFAULT_TASK_CALENDAR_NAME</A:displayname>
      <B:supported-calendar-component-set>
        <B:comp name="VTODO"/>
      </B:supported-calendar-component-set>
    </A:prop>
  </A:set>
</B:mkcalendar>`
	var tm mkcalendarBody
	if err := xml.Unmarshal([]byte(tasks), &tm); err != nil {
		t.Fatalf("unmarshal tasks: %v", err)
	}
	if !tm.isTasksOnly() {
		t.Fatalf("VTODO-only collection must be rejected, isTasksOnly=false")
	}

	events := `<?xml version="1.0" encoding="utf-8"?>
<B:mkcalendar xmlns:B="urn:ietf:params:xml:ns:caldav">
  <A:set xmlns:A="DAV:">
    <A:prop>
      <A:displayname>Work</A:displayname>
      <B:supported-calendar-component-set>
        <B:comp name="VEVENT"/>
      </B:supported-calendar-component-set>
    </A:prop>
  </A:set>
</B:mkcalendar>`
	var em mkcalendarBody
	if err := xml.Unmarshal([]byte(events), &em); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if em.isTasksOnly() {
		t.Fatalf("VEVENT collection must be allowed, isTasksOnly=true")
	}

	var none mkcalendarBody
	if none.isTasksOnly() {
		t.Fatalf("empty component set must be allowed, isTasksOnly=true")
	}
}

// A single event with empty or corrupt raw_ics must not fail the CalDAV read;
// otherwise one bad row breaks the entire calendar's sync.
func TestToCalendarObjectFallback(t *testing.T) {
	start := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"empty":   "",
		"corrupt": "not a calendar",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			e := &schemas.Event{
				UID:     "u-" + name,
				Title:   "Rebuilt",
				StartAt: start,
				EndAt:   start.Add(time.Hour),
				Status:  "confirmed",
				RawICS:  raw,
			}
			co, err := toCalendarObject(e, "/dav/x/calendars/1/"+e.UID+".ics")
			if err != nil {
				t.Fatalf("expected fallback, got error: %v", err)
			}
			evts := co.Data.Events()
			if len(evts) != 1 {
				t.Fatalf("expected 1 rebuilt VEVENT, got %d", len(evts))
			}
			if uid, _ := evts[0].Props.Text("UID"); uid != e.UID {
				t.Fatalf("rebuilt UID = %q, want %q", uid, e.UID)
			}
		})
	}
}
