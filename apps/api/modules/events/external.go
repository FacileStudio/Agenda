package events

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// External names an event mirrored from another app: the app's source tag,
// the object's id there and the link back to it.
type External struct {
	Source string
	ID     string
	URL    string
}

// UpsertExternal creates or updates the event mirrored from another app,
// keyed by (source, id). When the mirror already exists in a different
// calendar it is moved. It reuses the normal create and update paths, so raw
// ICS and sync tokens behave exactly as for a user edit, then stores the link
// back to the object.
func (s *Service) UpsertExternal(ctx context.Context, calendarID int64, ext External, req *CreateEventRequest) error {
	if err := validateEventFields(req.Title, req.StartAt, req.EndAt); err != nil {
		return err
	}
	if err := s.upsertExternal(ctx, calendarID, ext, req); err != nil {
		return err
	}
	err := s.orm.WithContext(ctx).Model(&schemas.Event{}).
		Where("source = ? AND external_id = ? AND external_url <> ?", ext.Source, ext.ID, ext.URL).
		Update("external_url", ext.URL).Error
	if err != nil {
		return errors.Internal("failed to store the mirrored event link", err)
	}
	return nil
}

func (s *Service) upsertExternal(ctx context.Context, calendarID int64, ext External, req *CreateEventRequest) error {
	var evt schemas.Event
	err := s.orm.WithContext(ctx).Where("source = ? AND external_id = ?", ext.Source, ext.ID).First(&evt).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		_, err = s.insertEvent(ctx, calendarID, nil, req, ext.Source, ext.ID)
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
	}, 0)
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
