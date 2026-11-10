// Package exchange gives an app a per-user RFC 8693 token exchange: it turns
// the Registre refresh token porte stored at sign-in into an access token for
// another suite app (an audience) with the scopes the app asks for.
//
// One Exchanger must serve every audience of a process. Registre rotates the
// refresh token on every refresh, so two refreshes of one grant in parallel
// leave one of them holding a dead token and the user signed out with
// invalid_grant. The Exchanger serializes them per user, across audiences, and
// persists each rotation through the IdentityStore before anyone else refreshes.
//
// Exchanged tokens are cached in memory until their expiry minus a margin. A
// downstream 401 forgets the token and retries once ([Exchanger.Do]). A refresh
// token Registre no longer accepts is [ErrReconnect]: the user signs in again.
package exchange

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/porte"
)

const (
	defaultMargin  = 30 * time.Second
	defaultTimeout = 15 * time.Second
)

// Identities is the slice of porte.IdentityStore the exchanger needs: read a
// user's identities and write a rotated refresh token back. pg.IdentityStore
// satisfies it, and seals the tokens when it was built with a TokenSealer.
type Identities interface {
	ListByUser(ctx context.Context, userID int64) ([]porte.StoredIdentity, error)
	Save(ctx context.Context, identity porte.StoredIdentity) error
}

// Target is what a token is exchanged for: the downstream app's audience (its
// Registre client id, or the name Registre's seed grants) and the scopes.
type Target struct {
	Audience string
	Scopes   []string
}

func (t Target) key() string {
	scopes := slices.Clone(t.Scopes)
	slices.Sort(scopes)
	return t.Audience + "\x00" + strings.Join(scopes, " ")
}

// Config wires an Exchanger. Issuer, ClientID, ClientSecret and Identities are
// required. Issuer must be the value porte/oidc was configured with, because
// it is also the provider porte_identities rows are filed under.
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	Identities   Identities

	// Client calls Registre. The default has a 15 s timeout.
	Client *http.Client
	// Margin is how long before its expiry a token stops being reused. 30 s by default.
	Margin time.Duration
	// Logger receives the failures that do not fail the call, such as a
	// rotated refresh token that could not be saved. slog.Default by default.
	Logger *slog.Logger
	// Now is the clock, for tests.
	Now func() time.Time
}

// Exchanger mints and caches exchanged tokens per user. It is safe for
// concurrent use; build one per process and share it.
type Exchanger struct {
	cfg      Config
	registre *registre

	mu    sync.Mutex
	users map[int64]*user
}

// New checks cfg and builds an Exchanger. It makes no network call: Registre's
// discovery document is read on the first exchange.
func New(cfg Config) (*Exchanger, error) {
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.Identities == nil {
		return nil, stderrors.New("exchange: Issuer, ClientID, ClientSecret and Identities are required")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Margin <= 0 {
		cfg.Margin = defaultMargin
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	e := &Exchanger{cfg: cfg, users: map[int64]*user{}}
	e.registre = &registre{cfg: &e.cfg}
	return e, nil
}

// Token returns an access token for target acting for userID, from the cache
// while it has more than Margin left, else minted through Registre.
//
// It fails with [ErrReconnect] when the user holds no usable Registre grant,
// with a [*RefusedError] when Registre refuses the exchange itself (a scope
// the app is not granted, for one), and with an error wrapping
// [ErrUnavailable] when Registre cannot be reached.
func (e *Exchanger) Token(ctx context.Context, userID int64, target Target) (string, error) {
	u := e.user(userID)
	key := target.key()
	if value, ok := e.cached(u, key); ok {
		return value, nil
	}
	if err := u.acquire(ctx); err != nil {
		return "", err
	}
	defer u.release()
	if value, ok := e.cached(u, key); ok {
		return value, nil
	}
	return e.mint(ctx, u, userID, target)
}

// Forget drops the cached token for one target, so the next call mints a new
// one. Call it when the downstream app answered 401 to the cached token.
func (e *Exchanger) Forget(userID int64, target Target) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if u, ok := e.users[userID]; ok {
		delete(u.tokens, target.key())
	}
}

// ForgetUser drops everything held in memory for a user: every exchanged
// token and the Registre access token. Call it on sign-out and on erasure.
func (e *Exchanger) ForgetUser(userID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.users, userID)
}
