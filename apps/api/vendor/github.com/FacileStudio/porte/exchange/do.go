package exchange

import (
	"context"
	"io"
	"net/http"

	"github.com/FacileStudio/porte"
)

// Call sends one downstream request with token as its bearer credential.
type Call func(ctx context.Context, token string) (*http.Response, error)

// Do runs call with the user's token for target. When the downstream app
// answers 401, the token is forgotten and call runs once more with a freshly
// minted one; a second 401 is returned as it is. call must therefore build a
// new request each time it runs.
func (e *Exchanger) Do(ctx context.Context, userID int64, target Target, call Call) (*http.Response, error) {
	response, err := e.attempt(ctx, userID, target, call)
	if err != nil || response.StatusCode != http.StatusUnauthorized {
		return response, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxBody))
	_ = response.Body.Close()
	e.Forget(userID, target)
	return e.attempt(ctx, userID, target, call)
}

func (e *Exchanger) attempt(ctx context.Context, userID int64, target Target, call Call) (*http.Response, error) {
	token, err := e.Token(ctx, userID, target)
	if err != nil {
		return nil, err
	}
	return call(ctx, token)
}

// Source is an Exchanger bound to one target, acting for the user porte
// authenticated on the request context. Its methods fit a client that takes
// a func(ctx) (string, error) token source, such as tronc/bulle.
type Source struct {
	exchanger *Exchanger
	target    Target
}

// Source binds the exchanger to target.
func (e *Exchanger) Source(target Target) Source {
	return Source{exchanger: e, target: target}
}

// Token is [Exchanger.Token] for the user on ctx, or [ErrNoUser].
func (s Source) Token(ctx context.Context) (string, error) {
	identity, ok := porte.From(ctx)
	if !ok || identity.UserID == 0 {
		return "", ErrNoUser
	}
	return s.exchanger.Token(ctx, identity.UserID, s.target)
}

// Forget is [Exchanger.Forget] for the user on ctx.
func (s Source) Forget(ctx context.Context) {
	if identity, ok := porte.From(ctx); ok {
		s.exchanger.Forget(identity.UserID, s.target)
	}
}

// Do is [Exchanger.Do] for the user on ctx, or [ErrNoUser].
func (s Source) Do(ctx context.Context, call Call) (*http.Response, error) {
	identity, ok := porte.From(ctx)
	if !ok || identity.UserID == 0 {
		return nil, ErrNoUser
	}
	return s.exchanger.Do(ctx, identity.UserID, s.target, call)
}
