package events

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
var internalError = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}

var Documentation = documentation.Module{
	Name:        "events",
	Description: "Event CRUD. Listing is scoped to a calendar; a single event is addressed by its own ID.",
	Routes: []documentation.Route{
		{
			Method:      "GET",
			Path:        "/calendars/{calendarID}/events",
			Summary:     "List events in a calendar",
			Description: "Returns the calendar's events, optionally filtered to a date range.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "calendarID", Type: "string", Description: "Calendar ID."},
			},
			ResponseBody: "[]Event",
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
			Description: "Creates an event and bumps the calendar's CalDAV sync token.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "calendarID", Type: "string", Description: "Calendar ID."},
			},
			RequestBody:  "CreateRequest",
			ResponseBody: "Event",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or an end before the start."},
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
			ResponseBody: "Event",
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
			Description:  "Updates the event and bumps its calendar's CalDAV sync token.",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{{Name: "eventID", Type: "string", Description: "Event ID."}},
			RequestBody:  "UpdateRequest",
			ResponseBody: "Event",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or an end before the start."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not write to this event's calendar."},
				{Status: 404, Code: "not_found", Description: "No event with that ID."},
				internalError,
			},
		},
		{
			Method:      "DELETE",
			Path:        "/events/{eventID}",
			Summary:     "Delete an event",
			Description: "Deletes the event and bumps its calendar's CalDAV sync token.",
			Auth:        "bearer token required",
			PathParams:  []documentation.Field{{Name: "eventID", Type: "string", Description: "Event ID."}},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not write to this event's calendar."},
				{Status: 404, Code: "not_found", Description: "No event with that ID."},
				internalError,
			},
		},
	},
}
