package authcontext

import "context"

type Identity struct {
	UserID string
	Email  string
}

type contextKey struct{}

func WithIdentity(parentContext context.Context, identity Identity) context.Context {
	return context.WithValue(parentContext, contextKey{}, identity)
}

func IdentityFromContext(parentContext context.Context) (Identity, bool) {
	identity, ok := parentContext.Value(contextKey{}).(Identity)
	return identity, ok
}

func MustIdentity(ctx context.Context) Identity {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		panic("authcontext: missing identity in context")
	}
	return identity
}
