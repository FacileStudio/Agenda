package activities

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "activities",
	Description: "The caller's own Sablier time sessions and Echo calls, received from the Antenne pool. They are not events: CalDAV, space calendars and other users never see them.",
	Routes: []documentation.Route{
		{
			Method:      "GET",
			Path:        "/activities",
			Summary:     "List the caller's sessions and calls",
			Description: "Returns the caller's activities overlapping the range, oldest first. `end_at` is null while a session or call is running.",
			Auth:        "session or bearer token required",
			QueryParams: []documentation.Field{
				{Name: "from", Type: "string", Description: "Range start, RFC 3339."},
				{Name: "to", Type: "string", Description: "Range end, RFC 3339, at most 100 days after from."},
			},
			ResponseBody: []ActivityResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "from or to is missing, unreadable, or the range is empty or longer than 100 days."},
				{Status: 401, Code: "unauthenticated", Description: "No valid session or token."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
