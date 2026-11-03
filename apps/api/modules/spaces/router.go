package spaces

import (
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the read-only space endpoints. Creating, renaming and
// membership changes happen in Registre, which this app follows.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	c := newController(service)
	router.Route("/spaces", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", c.list)
		r.Route("/{spaceID}", func(r chi.Router) {
			r.Get("/", c.get)
			r.Get("/members", c.listMembers)
		})
	})
}

// RegisterSync mounts Registre's nudge at /registre/sync under the /api
// router, which is directory.SyncPath. It carries no authentication and no
// rate limit on purpose: the nudge is anonymous and only schedules a pull
// that Directory authenticates itself.
func RegisterSync(router chi.Router, sync *Directory) {
	router.Method(http.MethodPost, "/registre/sync", sync.Handler())
}
