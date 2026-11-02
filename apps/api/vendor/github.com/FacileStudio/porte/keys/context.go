package keys

import "context"

type contextKey struct{}

// With returns a new context carrying the authenticated Key.
func With(ctx context.Context, key Key) context.Context {
	return context.WithValue(ctx, contextKey{}, key)
}

// From extracts the authenticated Key from ctx, if present.
func From(ctx context.Context) (Key, bool) {
	key, ok := ctx.Value(contextKey{}).(Key)
	return key, ok
}
