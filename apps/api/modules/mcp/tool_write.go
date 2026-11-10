package mcp

import (
	"context"
	"errors"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// errNoPersonal is the refusal for a create without a calendar when the
// user has no personal calendar.
var errNoPersonal = errors.New("the user has no personal calendar. Pass calendar_id; call agenda_list_calendars for the ids")

// createEvent is agenda_create_event.
func (t *toolset) createEvent(ctx context.Context, req *sdk.CallToolRequest, in createInput) (*sdk.CallToolResult, writeOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, writeOutput{}, err
	}
	loc, err := zone(in.Timezone)
	if err != nil {
		return nil, writeOutput{}, err
	}
	start, end, err := span(in.Start, in.End, in.AllDay, loc)
	if err != nil {
		return nil, writeOutput{}, err
	}
	calendarID, err := t.targetCalendar(ctx, userID, in.CalendarID)
	if err != nil {
		return nil, writeOutput{}, err
	}
	created, err := t.events.CreateEvent(ctx, userID, calendarID, &events.CreateEventRequest{
		Title: in.Title, Description: in.Description, Location: in.Location, StartAt: start, EndAt: end,
		IsAllDay: in.AllDay, RecurrenceRule: in.RecurrenceRule, Status: in.Status, ConferenceURL: in.ConferenceURL,
		Timezone: loc.String(), Attendees: attendees(in.Attendees),
	})
	if err != nil {
		return nil, writeOutput{}, mcpkit.Failure(err, hintCalendar)
	}
	return nil, writeOutput{Status: "created", Message: "The event is in the calendar.", Event: fullEvent(*created)}, nil
}

// targetCalendar is the calendar a create writes to: the one asked for, or
// the user's personal calendar.
func (t *toolset) targetCalendar(ctx context.Context, userID, asked int64) (int64, error) {
	if asked != 0 {
		return asked, nil
	}
	list, err := t.calendars.ListCalendars(ctx, userID, nil)
	if err != nil {
		return 0, mcpkit.Failure(err, "")
	}
	for _, c := range list {
		if c.IsPersonal && c.OwnerID == userID {
			return c.ID, nil
		}
	}
	return 0, errNoPersonal
}

// attendees turns invitee emails into attendee records; nil stays nil.
func attendees(emails []string) []events.Attendee {
	if emails == nil {
		return nil
	}
	list := make([]events.Attendee, len(emails))
	for i, email := range emails {
		list[i] = events.Attendee{Email: email}
	}
	return list
}

// updateEvent is agenda_update_event.
func (t *toolset) updateEvent(ctx context.Context, req *sdk.CallToolRequest, in updateInput) (*sdk.CallToolResult, writeOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, writeOutput{}, err
	}
	current, err := t.events.GetEvent(ctx, userID, in.EventID)
	if err != nil {
		return nil, writeOutput{}, mcpkit.Failure(err, hintEvent)
	}
	change, err := merged(in, *current)
	if err != nil {
		return nil, writeOutput{}, err
	}
	updated, err := t.events.UpdateEvent(ctx, userID, in.EventID, change)
	if err != nil {
		return nil, writeOutput{}, mcpkit.Failure(err, hintCalendar)
	}
	return nil, writeOutput{Status: "updated", Message: "The event is updated.", Event: fullEvent(*updated)}, nil
}

// merged is the full update request: the event as it is, with the fields
// the input sets.
func merged(in updateInput, e events.EventResponse) (*events.UpdateEventRequest, error) {
	zoneName := e.Timezone
	if in.Timezone != nil {
		zoneName = *in.Timezone
	}
	loc, err := zone(zoneName)
	if err != nil {
		return nil, err
	}
	next, err := reschedule(in, bounds{start: e.StartAt, end: e.EndAt, allDay: e.IsAllDay}, loc)
	if err != nil {
		return nil, err
	}
	out := &events.UpdateEventRequest{
		CalendarID: or(in.CalendarID, e.CalendarID), Title: or(in.Title, e.Title), Description: or(in.Description, e.Description),
		Location: or(in.Location, e.Location), StartAt: next.start, EndAt: next.end, IsAllDay: next.allDay,
		RecurrenceRule: or(in.RecurrenceRule, e.RecurrenceRule), Status: or(in.Status, e.Status),
		ConferenceURL: or(in.ConferenceURL, e.ConferenceURL), ConferenceProvider: e.ConferenceProvider,
		Timezone: or(in.Timezone, ""),
	}
	if in.Attendees != nil {
		out.Attendees = attendees(*in.Attendees)
		if out.Attendees == nil {
			out.Attendees = []events.Attendee{}
		}
	}
	return out, nil
}

// or is *value when set, else fallback.
func or[T any](value *T, fallback T) T {
	if value != nil {
		return *value
	}
	return fallback
}
