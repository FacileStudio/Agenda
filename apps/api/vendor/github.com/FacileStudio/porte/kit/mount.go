package kit

import (
	"errors"

	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

// ErrNoSessions is what Mount returns without a session manager.
var ErrNoSessions = errors.New("porte/kit: Mount needs a session manager")

// Config is what Mount mounts. OIDC may be nil, which mounts the session
// routes alone, as an app without an OIDC kit does today.
type Config struct {
	Sessions *session.Manager
	OIDC     *oidc.Kit
}

// Mount registers exactly the routes the session manager and the OIDC kit
// register themselves, in the order every app calls them. It adds no route.
func Mount(router chi.Router, cfg Config) error {
	if cfg.Sessions == nil {
		return ErrNoSessions
	}
	cfg.Sessions.Mount(router)
	if cfg.OIDC != nil {
		cfg.OIDC.Mount(router)
	}
	return nil
}
