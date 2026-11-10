package exchange

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/porte"
)

const refreshBudget = 30 * time.Second

func (e *Exchanger) currentSubject(u *user) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.usable(u.subject) {
		return u.subject.value
	}
	return ""
}

// subject returns a Registre access token for the user, and whether it was
// refreshed just now. The caller holds u.lock.
func (e *Exchanger) subject(ctx context.Context, u *user, userID int64, force bool) (string, bool, error) {
	if current := e.currentSubject(u); current != "" && !force {
		return current, false, nil
	}
	identity, err := e.identity(ctx, userID)
	if err != nil {
		return "", false, err
	}
	stored := cachedToken{value: identity.Tokens.AccessToken, expiry: identity.Tokens.Expiry}
	if !force && u.pending == "" && e.usable(stored) {
		e.setSubject(u, stored)
		return stored.value, false, nil
	}
	access, err := e.refresh(ctx, u, identity)
	return access, err == nil, err
}

// refresh trades the refresh token for a new access token and persists the
// rotated refresh token. It runs on a context the caller cannot cancel: once
// Registre has rotated the token, the new one must reach the store or the
// user is signed out.
func (e *Exchanger) refresh(ctx context.Context, u *user, identity porte.StoredIdentity) (string, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refreshBudget)
	defer cancel()
	used := identity.Tokens.RefreshToken
	if u.pending != "" {
		used = u.pending
	}
	if used == "" {
		return "", ErrReconnect
	}
	set, err := e.registre.refresh(ctx, used)
	if stderrors.Is(err, errGrantRejected) {
		identity, set, err = e.refreshRotated(ctx, identity, used)
	}
	if stderrors.Is(err, errGrantRejected) {
		u.pending = ""
		return "", ErrReconnect
	}
	if err != nil {
		return "", err
	}
	e.persist(ctx, u, identity, set)
	return set.AccessToken, nil
}

// refreshRotated retries once when the stored refresh token is no longer the
// one just refused: another replica, or porte's own role refresh, rotated it
// in the meantime.
func (e *Exchanger) refreshRotated(ctx context.Context, identity porte.StoredIdentity, used string) (porte.StoredIdentity, porte.TokenSet, error) {
	latest, err := e.identity(ctx, identity.UserID)
	if err != nil || latest.Tokens.RefreshToken == "" || latest.Tokens.RefreshToken == used {
		return identity, porte.TokenSet{}, errGrantRejected
	}
	set, err := e.registre.refresh(ctx, latest.Tokens.RefreshToken)
	return latest, set, err
}

func (e *Exchanger) persist(ctx context.Context, u *user, identity porte.StoredIdentity, set porte.TokenSet) {
	e.setSubject(u, cachedToken{value: set.AccessToken, expiry: set.Expiry})
	identity.Tokens = set
	if err := e.cfg.Identities.Save(ctx, identity); err != nil {
		u.pending = set.RefreshToken
		e.cfg.Logger.Warn("exchange: the rotated refresh token was not saved, keeping it in memory",
			slog.Int64("user_id", identity.UserID), slog.Any("error", err))
		return
	}
	u.pending = ""
}

// identity is the user's Registre identity: the federated row filed under
// the configured issuer. A user can hold several, such as a subject left over
// from a recreated Registre account, so the one holding a refresh token wins,
// and among those the most recently synced.
func (e *Exchanger) identity(ctx context.Context, userID int64) (porte.StoredIdentity, error) {
	stored, err := e.cfg.Identities.ListByUser(ctx, userID)
	if err != nil {
		return porte.StoredIdentity{}, fmt.Errorf("exchange: reading the identities: %w", err)
	}
	issuer := strings.TrimRight(e.cfg.Issuer, "/")
	var best porte.StoredIdentity
	found := false
	for _, identity := range stored {
		if strings.TrimRight(identity.Provider, "/") != issuer || identity.PasswordHash != "" {
			continue
		}
		if !found || better(identity, best) {
			best, found = identity, true
		}
	}
	if !found {
		return porte.StoredIdentity{}, ErrReconnect
	}
	return best, nil
}

func better(candidate, current porte.StoredIdentity) bool {
	hasToken, hadToken := candidate.Tokens.RefreshToken != "", current.Tokens.RefreshToken != ""
	if hasToken != hadToken {
		return hasToken
	}
	return candidate.SyncedAt.After(current.SyncedAt)
}
