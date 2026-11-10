package caldav

import (
	"context"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"gorm.io/gorm"
)

// createCalendar inserts cal and queues calendar.created in one transaction.
func (b *Backend) createCalendar(ctx context.Context, cal *schemas.Calendar, actorID int64) error {
	return b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cal).Error; err != nil {
			return err
		}
		return b.outbox.Calendar(tx, envelope.ActionCreated, cal, actorID)
	})
}

// writeEvent runs change and queues the event's publication in one
// transaction, so a CalDAV write is published if and only if it commits.
func (b *Backend) writeEvent(ctx context.Context, action envelope.Action, evt *schemas.Event, actorID int64, change func(*gorm.DB) error) error {
	return b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := change(tx); err != nil {
			return err
		}
		return b.outbox.Event(tx, action, evt, actorID)
	})
}

// actorOf is the id of the person making the request, or 0 without one.
func actorOf(ctx context.Context) int64 {
	if user := userFromContext(ctx); user != nil {
		return user.ID
	}
	return 0
}
