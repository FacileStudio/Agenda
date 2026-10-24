package events

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// UpsertExternal creates or updates the event mirrored from another app,
// keyed by (source, externalID). When the mirror already exists in a
// different calendar it is moved. It reuses the normal create and update
// paths, so raw ICS and sync tokens behave exactly as for a user edit.
func (s *Service) UpsertExternal(ctx context.Context, calendarID int64, source, externalID string, req *CreateEventRequest) error {
	if err := validateEventFields(req.Title, req.StartAt, req.EndAt); err != nil {
		return err
	}
	var evt schemas.Event
	err := s.orm.WithContext(ctx).Where("source = ? AND external_id = ?", source, externalID).First(&evt).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		_, err = s.insertEvent(ctx, calendarID, nil, req, source, externalID)
		return err
	}
	if err != nil {
		return errors.Internal("failed to look up mirrored event", err)
	}
	_, err = s.applyUpdate(ctx, &evt, &UpdateEventRequest{
		CalendarID:         calendarID,
		Title:              req.Title,
		Description:        req.Description,
		Location:           req.Location,
		StartAt:            req.StartAt,
		EndAt:              req.EndAt,
		IsAllDay:           req.IsAllDay,
		RecurrenceRule:     req.RecurrenceRule,
		Status:             req.Status,
		ConferenceURL:      req.ConferenceURL,
		ConferenceProvider: req.ConferenceProvider,
		Timezone:           req.Timezone,
	})
	return err
}

// DeleteExternal removes the mirrored event keyed by (source, externalID) and
// bumps its calendar's sync token. A missing mirror is not an error.
func (s *Service) DeleteExternal(ctx context.Context, source, externalID string) error {
	var evt schemas.Event
	err := s.orm.WithContext(ctx).Where("source = ? AND external_id = ?", source, externalID).First(&evt).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errors.Internal("failed to look up mirrored event", err)
	}
	if err := s.orm.WithContext(ctx).Delete(&evt).Error; err != nil {
		return errors.Internal("failed to delete mirrored event", err)
	}
	s.bumpSyncToken(ctx, evt.CalendarID)
	return nil
}

func allDayEnd(end time.Time) time.Time {
	end = end.UTC()
	if end.Equal(end.Truncate(24 * time.Hour)) {
		return end
	}
	return end.Truncate(24*time.Hour).AddDate(0, 0, 1)
}
