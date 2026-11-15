package plans

import (
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the authenticated plan endpoints. The guard is the
// suite's authentication middleware, which also enforces X-Facile-CSRF on
// cookie-authenticated mutations.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	mountRoutes(router, &controller{service: service}, authService.RequireAuth)
}

func mountRoutes(router chi.Router, c *controller, guard func(http.Handler) http.Handler) {
	router.Route("/plans", func(r chi.Router) {
		r.Use(guard)
		r.Get("/", c.list)
		r.Get("/{planID}", c.get)
		r.Post("/{planID}/apply", c.apply)
		r.Post("/{planID}/dismiss", c.dismiss)
	})
}
