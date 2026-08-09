package porte

import (
	"context"
	"strings"
	"time"
)

// Identity is what an authenticated request carries. It replaces the
// `(string, any, error)` return and the `interface{ GetEmail() string }`
// assertion that every app currently performs on the result — a typed value
// costs nothing and removes a runtime failure mode from the auth path.
//
// UserID is an int64, matching porte_users.id and the int64 foreign keys every
// app already has. The apps pass a decimal string around today; that
// conversion moves to the edge and stops being repeated per handler.
//
// Email, EmailVerified and Name are not populated by the middleware. porte
// authenticates a session, which tells it a user id and nothing else; the
// profile lives in the app's own user table. An app that needs the address on
// every request reads its row and puts it in its own context — one query it
// was already making — rather than porte making that query for every handler
// that does not care. They are here because a UserStore fills them in tests
// and because v0.2's local login has them in hand at the moment it issues.
type Identity struct {
	UserID        int64
	Email         string
	EmailVerified bool
	Name          string

	// Roles is what the identity provider says about this user, for this
	// application, as of the last refresh. The strings are opaque to porte:
	// it transports them and keeps them fresh, and never assigns them
	// meaning. What an "admin" may do is the app's business.
	Roles []string

	// SessionID identifies the session row this request authenticated
	// against, so an app can offer "revoke this one device" without
	// handling the token itself.
	SessionID int64

	// RolesSyncedAt is when Roles were last refreshed against the IdP.
	RolesSyncedAt time.Time
}

// HasRole reports whether the IdP granted this role. Comparison is exact: the
// claim is produced per-provider by a scope mapping that already filtered and
// stripped it for this application, so there is no prefix to parse.
func (i Identity) HasRole(role string) bool {
	for _, granted := range i.Roles {
		if granted == role {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether any of the given roles was granted.
func (i Identity) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if i.HasRole(role) {
			return true
		}
	}
	return false
}

type contextKey struct{}

// WithIdentity attaches an identity to a context. The middleware calls it; an
// app normally only calls it in tests.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

// From returns the identity attached to ctx. The second result is false on an
// unauthenticated request.
//
//	id, ok := porte.From(ctx)
//	if !ok || !id.HasRole("admin") { ... }
//
// This is where porte stops. There is no RequireRole for IdP roles and no
// policy engine: the three role models in production (a bool column with
// first-user-admin, workspace-scoped roles, and a USER/ADMIN enum) are product
// decisions, and a library that arbitrated them would be routed around by the
// second app that adopted it.
func From(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	return identity, ok
}

// Claims is what the identity provider asserted about a user during a
// callback, normalised. It is the only porte type an app's UserStore has to
// understand, which is why it carries plain fields rather than an oauth2 or
// go-oidc value.
type Claims struct {
	// Provider identifies which configured provider issued this. It is
	// half of the account matching key and exists from v0.1 so that
	// adding a second provider later is configuration, not a schema break.
	Provider string

	// Subject is the IdP's stable identifier for the user — the `sub`
	// claim. This, with Provider, is what an account is matched on. Never
	// email: email is mutable in the IdP, so matching on it lets a rename
	// orphan an account and a delete-then-recreate inherit one.
	Subject string

	Email         string
	EmailVerified bool

	Name              string
	PreferredUsername string
	GivenName         string
	FamilyName        string
	Picture           string

	// AvatarURL is filled by porte, not by the provider: when an
	// [AvatarStore] is wired, the picture is fetched through the SSRF
	// guard and stored before the upsert runs, so the app writes the
	// final URL in the same statement as the name and the email.
	AvatarURL string

	// Roles is the flat roles claim, absent unless Config.ClaimsScope is
	// set and the provider actually emitted it.
	Roles []string

	// Tokens is the OIDC token material, needed to refresh the profile and
	// the claims later without a second login.
	Tokens TokenSet
}

// DisplayName is the name precedence every app already implements: the name
// claim, then preferred_username, then given and family names joined. Empty
// when the provider asserted none of them.
func (c Claims) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.PreferredUsername != "" {
		return c.PreferredUsername
	}
	return strings.TrimSpace(c.GivenName + " " + c.FamilyName)
}

// AvatarKey is the opaque, stable key an [AvatarStore] files this identity's
// avatar under. It is derived from the account matching key rather than from a
// user id, because the avatar is fetched before the user exists.
func (c Claims) AvatarKey() string {
	return HashToken(c.Provider + "\x00" + c.Subject)
}

