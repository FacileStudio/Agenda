package auth

import (
	documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"
	"github.com/FacileStudio/porte"
)

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
			RequestBody:  RegisterRequest{},
			ResponseBody: AuthResponse{},
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
			RequestBody:  LoginRequest{},
			ResponseBody: AuthResponse{},
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
			ResponseBody: porte.ConfigResponse{},
			Errors: []documentation.Error{
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/logout",
			Summary:      "Log out",
			Description:  "Revokes the caller's session token.",
			Auth:         "bearer token required",
			ResponseBody: porte.LogoutResponse{},
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc",
			Summary:     "Start the SSO login",
			Description: "Redirects to the identity provider with PKCE and a nonce. Registered only when OIDC_ISSUER is set. Add ?flow=cli for the CLI handoff.",
			Auth:        "public",
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc/callback",
			Summary:     "Complete the SSO login",
			Description: "Verifies the ID token, upserts the account and sets the session cookie. A failure is a redirect to the login page carrying ?error=, not a JSON body — a refusal and a success are both a 302, so read Location and not the status.",
			Auth:        "public",
		},
		{
			Method:       "POST",
			Path:         "/auth/oidc/exchange",
			Summary:      "Exchange a CLI login code for a token",
			Description:  "Consumes the one-time code handed to a ?flow=cli login. The code is single use: a replay finds nothing.",
			Auth:         "public",
			RequestBody:  porte.ExchangeRequest{},
			ResponseBody: porte.ExchangeResponse{},
		},
		{
			Method:       "POST",
			Path:         "/auth/sync-profile",
			Summary:      "Refresh the profile from the provider",
			Description:  "Calls UserInfo with the stored refresh token and updates the name and photo. Rate-limited server-side.",
			Auth:         "session cookie or bearer token required",
			ResponseBody: porte.SyncProfileResponse{},
		},
		{
			Method:       "POST",
			Path:         "/auth/backchannel-logout",
			Summary:      "Revoke sessions on the provider's behalf",
			Description:  "Called by the identity provider, not the client. Validates the logout token and deletes that user's sessions.",
			Auth:         "signed logout token",
			RequestBody:  BackchannelLogoutRequest{},
			ResponseBody: porte.LogoutResponse{},
		},
	},
}
