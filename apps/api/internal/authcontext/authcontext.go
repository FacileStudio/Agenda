package authcontext

import "context"

// Identity carries the authenticated user's ID and email through the request context.
type Identity struct {
	UserID string
	Email  string
}

type contextKey struct{}

// WithIdentity returns a new context carrying the given identity.
func WithIdentity(parentContext context.Context, identity Identity) context.Context {
	return context.WithValue(parentContext, contextKey{}, identity)
}

// IdentityFromContext returns the identity stored in the context and whether one was present.
func IdentityFromContext(parentContext context.Context) (Identity, bool) {
	identity, ok := parentContext.Value(contextKey{}).(Identity)
	return identity, ok
}

// MustIdentity returns the identity from the context, panicking if none is present.
func MustIdentity(ctx context.Context) Identity {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		panic("authcontext: missing identity in context")
	}
	return identity
}
