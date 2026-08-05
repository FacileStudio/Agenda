package spaces

import (
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	c := newController(service)
	router.Route("/spaces", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", c.list)
		r.Post("/", c.create)
		r.Route("/{spaceID}", func(r chi.Router) {
			r.Get("/", c.get)
			r.Put("/", c.update)
			r.Delete("/", c.delete)
			r.Post("/leave", c.leave)
			r.Get("/members", c.listMembers)
			r.Post("/members", c.addMember)
			r.Delete("/members/{userID}", c.removeMember)
			r.Put("/members/{userID}/role", c.updateRole)
		})
	})
}
