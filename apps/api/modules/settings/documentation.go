package settings

import documentation "api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "settings",
	Description: "Application settings management.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/settings",
			Summary:      "Return current user settings",
			Description:  "Returns the authenticated user's settings.",
			Auth:         "bearer token required",
			ResponseBody: "SettingsResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PUT",
			Path:         "/settings",
			Summary:      "Update current user settings",
			Description:  "Updates the authenticated user's settings.",
			Auth:         "bearer token required",
			RequestBody:  "UpdateRequest",
			ResponseBody: "SettingsResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
