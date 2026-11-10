package exchange

import (
	"context"
	stderrors "errors"
	"time"
)

type cachedToken struct {
	value  string
	expiry time.Time
}

type user struct {
	lock    chan struct{}
	subject cachedToken
	pending string
	tokens  map[string]cachedToken
}

func (u *user) acquire(ctx context.Context) error {
	select {
	case u.lock <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (u *user) release() { <-u.lock }

func (e *Exchanger) user(userID int64) *user {
	e.mu.Lock()
	defer e.mu.Unlock()
	u, ok := e.users[userID]
	if !ok {
		u = &user{lock: make(chan struct{}, 1), tokens: map[string]cachedToken{}}
		e.users[userID] = u
	}
	return u
}

func (e *Exchanger) usable(token cachedToken) bool {
	return token.value != "" && e.cfg.Now().Add(e.cfg.Margin).Before(token.expiry)
}

func (e *Exchanger) cached(u *user, key string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	token, ok := u.tokens[key]
	if !ok || !e.usable(token) {
		return "", false
	}
	return token.value, true
}

func (e *Exchanger) remember(u *user, key string, token cachedToken) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.usable(token) {
		u.tokens[key] = token
	}
}

func (e *Exchanger) setSubject(u *user, token cachedToken) {
	e.mu.Lock()
	defer e.mu.Unlock()
	u.subject = token
}

func (e *Exchanger) mint(ctx context.Context, u *user, userID int64, target Target) (string, error) {
	subject, fresh, err := e.subject(ctx, u, userID, false)
	if err != nil {
		return "", err
	}
	minted, err := e.registre.exchange(ctx, subject, target)
	if stderrors.Is(err, errGrantRejected) && !fresh {
		if subject, _, err = e.subject(ctx, u, userID, true); err != nil {
			return "", err
		}
		minted, err = e.registre.exchange(ctx, subject, target)
	}
	if stderrors.Is(err, errGrantRejected) {
		e.setSubject(u, cachedToken{})
		return "", ErrReconnect
	}
	if err != nil {
		return "", err
	}
	e.remember(u, target.key(), minted)
	return minted.value, nil
}
