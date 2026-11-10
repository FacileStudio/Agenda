package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	"github.com/FacileStudio/Agenda/apps/api/internal/bus/bustest"
	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"gorm.io/gorm"
)

type davFixture struct {
	db      *gorm.DB
	backend *Backend
	user    schemas.User
	cal     schemas.Calendar
	ctx     context.Context
}

func newDavFixture(t *testing.T) *davFixture {
	t.Helper()
	db := openTestDatabase(t)
	f := &davFixture{db: db, backend: NewBackend(db, bus.New(true))}
	f.user = schemas.User{Email: "alice@example.com", Name: "Alice"}
	if err := db.Create(&f.user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	f.cal = schemas.Calendar{OwnerID: f.user.ID, Slug: "main", Name: "Main", SyncToken: "1"}
	if err := db.Create(&f.cal).Error; err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	f.ctx = withUser(context.Background(), &f.user)
	return f
}

func (f *davFixture) objectPath(uid string) string {
	return davPrefix + "/" + f.user.Email + "/calendars/" + strconv.FormatInt(f.cal.ID, 10) + "/" + uid + ".ics"
}

func (f *davFixture) put(t *testing.T, uid, summary string) error {
	t.Helper()
	body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//t//EN\r\nBEGIN:VEVENT\r\nUID:" + uid +
		"\r\nDTSTAMP:20261001T000000Z\r\nDTSTART:20261104T090000Z\r\nDTEND:20261104T100000Z\r\nSUMMARY:" + summary +
		"\r\nDESCRIPTION:private\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	parsed, err := ical.NewDecoder(strings.NewReader(body)).Decode()
	if err != nil {
		t.Fatalf("decode ics: %v", err)
	}
	_, err = f.backend.PutCalendarObject(f.ctx, f.objectPath(uid), parsed, &caldav.PutCalendarObjectOptions{})
	return err
}

func (f *davFixture) mkcalendar(segment string) error {
	request := httptest.NewRequest("MKCALENDAR", davPrefix+"/"+f.user.Email+"/calendars/"+segment+"/", nil)
	recorder := httptest.NewRecorder()
	f.backend.HandleMkcalendar(recorder, request.WithContext(f.ctx))
	if recorder.Code != http.StatusCreated {
		return &httpStatusError{recorder.Code}
	}
	return nil
}

type httpStatusError struct{ code int }

func (e *httpStatusError) Error() string { return "status " + strconv.Itoa(e.code) }

func TestCalDAVWritesQueueOneRowEach(t *testing.T) {
	f := newDavFixture(t)
	bustest.AssertQueued(t, f.db, func() error { return f.put(t, "evt-1", "Standup") }, "calendar_event.created")
	bustest.AssertQueued(t, f.db, func() error { return f.put(t, "evt-1", "Standup moved") }, "calendar_event.updated")
	bustest.AssertQueued(t, f.db, func() error {
		return f.backend.DeleteCalendarObject(f.ctx, f.objectPath("evt-1"))
	}, "calendar_event.deleted")
	bustest.AssertQueued(t, f.db, func() error {
		return f.backend.CreateCalendar(f.ctx, &caldav.Calendar{Path: davPrefix + "/" + f.user.Email + "/calendars/work/", Name: "Work"})
	}, "calendar.created")
	bustest.AssertQueued(t, f.db, func() error { return f.mkcalendar("apple") }, "calendar.created")
	bustest.AssertQueued(t, f.db, func() error {
		f.backend.ensurePersonalCalendar(f.ctx, f.user.ID)
		return nil
	}, "calendar.created")
}

func TestCalDAVWritesRollBackWithTheirEnqueue(t *testing.T) {
	f := newDavFixture(t)
	if err := f.put(t, "kept", "Kept"); err != nil {
		t.Fatalf("put: %v", err)
	}
	events := func() int64 { return bustest.Count(t, f.db, "events", "calendar_id = ?", f.cal.ID) }
	calendars := func() int64 { return bustest.Count(t, f.db, "calendars", "owner_id = ?", f.user.ID) }

	bustest.AssertRollsBack(t, f.db, func() error { return f.put(t, "lost", "Lost") }, func() bool { return events() == 1 })
	bustest.AssertRollsBack(t, f.db, func() error { return f.put(t, "kept", "Renamed") }, func() bool {
		return bustest.Count(t, f.db, "events", "uid = ? AND title = ?", "kept", "Kept") == 1
	})
	bustest.AssertRollsBack(t, f.db, func() error {
		return f.backend.DeleteCalendarObject(f.ctx, f.objectPath("kept"))
	}, func() bool { return events() == 1 })
	bustest.AssertRollsBack(t, f.db, func() error {
		return f.backend.CreateCalendar(f.ctx, &caldav.Calendar{Path: davPrefix + "/" + f.user.Email + "/calendars/lost/", Name: "Lost"})
	}, func() bool { return calendars() == 1 })
	bustest.AssertRollsBack(t, f.db, func() error { return f.mkcalendar("lost-apple") }, func() bool { return calendars() == 1 })
}
