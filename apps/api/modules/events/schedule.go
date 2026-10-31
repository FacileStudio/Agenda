package events

import (
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// updateZone is the zone an update anchors the event in: the one the request
// names, else the one already stored. Without this fallback a client that
// omits the zone would silently re-anchor a recurring series to UTC.
func updateZone(evt *schemas.Event, req *UpdateEventRequest) string {
	if req.Timezone != "" {
		return req.Timezone
	}
	master, _ := splitComponents(evt.RawICS)
	return effectiveTimezone(evt, master)
}

// scheduleChanged reports whether an update moves the series itself. Only
// then do the occurrence exceptions, which are keyed to the old schedule,
// stop making sense.
func scheduleChanged(evt *schemas.Event, req *UpdateEventRequest) bool {
	master, _ := splitComponents(evt.RawICS)
	return evt.RecurrenceRule != req.RecurrenceRule ||
		!evt.StartAt.Equal(req.StartAt) ||
		evt.IsAllDay != req.IsAllDay ||
		effectiveTimezone(evt, master) != updateZone(evt, req)
}

type updateSnapshot struct {
	raw        string
	keepSeries bool
	zone       string
}

func snapshotOf(evt *schemas.Event, req *UpdateEventRequest) updateSnapshot {
	return updateSnapshot{raw: evt.RawICS, keepSeries: !scheduleChanged(evt, req), zone: updateZone(evt, req)}
}

func renderNew(evt *schemas.Event, req *CreateEventRequest) error {
	if req.Color != nil {
		evt.Color = *req.Color
	}
	x := extras{attendees: req.Attendees, reminders: req.Reminders}
	raw, err := decorate(buildRawICS(evt, requestLocation(req.Timezone)), "", x, false)
	if err != nil {
		return errors.Internal("failed to render the event's calendar data", err)
	}
	evt.RawICS = raw
	return nil
}

func renderUpdated(evt *schemas.Event, req *UpdateEventRequest, prev updateSnapshot) error {
	evt.Timezone = prev.zone
	if req.Color != nil {
		evt.Color = *req.Color
	}
	x := extras{attendees: req.Attendees, reminders: req.Reminders}
	raw, err := decorate(buildRawICS(evt, requestLocation(prev.zone)), prev.raw, x, prev.keepSeries)
	if err != nil {
		return errors.Internal("failed to render the event's calendar data", err)
	}
	evt.RawICS = raw
	return nil
}

func validateEnums(status, tz string, attendees []Attendee, reminders []Reminder, color *string) (extras, error) {
	if err := validateStatus(status); err != nil {
		return extras{}, err
	}
	if err := validateTimezone(tz); err != nil {
		return extras{}, err
	}
	return prepareExtras(attendees, reminders, color)
}
