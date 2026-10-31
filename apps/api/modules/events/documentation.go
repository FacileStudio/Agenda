package events

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
var internalError = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}

var occurrencePathParams = []documentation.Field{
	{Name: "eventID", Type: "string", Description: "Event ID of the recurring series."},
	{Name: "when", Type: "string", Description: "Occurrence date YYYY-MM-DD in the event's zone, or its start as an RFC 3339 instant."},
}

var occurrenceErrors = []documentation.Error{
	{Status: 400, Code: "invalid_argument", Description: "The event does not recur, there is no occurrence on that date, or the body is invalid."},
	unauthenticated,
	{Status: 403, Code: "permission_denied", Description: "The caller may not write to this event's calendar, or the event is mirrored from Opus."},
	{Status: 404, Code: "not_found", Description: "No event with that ID."},
	internalError,
}

var Documentation = documentation.Module{
	Name:        "events",
	Description: "Event CRUD. Listing is scoped to a calendar; a single event is addressed by its own ID. Events carry attendees, reminders, a color, the zone they are anchored in and, for recurring events, per-occurrence exceptions.",
	Routes: []documentation.Route{
		{
			Method:      "GET",
			Path:        "/calendars/{calendarID}/events",
			Summary:     "List events in a calendar",
			Description: "Returns the calendar's events, optionally filtered to a date range (`from`, `to`, RFC 3339). With both bounds set, a recurring event comes back once per occurrence in the range, with the series ID; `tz` (IANA zone) anchors series that carry no zone of their own.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "calendarID", Type: "string", Description: "Calendar ID."},
			},
			ResponseBody: []EventResponse{},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 404, Code: "not_found", Description: "No calendar with that ID, or it is not shared with the caller."},
				internalError,
			},
		},
		{
			Method:      "POST",
			Path:        "/calendars/{calendarID}/events",
			Summary:     "Create an event",
			Description: "Creates an event and bumps the calendar's CalDAV sync token. `status` must be confirmed, tentative or cancelled and `timezone` an IANA name; an unknown value is rejected. `attendees` (email, name, role required or optional, response), `reminders` (minutes_before, method display, audio or email) and `color` (#RRGGBB) are optional. Agenda records invitations and reminders in the event's ICS for CalDAV clients; it sends no mail.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "calendarID", Type: "string", Description: "Calendar ID."},
			},
			RequestBody:  CreateEventRequest{},
			ResponseBody: EventResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body, an end before the start, or an unknown status, timezone, color, attendee or reminder."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not write to this calendar."},
				{Status: 404, Code: "not_found", Description: "No calendar with that ID."},
				internalError,
			},
		},
		{
			Method:       "GET",
			Path:         "/events/{eventID}",
			Summary:      "Return one event",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{{Name: "eventID", Type: "string", Description: "Event ID."}},
			ResponseBody: EventResponse{},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 404, Code: "not_found", Description: "No event with that ID, or its calendar is not shared with the caller."},
				internalError,
			},
		},
		{
			Method:       "PUT",
			Path:         "/events/{eventID}",
			Summary:      "Update an event",
			Description:  "Updates the event and bumps its calendar's CalDAV sync token. The body replaces the event, except that an empty `timezone` keeps the stored zone and an omitted `attendees`, `reminders` or `color` keeps the stored value (an empty list or string clears it). Changing the start, rule, zone or all-day flag of a recurring event discards its occurrence exceptions.",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{{Name: "eventID", Type: "string", Description: "Event ID."}},
			RequestBody:  UpdateEventRequest{},
			ResponseBody: EventResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body, an end before the start, or an unknown status, timezone, color, attendee or reminder."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not write to this event's calendar."},
				{Status: 404, Code: "not_found", Description: "No event with that ID."},
				internalError,
			},
		},
		{
			Method:       "DELETE",
			Path:         "/events/{eventID}",
			Summary:      "Delete an event",
			Description:  "Deletes the event and bumps its calendar's CalDAV sync token.",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{{Name: "eventID", Type: "string", Description: "Event ID."}},
			ResponseBody: OkResponse{},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not write to this event's calendar."},
				{Status: 404, Code: "not_found", Description: "No event with that ID."},
				internalError,
			},
		},
		{
			Method:       "PUT",
			Path:         "/events/{eventID}/occurrences/{when}",
			Summary:      "Override one occurrence of a recurring event",
			Description:  "Gives a single occurrence its own title, description, location, times or status (a RECURRENCE-ID instance in the ICS) and leaves the series alone. `when` is YYYY-MM-DD in the event's zone, or an RFC 3339 instant when the series recurs more than once a day. Omitted fields keep the series value. A cancelled occurrence must be restored first.",
			Auth:         "bearer token required",
			PathParams:   occurrencePathParams,
			RequestBody:  OccurrenceRequest{},
			ResponseBody: EventResponse{},
			Errors:       occurrenceErrors,
		},
		{
			Method:       "DELETE",
			Path:         "/events/{eventID}/occurrences/{when}",
			Summary:      "Cancel one occurrence of a recurring event",
			Description:  "Excludes the occurrence from the series (an EXDATE in the ICS) and drops any override it had. Repeating the call is harmless. Returns the series with its updated `exceptions`.",
			Auth:         "bearer token required",
			PathParams:   occurrencePathParams,
			ResponseBody: EventResponse{},
			Errors:       occurrenceErrors,
		},
		{
			Method:       "POST",
			Path:         "/events/{eventID}/occurrences/{when}/restore",
			Summary:      "Restore one occurrence of a recurring event",
			Description:  "Removes the cancellation or the override of the occurrence so it follows the series again.",
			Auth:         "bearer token required",
			PathParams:   occurrencePathParams,
			ResponseBody: EventResponse{},
			Errors:       occurrenceErrors,
		},
	},
}
