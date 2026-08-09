// Package local is the email and password half of porte: argon2id hashing, a
// register and a login, both landing in the same session as a federated login.
//
// It exists because five of the six apps porte replaces have a password form
// and would otherwise keep their own. Sharing the flow is worth less than
// sharing its details — the constant-time compare, the equalised timing on an
// unknown address, the length floor, the refusal to say which half of the pair
// was wrong. Those are what drift when six copies exist, and they are what an
// app gets wrong quietly.
//
// A password identity is one row of porte_identities under
// [porte.ProviderLocal], keyed on the normalised email. A human may hold that
// row and a federated one at the same time and they are the same account:
// signing in through either lands on the same user id.
//
// Holding both is arrived at through [Kit.SetPassword], from a request that is
// already authenticated — never through [Kit.Register], which refuses an
// address that already has an account. Registration cannot prove the caller
// owns the mailbox, so treating it as "the same human adding a password" hands
// every SSO account to whoever types its address first.
//
// It depends on porte/session, not on porte/oidc. An app that wants only
// passwords must not compile an OIDC client, which is the whole reason the
// session manager was extracted.
package local

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

// DefaultMinPasswordLength is the floor the suite already enforces. It is a
// length and nothing else: no character classes, which push users towards
// P@ssw0rd1 and buy nothing a length does not.
const DefaultMinPasswordLength = 12

// Config is the local login's policy.
type Config struct {
	// AllowRegistration gates POST /auth/register once an account exists.
	// The first account is always creatable regardless: locking an empty
	// instance out of itself is not a security property, and every app in
	// the suite already carries this exception.
	AllowRegistration bool

	// MinPasswordLength defaults to DefaultMinPasswordLength.
	MinPasswordLength int
}

// Deps are the stores and the session manager the kit writes through.
type Deps struct {
	Users      porte.PasswordUserStore
	Identities porte.IdentityStore
	Sessions   *session.Manager
	Logger     *slog.Logger

	// Count reports how many accounts exist, for the first-account
	// exception above. It is the app's because only the app knows what a
	// user row is, and because the app is the one holding the lock that
	// makes counting and inserting atomic.
	Count func(ctx context.Context) (int64, error)
}

// Kit serves the local routes and hashes the passwords behind them.
type Kit struct {
	cfg      Config
	deps     Deps
	sessions *session.Manager
	logger   *slog.Logger
}

// New returns a kit. It fails rather than degrading: a half-wired local login
// that answers 500 on the first sign-up is worse than one that refuses to boot.
func New(cfg Config, deps Deps) (*Kit, error) {
	for name, missing := range map[string]bool{
		"Users":      deps.Users == nil,
		"Identities": deps.Identities == nil,
		"Sessions":   deps.Sessions == nil,
		"Count":      deps.Count == nil,
	} {
		if missing {
			return nil, errors.Failed("porte/local: Deps." + name + " is required")
		}
	}
	if cfg.MinPasswordLength <= 0 {
		cfg.MinPasswordLength = DefaultMinPasswordLength
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Kit{cfg: cfg, deps: deps, sessions: deps.Sessions, logger: logger}, nil
}

// Mount registers POST /auth/register and POST /auth/login with porte's own
// {user_id, token} body.
//
// An app whose frontend expects a richer response — every existing Facile app
// answers {token, user}, and porte has no idea what a user looks like — should
// skip this and call [Kit.Register] and [Kit.Login] from its own handlers
// instead. That is the supported path, not a workaround.
func (k *Kit) Mount(router chi.Router) {
	if k.cfg.AllowRegistration {
		router.Post(porte.RouteRegister, k.handleRegister)
	}
	router.Post(porte.RouteLoginLocal, k.handleLogin)
}

// Register creates an account and signs it in, setting the session cookie and
// returning the bearer token, so one call serves a browser and a CLI.
//
// It is not race-free on its own and cannot be: counting the accounts and
// inserting one must happen under a lock on a database porte does not own. The
// app's PasswordUserStore is where that lock goes, and every Facile app
// already takes an advisory lock there.
func (k *Kit) Register(ctx context.Context, w http.ResponseWriter, r *http.Request, email, name, password string) (int64, string, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return 0, "", err
	}
	if len([]rune(password)) < k.cfg.MinPasswordLength {
		return 0, "", carrying(errors.Invalid(""), fmt.Errorf("%w: it must be at least %d characters", porte.ErrWeakPassword, k.cfg.MinPasswordLength))
	}

	if !k.cfg.AllowRegistration {
		count, err := k.deps.Count(ctx)
		if err != nil {
			return 0, "", errors.Internal("failed to count accounts", err)
		}
		if count > 0 {
			return 0, "", carrying(errors.Forbidden(""), porte.ErrRegistrationClosed)
		}
	}

	// An address that already has an account is refused, whether or not that
	// account has a password.
	//
	// Attaching one here would be an account takeover, and it was: an
	// account created by SSO, or migrated in without a hash, went to
	// whoever registered its address first — with a session issued on the
	// spot. porte cannot prove the caller owns the mailbox, because porte
	// has no mailer, so the only safe answer to "this address is already
	// somebody" is no. The same human adding a password to their own
	// account does it through SetPassword, from a request their existing
	// session already authenticates.
	//
	// The refusal pays for a hash it does not need so that it costs about
	// what creating an account costs. It is not perfect equalisation — the
	// create path also writes two rows — but argon2 is the dominant term,
	// and without this the response time alone answers "is this address
	// registered" for every address an attacker cares to try.
	var userID int64
	_, err = k.deps.Users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		EqualizeTiming(password)
		return 0, "", carrying(errors.Conflict(""), porte.ErrEmailTaken)
	case stderrors.Is(err, porte.ErrNotFound):
		userID, err = k.deps.Users.CreateFromPassword(ctx, email, strings.TrimSpace(name))
		if err != nil {
			return 0, "", err
		}
	default:
		return 0, "", errors.Internal("failed to look up the account", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return 0, "", errors.Internal("failed to hash the password", err)
	}
	if err := k.deps.Identities.Save(ctx, porte.StoredIdentity{
		UserID:       userID,
		Provider:     porte.ProviderLocal,
		Subject:      email,
		PasswordHash: hash,
	}); err != nil {
		return 0, "", errors.Internal("failed to store the identity", err)
	}

	token, _, err := k.sessions.IssueCookie(ctx, w, r, userID)
	if err != nil {
		return 0, "", err
	}
	return userID, token, nil
}

