package caldav

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_caldav_test", Migrate: schemas.Migrate}
)

func openTestDatabase(t *testing.T) *gorm.DB {
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
	return testDB
}

func requireMirroredRejection(t *testing.T, err error) {
	t.Helper()
	if !stderrors.Is(err, errMirrored) {
		t.Fatalf("error = %v, want the read-only rejection", err)
	}
}

func TestMirroredEventRejectsCalDAVPutAndDelete(t *testing.T) {
	db := openTestDatabase(t)
	user := schemas.User{Email: "alice@example.com", Name: "Alice"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cal := schemas.Calendar{OwnerID: user.ID, Slug: "opus-tasks", Name: "Opus tasks", SyncToken: "1"}
	if err := db.Create(&cal).Error; err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	evt := schemas.Event{CalendarID: cal.ID, UID: "mirror@agenda", Title: "Ship it", Source: "opus", ExternalID: "t1"}
	if err := db.Create(&evt).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	backend := NewBackend(db)
	ctx := withUser(context.Background(), &user)
	objPath := davPrefix + "/" + user.Email + "/calendars/" + strconv.FormatInt(cal.ID, 10) + "/mirror@agenda.ics"

	requireMirroredRejection(t, backend.DeleteCalendarObject(ctx, objPath))

	body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//t//EN\r\nBEGIN:VEVENT\r\nUID:mirror@agenda\r\nDTSTAMP:20261001T000000Z\r\nDTSTART:20261104T090000Z\r\nSUMMARY:Hacked\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	parsed, err := ical.NewDecoder(strings.NewReader(body)).Decode()
	if err != nil {
		t.Fatalf("decode ics: %v", err)
	}
	_, err = backend.PutCalendarObject(ctx, objPath, parsed, &caldav.PutCalendarObjectOptions{})
	requireMirroredRejection(t, err)

	var stored schemas.Event
	if err := db.First(&stored, evt.ID).Error; err != nil || stored.Title != "Ship it" {
		t.Fatalf("mirrored event changed: %+v, %v", stored, err)
	}
}
