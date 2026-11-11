package mcp

import "github.com/FacileStudio/tronc/apiref"

// DefaultPendingAuth is how the confirm page routes authenticate when
// PendingDoc.Auth is empty.
const DefaultPendingAuth = "session cookie"

// PendingDoc is the app-specific text of the confirm page routes'
// reference.
type PendingDoc struct {
	// Auth names how a request authenticates. Empty means
	// DefaultPendingAuth.
	Auth string

	// List describes GET /mcp/pending: what each action carries.
	List string

	// Confirm describes what POST /mcp/pending/{id}/confirm runs.
	Confirm string

	// Dismiss describes POST /mcp/pending/{id}/dismiss.
	Dismiss string

	// Changed describes the 412 a confirmation gets when the target changed.
	Changed string
}

// PendingRoutes documents the three routes PendingPage serves, with T the
// app's item type, for the app's apiref module.
func PendingRoutes[T any](doc PendingDoc) []apiref.Route {
	if doc.Auth == "" {
		doc.Auth = DefaultPendingAuth
	}
	unauthorized := apiref.Error{Status: 401, Code: "unauthenticated", Description: "The request carries no valid session."}
	delegated := apiref.Error{Status: 403, Code: "permission_denied", Description: "Only the person's own session can confirm actions."}
	gone := apiref.Error{Status: 404, Code: "not_found", Description: "The action ran, was dismissed or expired."}
	changed := apiref.Error{Status: 412, Code: "failed_precondition", Description: doc.Changed}
	id := []apiref.Field{{Name: "id", Type: "string", Description: "The action id from the confirm link"}}
	return []apiref.Route{
		{
			Method: "GET", Path: "/mcp/pending", Summary: "List actions waiting for confirmation",
			Description: doc.List, Auth: doc.Auth, ResponseBody: PendingList[T]{},
			Errors: []apiref.Error{unauthorized, delegated},
		},
		{
			Method: "POST", Path: "/mcp/pending/{id}/confirm", Summary: "Confirm an action",
			Description: doc.Confirm, Auth: doc.Auth, PathParams: id, RequestBody: EmptyRequest{}, ResponseBody: PendingResult{},
			Errors: []apiref.Error{unauthorized, delegated, gone, changed},
		},
		{
			Method: "POST", Path: "/mcp/pending/{id}/dismiss", Summary: "Dismiss an action",
			Description: doc.Dismiss, Auth: doc.Auth, PathParams: id, RequestBody: EmptyRequest{}, ResponseBody: PendingResult{},
			Errors: []apiref.Error{unauthorized, delegated, gone},
		},
	}
}
