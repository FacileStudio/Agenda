// Package live serves GET /api/live, the Server-Sent Events stream that tells
// a signed-in browser when something it can see has changed.
package live

import (
	"log/slog"

	"github.com/FacileStudio/Agenda/apps/api/internal/live"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the stream behind the session guard every other
// route uses.
func RegisterRoutes(router chi.Router, hub *live.Hub, access live.Access, authService *auth.Service, logger *slog.Logger) {
	c := &controller{hub: hub, access: access, logger: logger}
	router.With(authService.RequireAuth).Get("/live", c.stream)
}
