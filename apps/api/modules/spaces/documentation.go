package spaces

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var spaceID = documentation.Field{Name: "spaceID", Type: "string", Description: "Space ID."}

var unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
var internalError = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}
var spaceNotFound = documentation.Error{Status: 404, Code: "not_found", Description: "No space with that ID, or the caller is not a member."}

var Documentation = documentation.Module{
	Name: "spaces",
	Description: "Shared spaces, read-only: Registre owns their names, members and roles, " +
		"and Agenda follows it. Manage them at https://sso.facile.studio/spaces.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/spaces",
			Summary:      "List spaces",
			Description:  "Returns every space the authenticated user belongs to.",
			Auth:         "bearer token required",
			ResponseBody: []SpaceResponse{},
			Errors:       []documentation.Error{unauthenticated, internalError},
		},
		{
			Method:       "GET",
			Path:         "/spaces/{spaceID}",
			Summary:      "Return one space",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			ResponseBody: SpaceResponse{},
			Errors:       []documentation.Error{unauthenticated, spaceNotFound, internalError},
		},
		{
			Method:       "GET",
			Path:         "/spaces/{spaceID}/members",
			Summary:      "List space members",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			ResponseBody: []MemberResponse{},
			Errors:       []documentation.Error{unauthenticated, spaceNotFound, internalError},
		},
		{
			Method:  "POST",
			Path:    "/registre/sync",
			Summary: "Nudge a space sync",
			Description: "Registre posts here after any space change. The body is ignored; it only makes " +
				"Agenda pull Registre's snapshot within seconds.",
			Auth:   "none",
			Status: 202,
		},
		{
			Method:  "POST",
			Path:    "/registre/deletions",
			Summary: "Receive a GDPR deletion notice",
			Description: "Registre posts a signed user.deleted security event here when a person erases " +
				"their account. Agenda verifies the signature, deletes that person's data and answers 202; " +
				"a notice that does not verify is a 400.",
			Auth:   "signed security event",
			Status: 202,
		},
	},
}