// TokenSet is the OIDC token material for one identity, kept as plain strings
// so the store boundary does not depend on golang.org/x/oauth2.
//
// Two apps encrypt these columns at rest today and four store them in the
// clear. porte hands the store plaintext and the store decides: encryption at
// rest is a deployment property, and porte has no key management to offer.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// StoredIdentity is one row of porte_identities: a single way one human
// authenticates. A human may hold several — an OIDC subject and a local
// password are two rows, not two columns, which is what makes v0.2 a config
// change rather than a schema break.
type StoredIdentity struct {
	UserID   int64
	Provider string
	Subject  string

	// PasswordHash is empty for OIDC identities. Argon2, via the extracted
	// authcrypto, from v0.2.
	PasswordHash string

	Tokens TokenSet

	// Roles and RolesSyncedAt cache the claim server-side. Because sessions
	// are opaque and the claim never rides in a token, there is no token
	// size ceiling here and no group-overage problem to work around.
	Roles         []string
	RolesSyncedAt time.Time

	// SyncedAt rate-limits profile refreshes, preserving the existing
	// profile_synced_at behaviour.
	SyncedAt time.Time
}

// RolesStale reports whether the cached claim is older than ttl and should be
// refreshed against the IdP on this request.
func (s StoredIdentity) RolesStale(now time.Time, ttl time.Duration) bool {
	return s.RolesSyncedAt.IsZero() || now.Sub(s.RolesSyncedAt) >= ttl
}

// UserStore is the escape hatch, and the only thing porte itself compiles
// against for user data. porte/pg implements it with the default identity
// tables; an app with an exotic user model implements it and ignores porte/pg
// entirely.
//
// UpsertFromOIDC has side effects in real apps and the interface tolerates
// that deliberately: one app assigns a display colour on creation and makes
// the first user ever created an admin. That is product behaviour, it stays
// app-side, and it is exactly why the app implements this method instead of
// porte owning the write. porte calls it once per successful callback and
// cares only about the returned id.
//
// Matching is on (Provider, Subject). Falling back to email is permitted only
// when Claims.EmailVerified is true — an unverified email plus email matching
// is an account takeover primitive.
type UserStore interface {
	UpsertFromOIDC(ctx context.Context, claims Claims) (userID int64, err error)
}

// PasswordUserStore is the app's half of a local account, kept separate from
// UserStore because an app may enable passwords, federation, or both, and
// neither should force the other's method into its store.
//
// As with UpsertFromOIDC, the side effects are the app's on purpose: the rule
// that the first account created is an administrator is product behaviour, and
// porte has no business owning it. It is also why porte cannot make
// registration race-free by itself — counting accounts and inserting one must
// happen under a lock on a database porte does not own, so the app keeps the
// lock it already takes.
type PasswordUserStore interface {
	// CreateFromPassword creates the user row and returns its id. porte has
	// already validated the address and the password and hashed the
	// password by the time this is called.
	CreateFromPassword(ctx context.Context, email, name string) (userID int64, err error)

	// FindByEmail returns the user id for an address, or ErrNotFound.
	FindByEmail(ctx context.Context, email string) (userID int64, err error)
}

// IdentityStore persists the authentication rows and the cached claim. porte
// ships a Postgres implementation in porte/pg; an app that implements
// UserStore over its own tables implements this too.
type IdentityStore interface {
	// Find returns the identity for a provider and subject, or ErrNotFound.
	Find(ctx context.Context, provider, subject string) (StoredIdentity, error)

	// Save inserts or updates by (Provider, Subject). It writes the whole
	// row, so it must only be called with an identity the caller actually
	// holds the newest version of.
	Save(ctx context.Context, identity StoredIdentity) error

	// MarkRolesSynced moves only the roles_synced_at stamp.
	//
	// It exists because Save cannot do this safely. When a role refresh
	// fails, porte still records the attempt so a dead refresh token is not
	// retried on every request — but the identity it holds was read before
	// the refresh, and writing it back whole would undo a token rotation a
	// concurrent request had just persisted, locking the user out of every
	// later refresh. One column, one statement, no read-modify-write.
	MarkRolesSynced(ctx context.Context, provider, subject string, at time.Time) error

	// ListByUser returns every way this human can authenticate, which is
	// what an account settings screen needs and what v0.2 lists.
	ListByUser(ctx context.Context, userID int64) ([]StoredIdentity, error)
}
