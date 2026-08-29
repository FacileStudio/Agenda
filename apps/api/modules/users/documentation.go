package users

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "users",
	Description: "User listing plus current-user retrieval and update routes.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/users",
			Summary:      "List users",
			Description:  "Returns all authenticated users with profile metadata.",
			Auth:         "bearer token required",
			ResponseBody: "ListResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/users/me",
			Summary:      "Return the current user",
			Description:  "Returns the authenticated user with profile metadata.",
			Auth:         "bearer token required",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:  "PATCH",
			Path:    "/users/me",
			Summary: "Update the current user",
			Description: "Updates the authenticated user's name, email, and/or password. " +
				"Sending password alone adds a first password to an account that has none; " +
				"replacing an existing one also requires current_password, and rotates the session.",
			Auth:         "bearer token required",
			RequestBody:  "UpdateRequest",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body, invalid update input, or current_password missing."},
				{Status: 401, Code: "unauthenticated", Description: "Auth is missing or invalid, or current_password is wrong."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 409, Code: "already_exists", Description: "A user with the same email already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:      "GET",
			Path:        "/users/{id}",
			Summary:     "Return one user",
			Description: "Returns a single user's profile metadata by ID.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "id", Type: "string", Description: "User ID."},
			},
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "No user with that ID."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/users/me/avatar",
			Summary:      "Upload the current user's avatar",
			Description:  "Stores a new avatar file for the authenticated user and returns the updated profile.",
			Auth:         "bearer token required",
			RequestBody:  "multipart/form-data with avatar file",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Missing file or unsupported image type."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 413, Code: "resource_exhausted", Description: "Avatar file is too large."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "DELETE",
			Path:         "/users/me/avatar",
			Summary:      "Remove the current user's avatar",
			Description:  "Deletes the stored avatar file and returns the updated profile.",
			Auth:         "bearer token required",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/users/me/api-token",
			Summary:      "Report whether an API token exists",
			Description:  "Returns the current token's metadata without ever returning the token itself.",
			Auth:         "bearer token required",
			ResponseBody: "ApiTokenStatusResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/users/me/api-token",
			Summary:      "Create an API token",
			Description:  "Issues a new API token and returns it once — it cannot be read back afterwards.",
			Auth:         "bearer token required",
			RequestBody:  "CreateApiTokenRequest",
			ResponseBody: "ApiTokenResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or missing token name."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "DELETE",
			Path:         "/users/me/api-token",
			Summary:      "Revoke the current API token",
			Description:  "Deletes the authenticated user's API token.",
			Auth:         "bearer token required",
			ResponseBody: "DeletedResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
