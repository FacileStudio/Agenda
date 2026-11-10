package auth

import (
	"context"
	"net/http"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/session"
)

// Service is what is left of Agenda's authentication after porte took the
// credential: the session middleware every module mounts, the API tokens and
// the cookie check CalDAV runs on routes outside that middleware.
type Service struct {
	sessions    *session.Manager
	requireAuth func(http.Handler) http.Handler
}

// NewService wires the auth service onto the session manager and the user
// store that hydrates porte's user id into a kit.Identity.
func NewService(sessions *session.Manager, users *kit.Store) *Service {
	return &Service{sessions: sessions, requireAuth: kit.RequireAuth(sessions, users)}
}

// RequireAuth authenticates the request through porte and puts the caller's
// kit.Identity on its context. A session whose user is gone answers 401.
func (service *Service) RequireAuth(next http.Handler) http.Handler {
	return service.requireAuth(next)
}

// Issue mints a named API token: a porte session with a label and no expiry,
// which is what the separate api_tokens table used to be.
func (service *Service) Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error) {
	return service.sessions.Issue(ctx, userID, label)
}

// AuthenticateRequest resolves the caller of a route that is not mounted
// behind RequireAuth — the inline-image endpoint, which a browser reaches with
// an <img src> and therefore with a cookie and no header.
func (service *Service) AuthenticateRequest(w http.ResponseWriter, r *http.Request) (int64, error) {
	identity, err := service.sessions.Authenticate(w, r)
	if err != nil {
		return 0, err
	}
	return identity.UserID, nil
}

// Sessions exposes the manager for the modules that list or revoke tokens.
func (service *Service) Sessions() *session.Manager { return service.sessions }
