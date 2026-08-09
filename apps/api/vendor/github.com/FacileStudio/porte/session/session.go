// Package session is porte's credential: one opaque token, two transports, and
// the middleware that verifies it.
//
// It exists as its own package because v0.1 put all of this inside porte/oidc,
// where it had no business being. Ending a session, minting one and checking
// one are not OIDC concerns — they are what an app does whether it federates
// or not. The cost of the old layering showed up on the first adoption: an app
// with its own password login could not mint a porte session or set porte's
// cookie, so half its logins stayed on a token in localStorage while the other
// half got an HttpOnly cookie. That asymmetry is what this package removes.
//
// It depends on the contract, chi and tronc. Notably not on go-oidc or
// oauth2: an app that only wants passwords must not compile an OIDC client.
package session

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

// touchInterval coalesces last_used_at writes. Recording the exact second of
// every request would put one UPDATE on the hot path of every authenticated
// call to buy nothing: the column exists so a user can recognise a session in
// a list, and a minute of resolution does that.
const touchInterval = time.Minute

// ClaimsSource fills the parts of an identity that only a federated provider
// can answer for. porte/oidc implements it; an app with only a local login
// leaves it nil and pays exactly one query per authenticated request.
//
// It is an interface so that this package does not depend on porte/oidc, which
// would put go-oidc back into every binary and undo the split.
type ClaimsSource interface {
	Attach(ctx context.Context, identity *porte.Identity)
}

// Deps are what a Manager needs. Only Sessions is required.
type Deps struct {
	Sessions porte.SessionStore
	Logger   *slog.Logger
	Claims   ClaimsSource

	// Now defaults to time.Now. It is here so a test can age a session past
	// an idle window without sleeping for a week.
	Now func() time.Time
}

// Manager issues, verifies and ends sessions.
type Manager struct {
	cfg    porte.Config
	store  porte.SessionStore
	logger *slog.Logger
	claims ClaimsSource
	now    func() time.Time
}

// New returns a Manager. The configuration supplies the TTLs and, through
// Config.HTTPS, decides whether cookies are marked Secure.
func New(cfg porte.Config, deps Deps) (*Manager, error) {
	if deps.Sessions == nil {
		return nil, errors.Failed("porte/session: a SessionStore is required")
	}
	manager := &Manager{
		cfg:    cfg.Resolved(),
		store:  deps.Sessions,
		logger: deps.Logger,
		claims: deps.Claims,
		now:    deps.Now,
	}
	if manager.logger == nil {
		manager.logger = slog.Default()
	}
	if manager.now == nil {
		manager.now = time.Now
	}
	return manager, nil
}

// WithClaims returns m with a claims source attached. porte/oidc calls it
// after construction, because the kit and the manager each need the other and
// something has to be built first.
func (m *Manager) WithClaims(claims ClaimsSource) *Manager {
	m.claims = claims
	return m
}

// Config reports the resolved configuration, mostly so a caller can read the
// TTLs it did not set.
func (m *Manager) Config() porte.Config { return m.cfg }

// Mount registers POST /auth/logout. It lives here rather than with the OIDC
// routes because ending a session works identically with or without a
// provider, and gating it behind one cost the first adopter a second logout
// handler answering a second response shape.
func (m *Manager) Mount(router chi.Router) {
	router.Group(func(authenticated chi.Router) {
		authenticated.Use(m.RequireAuth)
		authenticated.Post(porte.RouteLogout, m.handleLogout)
	})
}

func (m *Manager) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := m.Clear(r.Context(), w, r); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, porte.LogoutResponse{LoggedOut: true})
}

// Issue mints a session, stores only its hash, and returns the plaintext
// exactly once. A label makes the row a named API token instead of a login.
func (m *Manager) Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error) {
	token, err := porte.NewToken()
	if err != nil {
		return "", porte.Session{}, errors.Internal("failed to issue a session", err)
	}
	now := m.now()
	session, err := m.store.Create(ctx, porte.Session{
		TokenHash:  porte.HashToken(token),
		UserID:     userID,
		Label:      label,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(m.cfg.SessionTTL),
	})
	if err != nil {
		return "", porte.Session{}, errors.Internal("failed to store the session", err)
	}
	return token, session, nil
}

// IssueCookie is Issue plus the Set-Cookie a browser login wants, and it
// returns the plaintext token as well so one call serves a browser and a CLI
// hitting the same endpoint.
//
// This is the method v0.1 was missing. Without it an app's own login could not
// put its session where porte's middleware, its CSRF rule and its idle window
// expect to find it, so adopting porte meant adopting it for half your logins.
func (m *Manager) IssueCookie(ctx context.Context, w http.ResponseWriter, r *http.Request, userID int64) (string, porte.Session, error) {
	token, session, err := m.Issue(ctx, userID, "")
	if err != nil {
		return "", porte.Session{}, err
	}
	m.SetSessionCookie(w, r, token)
	return token, session, nil
}

// Clear ends the session this request authenticated with and expires the
// cookie. It revokes by id, and by the id of the caller's own session, so a
// handler cannot end somebody else's by guessing an integer.
func (m *Manager) Clear(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	identity, ok := porte.From(ctx)
	if !ok {
		return errors.Unauthorized("missing auth")
	}
	if err := m.store.DeleteByID(ctx, identity.UserID, identity.SessionID); err != nil && !isNotFound(err) {
		return errors.Internal("failed to end the session", err)
	}
	m.ClearCookie(w, r, porte.SessionCookieName)
	return nil
}

