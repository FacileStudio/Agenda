package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	"github.com/FacileStudio/Agenda/apps/api/internal/bus/bustest"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"github.com/FacileStudio/porte/kit"
	"gorm.io/gorm"
)

func erasureFixture(t *testing.T) (*gorm.DB, *kit.Store, schemas.Calendar, schemas.Event) {
	t.Helper()
	db := openTestDatabase(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	person := schemas.User{Email: "erased@example.com", Name: "Erased"}
	if err := db.Create(&person).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
	link(t, db, person.ID, "sub-erased")
	cal := schemas.Calendar{OwnerID: person.ID, Slug: "personal", Name: "Erased's calendar", IsPersonal: true}
	if err := db.Create(&cal).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	evt := schemas.Event{CalendarID: cal.ID, UID: "uid-erased", Title: "Doctor", Description: "private"}
	if err := db.Create(&evt).Error; err != nil {
		t.Fatalf("seed event: %v", err)
	}
	return db, NewStore(sqlDB, bus.New(true)), cal, evt
}

// TestErasureQueuesDeletionsWithIdsOnly checks an erased person's events and
// calendars are published as deletions in the erasure's transaction, and
// that the payloads name nothing but the ids.
func TestErasureQueuesDeletionsWithIdsOnly(t *testing.T) {
	db, store, cal, evt := erasureFixture(t)
	bustest.AssertQueued(t, db, func() error {
		return store.EraseAccount(context.Background(), testIssuer, "sub-erased")
	}, "calendar_event.deleted", "calendar.deleted")

	rows := bustest.Rows(t, db)
	var deletedEvent envelope.Event[envelope.CalendarEvent]
	decodeStrict(t, rows[0].Payload, &deletedEvent)
	if want := (envelope.CalendarEvent{FacileID: itoa(evt.ID), CalendarFacileID: itoa(cal.ID)}); deletedEvent.Payload != want {
		t.Fatalf("event payload %+v, want ids only %+v", deletedEvent.Payload, want)
	}
	var deletedCalendar envelope.Event[envelope.Calendar]
	decodeStrict(t, rows[1].Payload, &deletedCalendar)
	if want := (envelope.Calendar{FacileID: itoa(cal.ID)}); deletedCalendar.Payload != want {
		t.Fatalf("calendar payload %+v, want the id only %+v", deletedCalendar.Payload, want)
	}
	for _, row := range rows {
		for _, personal := range [][]byte{[]byte("erased@example.com"), []byte("Doctor"), []byte("Erased's calendar")} {
			if bytes.Contains(row.Payload, personal) {
				t.Fatalf("an erasure payload names %q: %s", personal, row.Payload)
			}
		}
	}
}

func TestErasureRollsBackWithItsEnqueue(t *testing.T) {
	db, store, _, _ := erasureFixture(t)
	bustest.AssertRollsBack(t, db, func() error {
		return store.EraseAccount(context.Background(), testIssuer, "sub-erased")
	}, func() bool { return bustest.Count(t, db, "users", "email = ?", "erased@example.com") == 1 })
}

func decodeStrict(t *testing.T, raw []byte, into any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
