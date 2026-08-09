package porte

import (
	"context"
	"time"
)

// Session is one issued credential. There is exactly one session model and two
// transports: an HttpOnly cookie in browsers, an Authorization: Bearer header
// for CLIs and API clients. Same table, same hash, same revocation.
//
// The token is opaque and random, stored only as a hash. That is what every
// app already does and it stays: it is revocable by construction, which a
// self-contained JWT is not.
type Session struct {
	ID int64

	// TokenHash is the stored form. The plaintext token exists only in the
	// response that issues it and in the client that holds it; porte never
	// persists it and cannot show it again.
	TokenHash string

	UserID int64

	// Label turns a session row into a named API token. Two apps have each
	// grown their own separate ApiToken type and table for exactly this,
	// which is one more mechanism than the problem needs.
	Label string

	CreatedAt time.Time

	// LastUsedAt is what makes a session list auditable and lets a user
	// recognise which row to revoke. No app records it today, which is why
	// none of them can offer "your active sessions".
	LastUsedAt time.Time

	// ExpiresAt zero means no expiry, which is what a long-lived API token
	// wants. Browser sessions always carry one.
	ExpiresAt time.Time
}

// Expired reports whether the session may no longer authenticate a request. A
// zero ExpiresAt never expires.
func (s Session) Expired(now time.Time) bool {
	return !s.ExpiresAt.IsZero() && now.After(s.ExpiresAt)
}

// IsAPIToken reports whether this row was created as a named token rather than
// an interactive login.
func (s Session) IsAPIToken() bool { return s.Label != "" }

// SessionStore persists sessions. porte/pg implements it over database/sql,
// so no ORM is forced on a consumer.
//
// Every method takes the hash, never the token: keeping the plaintext out of
// the store interface is what stops it reaching a log line or a query
// parameter.
type SessionStore interface {
	Create(ctx context.Context, session Session) (Session, error)

	// Find returns the session for a token hash, or ErrNotFound. It must
	// not filter on expiry — an expired session is a different answer from
	// a missing one, and the caller distinguishes them.
	Find(ctx context.Context, tokenHash string) (Session, error)

	// Touch records use. Implementations should coalesce writes rather
	// than issue one UPDATE per request.
	Touch(ctx context.Context, tokenHash string, at time.Time) error

	Delete(ctx context.Context, tokenHash string) error

	// DeleteByUser drops every session a user holds. This is what
	// back-channel logout calls, and it is the only mechanism by which an
	// administrative deactivation in the IdP reaches an app that issued an
	// opaque, long-lived session of its own.
	DeleteByUser(ctx context.Context, userID int64) (deleted int64, err error)

	// ListByUser backs an active-sessions screen.
	ListByUser(ctx context.Context, userID int64) ([]Session, error)

	// DeleteByID revokes one session. It takes the user id as well so a
	// handler cannot revoke another user's session by guessing an integer.
	DeleteByID(ctx context.Context, userID, sessionID int64) error

	// DeleteExpired is the sweeper. Returning the count makes it loggable.
	DeleteExpired(ctx context.Context, now time.Time) (deleted int64, err error)
}

// LoginCode is the one-time code that bridges a browser login to a CLI: the
// CLI opens the login URL, the user authenticates in the browser, and the
// callback issues a code instead of setting a cookie. The CLI exchanges it at
// RouteExchange for a bearer token.
//
// This is the flow one app already has, with its sync.Map replaced by a table.
// An in-memory map loses every pending code on redeploy and hands the code to
// the wrong replica as soon as there are two.
//
// The session is created at exchange time, not at callback time. An earlier
// revision carried a SessionID here so the exchange would be a pure lookup,
// which cannot work: the session row stores only a hash, so handing the CLI a
// usable token later would mean keeping the plaintext at rest. Consume is
// atomic, so a code still yields at most one session.
type LoginCode struct {
	// CodeHash is the stored form, hashed exactly like a session token.
	// The code is short-lived but it is a bearer credential while it lives.
	CodeHash string

	UserID int64

	ExpiresAt time.Time
}

// Expired reports whether the code is past its window.
func (c LoginCode) Expired(now time.Time) bool { return now.After(c.ExpiresAt) }

// LoginCodeStore persists pending CLI login codes.
type LoginCodeStore interface {
	Create(ctx context.Context, code LoginCode) error

	// Consume returns the code and deletes it in one operation, so a
	// replay finds nothing. It returns ErrCodeConsumed when the row is
	// gone but the code was well-formed, and ErrNotFound otherwise —
	// distinguishing a replay from a typo is worth one error value.
	Consume(ctx context.Context, codeHash string) (LoginCode, error)

	DeleteExpired(ctx context.Context, now time.Time) (deleted int64, err error)
}

// AvatarStore is where a downloaded IdP avatar lands. The fetch itself —
// HTTPS-only validation, private address rejection, size limit, content type
// check — belongs to porte and exists once here, because six divergent copies
// of an SSRF guard is the one place in this suite where drift is a
// vulnerability rather than an inconsistency.
//
// Where the bytes go is not porte's business: one app writes them under a
// storage dir, another serves them from object storage.
type AvatarStore interface {
	// Put stores the avatar and returns the URL to record. key is an
	// opaque, stable per-identity string, not a user id: the avatar is
	// fetched before UpsertFromOIDC runs, so no user id exists yet, and
	// the resulting URL rides into the upsert on Claims.AvatarURL. That
	// ordering is what keeps the whole callback to one write.
	//
	// A stable key also means a re-sync overwrites rather than
	// accumulating a new file per login, which is what the apps do today.
	//
	// contentType is already validated as an image type.
	Put(ctx context.Context, key string, data []byte, contentType string) (avatarURL string, err error)

	// Remove drops a previously stored avatar. It must be a no-op when the
	// avatar is already gone.
	Remove(ctx context.Context, avatarURL string) error
}
