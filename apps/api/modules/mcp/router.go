package mcp

import (
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the MCP endpoint at the resource's path and its RFC
// 9728 metadata on router, the root router outside /api.
func RegisterRoutes(router chi.Router, service *Service) {
	service.server.Mount(router)
}

// RegisterPendingRoutes mounts the confirm page's API under /mcp/pending on
// the /api router, for the signed-in person.
func RegisterPendingRoutes(router chi.Router, service *Service, authService middleware.Authenticator) {
	router.Route("/mcp/pending", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", service.handleList)
		r.Post("/{id}/confirm", service.handleConfirm)
		r.Post("/{id}/dismiss", service.handleDismiss)
	})
}
