package auth

import (
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte owns through porte/kit: /auth/config,
// /auth/logout, /auth/sync-profile and the whole OIDC flow.
//
// Agenda adds none of its own. There is no register or login route, and
// porte mounts none while an issuer is configured, so sign-in goes through
// Registre only.
func RegisterRoutes(router chi.Router, sessions *session.Manager, oidcKit *oidc.Kit) error {
	return kit.Mount(router, kit.Config{Sessions: sessions, OIDC: oidcKit})
}
