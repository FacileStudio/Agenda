package live

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "live",
	Description: "Server-Sent Events telling the web client when something it can see changed, whoever changed it: another user, CalDAV, MCP, the Opus mirror or the database itself.",
	Routes: []documentation.Route{
		{
			Method:      "GET",
			Path:        "/live",
			Summary:     "Stream change notices",
			Description: "A `text/event-stream`. Each `change` event carries JSON: `calendar_id` for an event change in a calendar the caller can read, plus `calendar: true` when the calendar itself or who can read it changed, `activity: true` when the caller's own activities changed, or `resync: true` when notices may have been missed. Frames carry no content; the client refetches. A comment line arrives every 25 seconds, and the server closes the stream after 30 minutes so the browser reconnects through the session check.",
			Auth:        "session or bearer token required",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "No valid session or token."},
			},
		},
	},
}
