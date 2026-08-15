package events

import (
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the authenticated event CRUD endpoints.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	c := newController(service)
	router.Route("/calendars/{calendarID}/events", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", c.list)
		r.Post("/", c.create)
	})
	router.Route("/events/{eventID}", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", c.get)
		r.Put("/", c.update)
		r.Delete("/", c.delete)
	})
}