// Login verifies a password and issues a session.
//
// An unknown address and a wrong password are the same error and cost about
// the same time. Either half of that being false turns the login form into an
// account enumeration oracle, and the timing half is the one that gets dropped
// when this is reimplemented.
func (k *Kit) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (int64, string, error) {
	userID, err := k.Verify(ctx, email, password)
	if err != nil {
		return 0, "", err
	}

	token, _, err := k.sessions.IssueCookie(ctx, w, r, userID)
	if err != nil {
		return 0, "", err
	}
	return userID, token, nil
}

// Verify checks a password and returns the user id, issuing nothing.
//
// Login is this plus a session, and they are separate because not every caller
// is a browser. CalDAV and IMAP clients re-send Basic credentials on every
// request, so a protocol handler that reached for Login would mint a session
// row per request — an unbounded write to the credential table, and a
// Set-Cookie header on a response no browser will read.
//
// It carries the same enumeration guarantees as Login, which is the reason it
// lives here rather than being reimplemented against the identity store: an
// unknown address costs a real hash and returns the same error a wrong
// password does.
func (k *Kit) Verify(ctx context.Context, email, password string) (int64, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		EqualizeTiming(password)
		return 0, carrying(errors.Unauthorized(""), porte.ErrWrongPassword)
	}

	stored, err := k.deps.Identities.Find(ctx, porte.ProviderLocal, normalized)
	if err != nil {
		if stderrors.Is(err, porte.ErrNotFound) {
			EqualizeTiming(password)
			return 0, carrying(errors.Unauthorized(""), porte.ErrWrongPassword)
		}
		return 0, errors.Internal("failed to read the identity", err)
	}
	if !VerifyPassword(password, stored.PasswordHash) {
		return 0, carrying(errors.Unauthorized(""), porte.ErrWrongPassword)
	}
	return stored.UserID, nil
}

// SetPassword adds or replaces the password on an existing account. It is what
// an account settings screen calls, and what an app calls to give a
// federated-only user a password.
func (k *Kit) SetPassword(ctx context.Context, userID int64, email, password string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if len([]rune(password)) < k.cfg.MinPasswordLength {
		return carrying(errors.Invalid(""), fmt.Errorf("%w: it must be at least %d characters", porte.ErrWeakPassword, k.cfg.MinPasswordLength))
	}
	hash, err := HashPassword(password)
	if err != nil {
		return errors.Internal("failed to hash the password", err)
	}
	return k.deps.Identities.Save(ctx, porte.StoredIdentity{
		UserID:       userID,
		Provider:     porte.ProviderLocal,
		Subject:      normalized,
		PasswordHash: hash,
	})
}

func (k *Kit) handleRegister(w http.ResponseWriter, r *http.Request) {
	var request porte.CredentialsRequest
	if err := httpjson.DecodeJSON(w, r, &request); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	userID, token, err := k.Register(r.Context(), w, r, request.Email, request.Name, request.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpjson.WriteJSON(w, http.StatusCreated, porte.ExchangeResponse{
		UserID: strconv.FormatInt(userID, 10),
		Token:  token,
	})
}

func (k *Kit) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request porte.CredentialsRequest
	if err := httpjson.DecodeJSON(w, r, &request); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	userID, token, err := k.Login(r.Context(), w, r, request.Email, request.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpjson.WriteJSON(w, http.StatusOK, porte.ExchangeResponse{
		UserID: strconv.FormatInt(userID, 10),
		Token:  token,
	})
}

// carrying returns an app error that keeps tronc's code — and so the HTTP
// status — while wrapping the sentinel, so an app can match it with errors.Is
// instead of comparing message text. The contract promises the sentinels are
// matchable; without the wrap they were decoration.
func carrying(status *errors.Error, cause error) error {
	return errors.New(status.Code, cause.Error(), cause)
}

// normalizeEmail lowercases, trims and validates. The normalised form is the
// identity's Subject, so two spellings of one address cannot become two
// accounts.
func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", carrying(errors.Invalid(""), porte.ErrInvalidEmail)
	}
	return normalized, nil
}
