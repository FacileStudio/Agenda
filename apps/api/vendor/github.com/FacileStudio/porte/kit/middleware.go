package kit

import (
	"context"
	"net/http"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Authenticator is porte's session middleware: *session.Manager and
// *oidc.Kit both satisfy it.
type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
}

// Resolver turns the user id porte authenticated into an Identity. *Store
// satisfies it.
type Resolver interface {
	Identity(ctx context.Context, userID int64) (Identity, error)
}

// RequireAuth runs porte's middleware, which authenticates the cookie or
// bearer credential and enforces X-Facile-CSRF on cookie mutations, then
// hydrates the Identity from the app's users table. A resolver error is
// written as the response, so a session whose user no longer exists answers
// 401.
func RequireAuth(auth Authenticator, users Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticated, ok := porte.From(r.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth token"))
				return
			}
			identity, err := users.Identity(r.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
		}))
	}
}