// List returns every session a user holds, newest first. It backs the "your
// active sessions" screen the contract describes, and an app that names its
// API tokens — a labelled session — needs it to show them.
//
// It is here rather than left to the store because an app that has a Manager
// should not also have to hold the store: the whole point of the manager is
// that one thing owns the credential.
func (m *Manager) List(ctx context.Context, userID int64) ([]porte.Session, error) {
	return m.store.ListByUser(ctx, userID)
}

// Revoke ends one of a user's sessions by id. It takes the user id as well, so
// a handler cannot revoke somebody else's session by guessing an integer.
func (m *Manager) Revoke(ctx context.Context, userID, sessionID int64) error {
	return m.store.DeleteByID(ctx, userID, sessionID)
}

// RevokeUser drops every session a user holds. It is what back-channel logout
// calls, and it is the only mechanism by which an administrative deactivation
// in an identity provider reaches an app that issued its own opaque session.
func (m *Manager) RevokeUser(ctx context.Context, userID int64) (int64, error) {
	return m.store.DeleteByUser(ctx, userID)
}

// RequireAuth rejects unauthenticated requests. On success the handler reads
// the caller with porte.From(r.Context()).
func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.Authenticate(w, r)
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(porte.WithIdentity(r.Context(), identity)))
	})
}

// Optional attaches an identity when the request carries a valid one and lets
// the request through either way. It is for routes that serve both a signed-in
// and an anonymous caller — a public share link that shows an edit button to
// its owner.
func (m *Manager) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.Authenticate(w, r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(porte.WithIdentity(r.Context(), identity)))
	})
}

// Authenticate resolves the credential on a request. It is exported for an app
// that needs the identity outside a middleware chain — a WebSocket upgrade,
// say — and it is what both middlewares above are.
//
// Two transports, and only two: the session cookie and an Authorization
// bearer. No query parameter is read — a credential in a URL lands in access
// logs, referrers and browser history, and the two places that genuinely
// needed one, EventSource and download navigations, are exactly what the
// cookie transport serves for free.
func (m *Manager) Authenticate(w http.ResponseWriter, r *http.Request) (porte.Identity, error) {
	token, fromCookie := m.credential(r)
	if token == "" {
		return porte.Identity{}, errors.Unauthorized("missing auth")
	}

	// A cookie is attached by the browser whether or not the page meant to
	// send it, which is the whole CSRF problem. SameSite=Lax stops the
	// cross-site form post; this stops the rest, because a browser will not
	// attach a custom header cross-site without a preflight the app never
	// answers. Bearer callers are exempt: nothing attaches a header on
	// their behalf.
	if fromCookie && mutating(r) && r.Header.Get(porte.CSRFHeaderName) == "" {
		return porte.Identity{}, errors.Forbidden("missing " + porte.CSRFHeaderName + " header")
	}

	hash := porte.HashToken(token)
	stored, err := m.store.Find(r.Context(), hash)
	if err != nil {
		if isNotFound(err) {
			if fromCookie {
				m.ClearCookie(w, r, porte.SessionCookieName)
			}
			return porte.Identity{}, errors.Unauthorized("invalid session")
		}
		return porte.Identity{}, errors.Internal("failed to read the session", err)
	}

	now := m.now()
	if stored.Expired(now) || m.idledOut(stored, fromCookie, now) {
		if fromCookie {
			m.ClearCookie(w, r, porte.SessionCookieName)
		}
		// The row is dead either way, and leaving it costs a lookup on
		// every replay of a token that will never authenticate again.
		if err := m.store.Delete(r.Context(), hash); err != nil && !isNotFound(err) {
			m.logger.Warn("porte: failed to drop a dead session", slog.Any("error", err))
		}
		return porte.Identity{}, errors.Unauthorized("session expired")
	}
	if now.Sub(stored.LastUsedAt) >= touchInterval {
		if err := m.store.Touch(r.Context(), hash, now); err != nil {
			m.logger.Warn("porte: failed to record session use", slog.Any("error", err))
		}
	}

	identity := porte.Identity{UserID: stored.UserID, SessionID: stored.ID}
	if m.claims != nil {
		m.claims.Attach(r.Context(), &identity)
	}
	return identity, nil
}

// credential returns the token and whether it came from the cookie. Cookie
// first: a browser that has both is a browser, and the cookie is the transport
// with the CSRF check behind it.
func (m *Manager) credential(r *http.Request) (string, bool) {
	if value, ok := m.ReadCookie(r, porte.SessionCookieName); ok {
		return value, true
	}
	authorization := r.Header.Get("Authorization")
	if len(authorization) > 7 && strings.EqualFold(authorization[:7], "bearer ") {
		return strings.TrimSpace(authorization[7:]), false
	}
	return "", false
}

// idledOut reports whether a browser session has gone unused for longer than
// the configured idle window. It reads LastUsedAt, which the touch above keeps
// to within a minute — coarse enough to be cheap, far finer than a window
// measured in days.
//
// The window applies to the cookie transport only. Everything arriving as a
// bearer is a CLI or an API token, which is idle by design: a deploy script
// nobody runs for a fortnight, a nightly job. Expiring those would break the
// one class of credential with no human present to renew it, and it is not
// where the risk is — the window exists for the browser left signed in on a
// machine somebody else can reach.
func (m *Manager) idledOut(session porte.Session, fromCookie bool, now time.Time) bool {
	idle := m.cfg.IdleTimeout()
	if !fromCookie || idle <= 0 || session.LastUsedAt.IsZero() {
		return false
	}
	return now.Sub(session.LastUsedAt) >= idle
}

func mutating(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func isNotFound(err error) bool { return stderrors.Is(err, porte.ErrNotFound) }
