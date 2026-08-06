package auth

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "auth",
	Description: "Authentication routes.",
	Routes: []documentation.Route{
		{
			Method:       "POST",
			Path:         "/auth/register",
			Summary:      "Register a new user",
			Description:  "Creates a user account and returns an auth token.",
			Auth:         "public",
			RequestBody:  "RegisterRequest",
			ResponseBody: "AuthResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid registration input."},
				{Status: 409, Code: "already_exists", Description: "A user with the same email already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/login",
			Summary:      "Authenticate a user",
			Description:  "Authenticates credentials and returns an auth token.",
			Auth:         "public",
			RequestBody:  "LoginRequest",
			ResponseBody: "AuthResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid login input."},
				{Status: 401, Code: "unauthenticated", Description: "Email or password is invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/auth/config",
			Summary:      "Return the auth configuration",
			Description:  "Reports which sign-in methods this deployment offers, so the client can hide password fields under SSO_ONLY.",
			ResponseBody: "ConfigResponse",
			Errors: []documentation.Error{
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:      "POST",
			Path:        "/auth/logout",
			Summary:     "Log out",
			Description: "Revokes the caller's session token.",
			Auth:        "bearer token required",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
