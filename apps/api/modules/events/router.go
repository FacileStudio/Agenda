package events

import (
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the authenticated event CRUD endpoints.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	mountRoutes(router, newController(service), middleware.RequireAuth(authService))
}

func mountRoutes(router chi.Router, c *controller, guard func(http.Handler) http.Handler) {
	router.Route("/calendars/{calendarID}/events", func(r chi.Router) {
		r.Use(guard)
		r.Get("/", c.list)
		r.Post("/", c.create)
	})
	router.Route("/events/{eventID}", func(r chi.Router) {
		r.Use(guard)
		r.Get("/", c.get)
		r.Put("/", c.update)
		r.Delete("/", c.delete)
		r.Put("/occurrences/{when}", c.updateOccurrence)
		r.Delete("/occurrences/{when}", c.cancelOccurrence)
		r.Post("/occurrences/{when}/restore", c.restoreOccurrence)
	})
}
