package calendars

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var calendarID = documentation.Field{Name: "calendarID", Type: "string", Description: "Calendar ID."}

var unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
var internalError = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}
var calendarNotFound = documentation.Error{Status: 404, Code: "not_found", Description: "No calendar with that ID, or it is not shared with the caller."}

var Documentation = documentation.Module{
	Name:        "calendars",
	Description: "Calendar CRUD plus sharing and member management.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/calendars",
			Summary:      "List calendars",
			Description:  "Returns every calendar the authenticated user owns or has been given access to.",
			Auth:         "bearer token required",
			ResponseBody: "[]Calendar",
			Errors:       []documentation.Error{unauthenticated, internalError},
		},
		{
			Method:       "POST",
			Path:         "/calendars",
			Summary:      "Create a calendar",
			Auth:         "bearer token required",
			RequestBody:  "CreateRequest",
			ResponseBody: "Calendar",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or missing name."},
				unauthenticated, internalError,
			},
		},
		{
			Method:       "GET",
			Path:         "/calendars/{calendarID}",
			Summary:      "Return one calendar",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{calendarID},
			ResponseBody: "Calendar",
			Errors:       []documentation.Error{unauthenticated, calendarNotFound, internalError},
		},
		{
			Method:       "PUT",
			Path:         "/calendars/{calendarID}",
			Summary:      "Update a calendar",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{calendarID},
			RequestBody:  "UpdateRequest",
			ResponseBody: "Calendar",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not modify this calendar."},
				calendarNotFound, internalError,
			},
		},
		{
			Method:      "DELETE",
			Path:        "/calendars/{calendarID}",
			Summary:     "Delete a calendar",
			Description: "Deletes the calendar and every event it holds.",
			Auth:        "bearer token required",
			PathParams:  []documentation.Field{calendarID},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "Only the owner may delete a calendar."},
				calendarNotFound, internalError,
			},
		},
		{
			Method:       "GET",
			Path:         "/calendars/{calendarID}/members",
			Summary:      "List calendar members",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{calendarID},
			ResponseBody: "[]CalendarMember",
			Errors:       []documentation.Error{unauthenticated, calendarNotFound, internalError},
		},
		{
			Method:       "POST",
			Path:         "/calendars/{calendarID}/members",
			Summary:      "Share a calendar",
			Description:  "Grants another user access to the calendar at the requested role.",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{calendarID},
			RequestBody:  "ShareRequest",
			ResponseBody: "CalendarMember",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or unknown role."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not share this calendar."},
				calendarNotFound,
				{Status: 409, Code: "already_exists", Description: "That user already has access."},
				internalError,
			},
		},
		{
			Method:     "DELETE",
			Path:       "/calendars/{calendarID}/members/{memberID}",
			Summary:    "Revoke a member's access",
			Auth:       "bearer token required",
			PathParams: []documentation.Field{calendarID, {Name: "memberID", Type: "string", Description: "Calendar member ID."}},
			Errors: []documentation.Error{
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "The caller may not modify this calendar's members."},
				{Status: 404, Code: "not_found", Description: "No such calendar or member."},
				internalError,
			},
		},
	},
}
