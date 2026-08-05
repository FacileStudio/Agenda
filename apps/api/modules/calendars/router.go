package calendars

import (
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	c := newController(service)
	router.Route("/calendars", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", c.list)
		r.Post("/", c.create)
		r.Route("/{calendarID}", func(r chi.Router) {
			r.Get("/", c.get)
			r.Put("/", c.update)
			r.Delete("/", c.delete)
			r.Get("/members", c.listMembers)
			r.Post("/members", c.share)
			r.Delete("/members/{memberID}", c.removeMember)
		})
	})
}
