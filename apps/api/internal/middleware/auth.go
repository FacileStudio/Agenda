package middleware

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Authenticator is the auth service: porte's session middleware, plus the
// lookup that turns the user id porte resolved into the identity the rest of
// Agenda reads.
//
// It stays one parameter so every module router keeps calling
// middleware.RequireAuth(authService) unchanged — the seam this migration was
// designed around.
type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
	IdentityForUser(ctx context.Context, userID int64) (id string, email string, err error)
}

// RequireAuth runs porte's middleware and then fills in what porte
// deliberately does not carry.
//
// porte verifies the credential — cookie or bearer, one hashed session row,
// one expiry, one idle window — and hands on a user id. It holds no email,
// because a library that decided what an identity means to an app would be
// routed around by the second app that adopted it. So the profile is looked up
// here and lands in the same context every controller already reads, keeping
// authcontext.Identity.UserID a decimal string and leaving the ParseInt call
// sites downstream untouched.
func RequireAuth(auth Authenticator) func(http.Handler) http.Handler {
	hydrate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authenticated, ok := porte.From(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}
			userID, email, err := auth.IdentityForUser(request.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			authContext := authcontext.WithIdentity(request.Context(), authcontext.Identity{
				UserID: userID,
				Email:  email,
			})
			next.ServeHTTP(w, request.WithContext(authContext))
		})
	}
	return func(next http.Handler) http.Handler {
		return auth.RequireAuth(hydrate(next))
	}
}
