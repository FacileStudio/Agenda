package porte

import (
	"context"
	"errors"
	"time"
)

// The magic link routes, at the paths porte/magiclink mounts them on. Both are
// POST: a GET that signed anyone in would be spent by the first mail scanner
// that prefetched the link, before the human ever clicked it.
const (
	RouteMagicLink        = "/auth/magic-link"
	RouteMagicLinkConsume = "/auth/magic-link/consume"
)

// DefaultMagicLinkTTL is how long an emailed link stays usable. A link is a
// bearer credential sitting in a mailbox, so the window is short.
const DefaultMagicLinkTTL = 15 * time.Minute

// MagicLinkRateWindow is the period the per-address and per-source limits are
// counted over. It is longer than DefaultMagicLinkTTL, so a store that sweeps
// must keep a row for this long to keep counting it.
const MagicLinkRateWindow = time.Hour

// ErrMagicLinkInvalid is the one answer to a link that cannot be used. An
// unknown, expired, already spent and superseded token all return it, because
// four distinct answers would tell a holder of a stolen token which of those
// states it is in, and a guesser which tokens ever existed.
var ErrMagicLinkInvalid = errors.New("porte: this sign-in link is invalid or has expired")

// MagicLink is one issued link as a store holds it. The raw token exists only
// in the email; TokenHash is [HashToken] of it.
type MagicLink struct {
	TokenHash string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time

	// IP is the address the request came from, kept so the per-source limit
	// has something to count. Empty when the caller could not tell.
	IP string
}

// LinkKey selects links for [MagicLinkStore.CountSince]. A link matches when it
// satisfies every field that is set, and a zero key matches nothing.
type LinkKey struct {
	UserID int64
	IP     string
}

// MagicLinkStore persists issued links. porte/pg ships an implementation, and
// magiclinktest.Conformance is how an app proves its own.
type MagicLinkStore interface {
	// Create stores the link and, in the same atomic step, ends every older
	// unconsumed link of the same user.
	//
	// Supersession lives here and not in a separate call because two
	// requests racing each other would otherwise each invalidate, then each
	// insert, and leave two live links where the contract promises one.
	// Ended links stay as rows, consumed, so the rate limit still counts
	// them.
	Create(ctx context.Context, link MagicLink) error

	// Consume spends a link in one conditional update: the hash matches, the
	// link is unconsumed, and it has not expired at now. It returns the
	// user id, or [ErrNotFound] for every other case without telling them
	// apart. Two consumes of one token cannot both succeed.
	Consume(ctx context.Context, tokenHash string, now time.Time) (userID int64, err error)

	// CountSince counts the links matching key that were created at or
	// after since, spent or not.
	CountSince(ctx context.Context, key LinkKey, since time.Time) (int, error)

	// DeleteExpired removes links that expired before cutoff, and returns
	// how many. A cutoff of now minus [MagicLinkRateWindow] keeps
	// everything the limits still need.
	DeleteExpired(ctx context.Context, cutoff time.Time) (int64, error)
}

// MagicLinkRequest is the body of POST /auth/magic-link.
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// MagicLinkRequestResponse is the body of POST /auth/magic-link. It is the same
// for an address that has an account and one that does not.
type MagicLinkRequestResponse struct {
	OK bool `json:"ok"`
}

// MagicLinkConsumeRequest is the body of POST /auth/magic-link/consume.
type MagicLinkConsumeRequest struct {
	Token string `json:"token"`
}
