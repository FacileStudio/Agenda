package kit

import "context"

// Identity is the signed-in caller as the app's users table describes it.
// porte authenticates the credential and carries a user id; the kit adds the
// profile the app's handlers read.
type Identity struct {
	UserID     int64
	Email      string
	Name       string
	PictureURL string
	IsAdmin    bool
}

type contextKey struct{}

// WithIdentity returns a copy of ctx carrying identity.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

// FromContext returns the identity RequireAuth stored on ctx, reporting
// whether one was there.
func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	return identity, ok
}
