package events

import (
	"context"
	"strconv"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"github.com/emersion/go-ical"
)

// seriesCtx is the parsed series an occurrence edit works on. at is the
// original start of the targeted occurrence and tzid the zone DTSTART names,
// which every time the edit writes must repeat.
type seriesCtx struct {
	evt    *schemas.Event
	cal    *ical.Calendar
	master *ical.Component
	loc    *time.Location
	tzid   string
	at     time.Time
}

type seriesEdit func(c *seriesCtx) error

// CancelOccurrence excludes one occurrence of a recurring event (an EXDATE)
// and drops any override it had. `when` is YYYY-MM-DD in the event's zone, or
// an RFC 3339 instant for a series with several occurrences a day.
func (s *Service) CancelOccurrence(ctx context.Context, userID, eventID int64, when string) (*EventResponse, error) {
	return s.editSeries(ctx, userID, eventID, when, cancelOccurrence)
}

// RestoreOccurrence undoes a cancellation or an override, returning the
// occurrence to what the series says.
func (s *Service) RestoreOccurrence(ctx context.Context, userID, eventID int64, when string) (*EventResponse, error) {
	return s.editSeries(ctx, userID, eventID, when, restoreOccurrence)
}

// UpdateOccurrence overrides one occurrence (a RECURRENCE-ID instance) with
// its own title, place, times or status, leaving the rest of the series alone.
func (s *Service) UpdateOccurrence(ctx context.Context, userID, eventID int64, when string, req *OccurrenceRequest) (*EventResponse, error) {
	if req.Status != nil {
		if err := validateOverrideStatus(*req.Status); err != nil {
			return nil, err
		}
	}
	return s.editSeries(ctx, userID, eventID, when, overrideOccurrence(req))
}

func validateOverrideStatus(status string) error {
	if status == "confirmed" || status == "tentative" {
		return nil
	}
	return errors.Invalid("occurrence status must be confirmed or tentative; use DELETE to cancel it")
}

func (s *Service) editSeries(ctx context.Context, userID, eventID int64, when string, edit seriesEdit) (*EventResponse, error) {
	evt, err := s.loadWithAccess(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}
	if err := s.checkCalendarWriteAccess(ctx, userID, evt.CalendarID); err != nil {
		return nil, err
	}
	if evt.Source != "" {
		return nil, errReadOnly
	}
	c, err := openSeries(evt, when)
	if err != nil {
		return nil, err
	}
	if err := edit(c); err != nil {
		return nil, err
	}
	return s.saveSeries(ctx, c)
}

func (s *Service) saveSeries(ctx context.Context, c *seriesCtx) (*EventResponse, error) {
	evt := c.evt
	evt.ETag = newETag()
	evt.Sequence++
	sequence := ical.NewProp(ical.PropSequence)
	sequence.Value = strconv.Itoa(evt.Sequence)
	c.master.Props.Set(sequence)
	c.master.Props.SetDateTime(ical.PropLastModified, time.Now().UTC())
	raw, err := encodeCalendar(c.cal)
	if err != nil {
		return nil, errors.Internal("failed to encode the event's calendar data", err)
	}
	evt.RawICS = raw
	if err := s.orm.WithContext(ctx).Omit("CreatedBy").Save(evt).Error; err != nil {
		return nil, errors.Internal("failed to update event", err)
	}
	s.bumpSyncToken(ctx, evt.CalendarID)
	resp := toResponse(evt)
	return &resp, nil
}
