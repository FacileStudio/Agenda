package mcp

import (
	documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// Documentation describes the confirm page routes for the API reference. The
// MCP endpoint itself is outside /api: POST /mcp speaks the Model Context
// Protocol to any MCP client holding a Registre access token for this
// server, and /.well-known/oauth-protected-resource/mcp describes how to get
// one.
var Documentation = documentation.Module{
	Name: "mcp",
	Description: "Sensitive actions MCP clients asked for, waiting for the person to confirm them: deleting an " +
		"event. A client that cannot ask the person itself hands them a link to the pending page, which uses " +
		"these routes. The MCP endpoint is POST /mcp, outside /api.",
	Routes: pendingRoutes(),
}

// pendingRoutes documents the three confirm page routes.
func pendingRoutes() []documentation.Route {
	return mcpkit.PendingRoutes[PendingItem](mcpkit.PendingDoc{
		Auth: "session cookie or bearer token",
		List: "Open actions, newest first, with their event as it is now. state is ready, changed (the " +
			"event was edited after the request, so it cannot be confirmed) or missing.",
		Confirm: "Deletes the event, once. An event edited since the request is refused with 412 and the " +
			"action closes.",
		Dismiss: "Closes the action without running it. The event stays.",
		Changed: "The event changed after the request.",
	})
}
