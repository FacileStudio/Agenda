package events

import (
	"context"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// NewEvent is one event of a batch: the calendar it lands in and its fields.
type NewEvent struct {
	CalendarID int64
	Request    CreateEventRequest
}

// CreateBatch creates every event in one transaction, so either all exist or
// none does. It validates every item and checks write access to each calendar
// before anything is saved. within runs inside the same transaction once the
// rows exist, with their ids in the order of items, so the caller can record
// them atomically; an error it returns rolls everything back and comes back
// unchanged. A nil within is allowed.
func (s *Service) CreateBatch(
	ctx context.Context, userID int64, items []NewEvent, within func(tx *gorm.DB, ids []int64) error,
) ([]EventResponse, error) {
	rows := make([]*schemas.Event, len(items))
	for i := range items {
		if err := prepareCreate(&items[i].Request); err != nil {
			return nil, err
		}
		if err := s.checkCalendarWriteAccess(ctx, userID, items[i].CalendarID); err != nil {
			return nil, err
		}
		evt, err := s.newEvent(ctx, items[i].CalendarID, &userID, &items[i].Request, "", "")
		if err != nil {
			return nil, err
		}
		rows[i] = evt
	}
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ids := make([]int64, len(rows))
		for i, evt := range rows {
			if err := tx.Create(evt).Error; err != nil {
				return errors.Internal("failed to create event", err)
			}
			if err := s.outbox.Event(tx, envelope.ActionCreated, evt, userID); err != nil {
				return errors.Internal("failed to create event", err)
			}
			ids[i] = evt.ID
		}
		if within == nil {
			return nil
		}
		return within(tx, ids)
	})
	if err != nil {
		return nil, err
	}
	return s.afterBatch(ctx, rows)
}

// afterBatch bumps the sync token of each calendar the batch touched once and
// reads the saved events back, in the order of rows.
func (s *Service) afterBatch(ctx context.Context, rows []*schemas.Event) ([]EventResponse, error) {
	touched := map[int64]bool{}
	ids := make([]int64, len(rows))
	for i, evt := range rows {
		ids[i] = evt.ID
		if !touched[evt.CalendarID] {
			touched[evt.CalendarID] = true
			s.bumpSyncToken(ctx, evt.CalendarID)
		}
	}
	var saved []schemas.Event
	if err := s.orm.WithContext(ctx).Preload("CreatedBy").Where("id IN ?", ids).Find(&saved).Error; err != nil {
		return nil, errors.Internal("failed to read the events back", err)
	}
	byID := make(map[int64]schemas.Event, len(saved))
	for _, evt := range saved {
		byID[evt.ID] = evt
	}
	ordered := make([]schemas.Event, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, byID[id])
	}
	return s.toResponses(ctx, ordered)
}
