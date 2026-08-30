package spaces

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var spaceID = documentation.Field{Name: "spaceID", Type: "string", Description: "Space ID."}
var memberUserID = documentation.Field{Name: "userID", Type: "string", Description: "ID of the member's user."}

var unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
var internalError = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}
var spaceNotFound = documentation.Error{Status: 404, Code: "not_found", Description: "No space with that ID, or the caller is not a member."}
var notPermitted = documentation.Error{Status: 403, Code: "permission_denied", Description: "The caller's role does not allow this."}

var Documentation = documentation.Module{
	Name:        "spaces",
	Description: "Shared workspaces: membership, roles, and the calendars they group.",
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
			Method:       "POST",
			Path:         "/spaces",
			Summary:      "Create a space",
			Description:  "Creates a space with the caller as its owner.",
			Auth:         "bearer token required",
			RequestBody:  CreateSpaceRequest{},
			ResponseBody: SpaceResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or missing name."},
				unauthenticated, internalError,
			},
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
			Method:       "PUT",
			Path:         "/spaces/{spaceID}",
			Summary:      "Update a space",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			RequestBody:  UpdateSpaceRequest{},
			ResponseBody: SpaceResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body."},
				unauthenticated, notPermitted, spaceNotFound, internalError,
			},
		},
		{
			Method:       "DELETE",
			Path:         "/spaces/{spaceID}",
			Summary:      "Delete a space",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			ResponseBody: OkResponse{},
			Errors:       []documentation.Error{unauthenticated, notPermitted, spaceNotFound, internalError},
		},
		{
			Method:       "POST",
			Path:         "/spaces/{spaceID}/leave",
			Summary:      "Leave a space",
			Description:  "Removes the authenticated user's own membership.",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			ResponseBody: OkResponse{},
			Errors:       []documentation.Error{unauthenticated, notPermitted, spaceNotFound, internalError},
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
			Method:       "POST",
			Path:         "/spaces/{spaceID}/members",
			Summary:      "Add a member",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID},
			RequestBody:  AddMemberRequest{},
			ResponseBody: OkResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or unknown role."},
				unauthenticated, notPermitted, spaceNotFound,
				{Status: 409, Code: "already_exists", Description: "That user is already a member."},
				internalError,
			},
		},
		{
			Method:       "DELETE",
			Path:         "/spaces/{spaceID}/members/{userID}",
			Summary:      "Remove a member",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID, memberUserID},
			ResponseBody: OkResponse{},
			Errors: []documentation.Error{
				unauthenticated, notPermitted,
				{Status: 404, Code: "not_found", Description: "No such space or member."},
				internalError,
			},
		},
		{
			Method:       "PUT",
			Path:         "/spaces/{spaceID}/members/{userID}/role",
			Summary:      "Change a member's role",
			Auth:         "bearer token required",
			PathParams:   []documentation.Field{spaceID, memberUserID},
			RequestBody:  UpdateRoleRequest{},
			ResponseBody: OkResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or unknown role."},
				unauthenticated, notPermitted,
				{Status: 404, Code: "not_found", Description: "No such space or member."},
				internalError,
			},
		},
	},
}
