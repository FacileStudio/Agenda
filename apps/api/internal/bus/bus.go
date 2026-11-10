// Package bus queues Agenda's calendar and event changes for the Antenne pool.
//
// Every change is written to antenne_outbox inside the transaction that makes
// it, so a change is published if and only if it committed. The outbox
// drainer in internal/pool sends the rows. A nil *Outbox is publishing turned
// off, which is what Agenda runs without the Antenne variables: every method
// is then a no-op and no row is written.
//
// Payloads carry only what envelope.Calendar and envelope.CalendarEvent
// define: never a description, a location or an attendee. The facile_id of
// both is the row's integer id as a decimal string; Agenda rows have no other
// stable public id.
package bus

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"github.com/FacileStudio/antenne-client/go/outbox"
	"gorm.io/gorm"
)

// Channels returns the channels Agenda publishes on, declared when it registers.
func Channels() []catalog.Channel {
	return []catalog.Channel{
		catalog.CalendarCreated, catalog.CalendarUpdated, catalog.CalendarDeleted,
		catalog.CalendarEventCreated, catalog.CalendarEventUpdated, catalog.CalendarEventDeleted,
	}
}

// Outbox enqueues changes. The zero value publishes; nil does nothing.
type Outbox struct{}

// New returns the outbox when publishing is on, and nil when it is off.
func New(enabled bool) *Outbox {
	if !enabled {
		return nil
	}
	return &Outbox{}
}

var calendarChannels = map[envelope.Action]catalog.Channel{
	envelope.ActionCreated: catalog.CalendarCreated,
	envelope.ActionUpdated: catalog.CalendarUpdated,
	envelope.ActionDeleted: catalog.CalendarDeleted,
}

var eventChannels = map[envelope.Action]catalog.Channel{
	envelope.ActionCreated: catalog.CalendarEventCreated,
	envelope.ActionUpdated: catalog.CalendarEventUpdated,
	envelope.ActionDeleted: catalog.CalendarEventDeleted,
}

// Calendar queues a calendar change made in tx by actorID. A space's calendar
// carries no owner email.
func (o *Outbox) Calendar(tx *gorm.DB, action envelope.Action, cal *schemas.Calendar, actorID int64) error {
	if o == nil {
		return nil
	}
	sqlTx, err := transactionOf(tx)
	if err != nil {
		return err
	}
	ctx := tx.Statement.Context
	payload := envelope.Calendar{FacileID: idOf(cal.ID), Name: cal.Name}
	if cal.SpaceID == nil {
		if payload.OwnerEmail, err = emailOf(ctx, sqlTx, cal.OwnerID); err != nil {
			return err
		}
	}
	if payload.ActorEmail, err = emailOf(ctx, sqlTx, actorID); err != nil {
		return err
	}
	return enqueue(ctx, sqlTx, change{calendarChannels[action], envelope.ObjectCalendar, action, payload.FacileID, cal.UpdatedAt}, payload)
}

// Event queues a change to an event Agenda owns, made in tx by actorID. An
// event mirrored from another app, with a source, is that app's to publish.
func (o *Outbox) Event(tx *gorm.DB, action envelope.Action, evt *schemas.Event, actorID int64) error {
	if o == nil || evt.Source != "" {
		return nil
	}
	sqlTx, err := transactionOf(tx)
	if err != nil {
		return err
	}
	ctx := tx.Statement.Context
	payload := envelope.CalendarEvent{
		FacileID:         idOf(evt.ID),
		CalendarFacileID: idOf(evt.CalendarID),
		Title:            evt.Title,
		StartsAt:         evt.StartAt.UTC().Format(time.RFC3339),
		EndsAt:           evt.EndAt.UTC().Format(time.RFC3339),
		AllDay:           evt.IsAllDay,
	}
	if payload.ActorEmail, err = emailOf(ctx, sqlTx, actorID); err != nil {
		return err
	}
	return enqueue(ctx, sqlTx, change{eventChannels[action], envelope.ObjectCalendarEvent, action, payload.FacileID, evt.UpdatedAt}, payload)
}

// ErasedCalendar queues the deletion of a calendar removed with an erased
// person. It carries the id and nothing personal.
func (o *Outbox) ErasedCalendar(ctx context.Context, tx *sql.Tx, calendarID int64) error {
	if o == nil {
		return nil
	}
	payload := envelope.Calendar{FacileID: idOf(calendarID)}
	return enqueue(ctx, tx, change{catalog.CalendarDeleted, envelope.ObjectCalendar, envelope.ActionDeleted, payload.FacileID, time.Time{}}, payload)
}

// ErasedEvent queues the deletion of an event removed with an erased person.
// It carries the two ids and nothing personal.
func (o *Outbox) ErasedEvent(ctx context.Context, tx *sql.Tx, eventID, calendarID int64) error {
	if o == nil {
		return nil
	}
	payload := envelope.CalendarEvent{FacileID: idOf(eventID), CalendarFacileID: idOf(calendarID)}
	return enqueue(ctx, tx, change{catalog.CalendarEventDeleted, envelope.ObjectCalendarEvent, envelope.ActionDeleted, payload.FacileID, time.Time{}}, payload)
}

// change names one queued change: where it goes and what it is about.
type change struct {
	channel   catalog.Channel
	object    envelope.ObjectType
	action    envelope.Action
	id        string
	updatedAt time.Time
}

// enqueue wraps payload in the suite envelope and writes it to the outbox. The
// idempotency key names the object, its id and the action, and for an update
// the row's updated_at, so a retry of one change reuses its key and two
// changes never share one. A created or deleted id happens once: ids are never
// reused.
func enqueue[T any](ctx context.Context, tx *sql.Tx, c change, payload T) error {
	key := "agenda:" + string(c.object) + ":" + c.id + ":" + string(c.action)
	if c.action == envelope.ActionUpdated {
		key += ":" + strconv.FormatInt(c.updatedAt.UnixNano(), 10)
	}
	event := envelope.Event[T]{
		Version:        envelope.EventVersion,
		App:            envelope.AppAgenda,
		Object:         c.object,
		Action:         c.action,
		FacileID:       c.id,
		Payload:        payload,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		IdempotencyKey: key,
	}
	return outbox.Enqueue(ctx, tx, c.channel, event, key)
}
