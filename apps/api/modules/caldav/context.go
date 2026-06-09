package caldav

import (
	"context"

	"api/schemas"
)

type contextKey int

const userContextKey contextKey = iota

func withUser(ctx context.Context, user *schemas.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func userFromContext(ctx context.Context) *schemas.User {
	u, _ := ctx.Value(userContextKey).(*schemas.User)
	return u
}
