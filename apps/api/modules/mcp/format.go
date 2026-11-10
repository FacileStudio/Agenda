package mcp

import (
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// Description limits of the two read formats.
const (
	conciseDescription  = 300
	detailedDescription = 20_000
)

// eventZone is the zone an event is anchored in, or fallback.
func eventZone(e events.EventResponse, fallback *time.Location) *time.Location {
	if e.Timezone == "" {
		return fallback
	}
	if loc, err := time.LoadLocation(e.Timezone); err == nil {
		return loc
	}
	return fallback
}

// when renders an event bound: a date for an all-day event, else RFC 3339
// in loc.
func when(t time.Time, allDay bool, loc *time.Location) string {
	if allDay {
		return t.In(loc).Format(dateLayout)
	}
	return stamp(t, loc)
}

// summarize renders one event of a list in loc.
func summarize(e events.EventResponse, loc *time.Location, full bool) eventSummary {
	shown := loc
	if e.IsAllDay {
		shown = eventZone(e, loc)
	}
	s := eventSummary{
		EventID: e.ID, CalendarID: e.CalendarID, Title: e.Title, AllDay: e.IsAllDay, Location: e.Location,
		Start: when(e.StartAt, e.IsAllDay, shown), End: when(e.EndAt, e.IsAllDay, shown),
	}
	if e.OccurrenceStart != nil && e.RecurrenceRule != "" {
		s.OccurrenceStart = stamp(*e.OccurrenceStart, loc)
	}
	if full {
		description, _ := mcpkit.Clip(e.Description, conciseDescription)
		s.eventDetail = eventDetail{
			Description: description, Status: e.Status, Timezone: e.Timezone, RecurrenceRule: e.RecurrenceRule,
			ConferenceURL: e.ConferenceURL, Attendees: attendeeEmails(e.Attendees),
		}
	}
	return s
}

// attendeeEmails lists the invitees' addresses.
func attendeeEmails(list []events.Attendee) []string {
	emails := make([]string, 0, len(list))
	for _, a := range list {
		emails = append(emails, a.Email)
	}
	return emails
}

// fullEvent renders one event with everything a model may need.
func fullEvent(e events.EventResponse) eventOutput {
	loc := eventZone(e, time.UTC)
	description, cut := mcpkit.Clip(e.Description, detailedDescription)
	out := eventOutput{
		EventID: e.ID, CalendarID: e.CalendarID, Title: e.Title, AllDay: e.IsAllDay, Timezone: loc.String(),
		Start: when(e.StartAt, e.IsAllDay, loc), End: when(e.EndAt, e.IsAllDay, loc), Status: e.Status,
		Location: e.Location, Description: description, Truncated: cut, RecurrenceRule: e.RecurrenceRule,
		ConferenceURL: e.ConferenceURL, Exceptions: len(e.Exceptions), ReadOnly: e.Source != "",
	}
	for _, a := range e.Attendees {
		out.Attendees = append(out.Attendees, attendee{Email: a.Email, Name: a.Name, Response: a.Response})
	}
	for _, r := range e.Reminders {
		out.Reminders = append(out.Reminders, r.MinutesBefore)
	}
	return out
}
