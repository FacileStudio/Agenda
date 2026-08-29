// Package porte is the authentication kit for the Facile Suite: the OIDC
// plumbing that six Go apps have each written separately.
//
// This package is the frozen contract only — types, interfaces and the wire
// shapes. It deliberately depends on nothing outside the standard library, so
// that an app's store and domain code never compiles against go-oidc, oauth2
// or a database driver. The engine lives in porte/oidc and the default
// PostgreSQL stores in porte/pg; only an app's main.go imports those.
//
//	porte       the contract. standard library only
//	porte/oidc  the flow, the routes and the middleware. go-oidc, oauth2, tronc, chi
//	porte/pg    the identity tables. database/sql, no ORM
//
// porte carries authentication and transports authorization. It decides
// neither: the identity provider assigns roles, the app decides what a role
// may do, and porte owns only the transport and the freshness in between.
//
// See SPEC.md for the reasoning behind every decision expressed here.
package porte

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ErrNotFound is what a store returns when a row does not exist. Stores are
// implemented by consumers, so the contract needs a sentinel of its own rather
// than leaking sql.ErrNoRows or a GORM error across the boundary.
var ErrNotFound = errors.New("porte: not found")

// ProviderLocal is the Provider value a password identity is stored under, so
// that (Provider, Subject) remains the account matching key whether the human
// signed in with a password or through an identity provider. Subject is the
// normalised email address.
//
// A human may hold a local identity and a federated one at the same time. They
// are two rows, which is why identities were given their own table in v0.1
// rather than a column on the user.
const ProviderLocal = "local"

// The local login's failure modes, as sentinels so a handler can map them to
// status codes without matching on message text.
//
// ErrWrongPassword is deliberately what an unknown address returns too. An
// error that distinguishes them is an account enumeration oracle, and the
// timing is equalised for the same reason.
var (
	ErrWrongPassword      = errors.New("porte: invalid email or password")
	ErrEmailTaken         = errors.New("porte: an account with this email already exists")
	ErrRegistrationClosed = errors.New("porte: registration is disabled")
	ErrWeakPassword       = errors.New("porte: password is too short")
	ErrInvalidEmail       = errors.New("porte: a valid email is required")
)

// ErrNoPassword is returned when a change is asked of an account that has no
// password to change. It is not an enumeration risk the way ErrWrongPassword
// would be: reaching it takes an authenticated session for the account in
// question, so the caller already knows whose account it is.
var ErrNoPassword = errors.New("porte: this account has no password")

// ErrPasswordSet is returned when a first password is offered to an account
// that already has one. The two operations are deliberately separate calls:
// setting a first password cannot ask for the current one, so letting one
// method do both would make the confirmation optional at exactly the moment it
// matters — which is how four of porte's adopters shipped a password change
// that never asked for the old password.
var ErrPasswordSet = errors.New("porte: this account already has a password")

// ErrCodeConsumed is returned when a login code was already exchanged. It is
// distinct from ErrNotFound so a replayed code can be logged as an attack
// rather than as a typo.
var ErrCodeConsumed = errors.New("porte: login code already consumed")

// The routes porte mounts. All six apps already serve the first three at these
// exact paths with these exact response shapes, which is what makes them safe
// to freeze.
const (
	RouteConfig   = "/auth/config"
	RouteLogin    = "/auth/oidc"
	RouteCallback = "/auth/oidc/callback"
	RouteExchange = "/auth/oidc/exchange"

	// RouteDeviceExchange trades an access token the provider's device
	// grant already issued for this app's own session token. It is what
	// lets a CLI run RFC 8628 once and sign in to every tool in the suite,
	// including from a machine whose browser is somewhere else.
	//
	// The path and the wire shape are frozen by the shipped caller
	// (facile@5483f18), not chosen here. A 404 on this path is that
	// caller's signal that an app has not shipped the endpoint, and it
	// falls back to the loopback flow silently, so an app that mounts the
	// route must answer every bad request on its merits. The caller probes
	// with a POST carrying an empty body and must get a 400, never a 404.
	//
	// CLIAudience is what mounts it. MachineAudience does not.
	RouteDeviceExchange = "/auth/oidc/device/exchange"

	RouteLogout            = "/auth/logout"
	RouteSyncProfile       = "/auth/sync-profile"
	RouteBackchannelLogout = "/auth/backchannel-logout"

	// The local login's two routes, at the paths every app in the suite
	// already serves them on.
	RouteRegister   = "/auth/register"
	RouteLoginLocal = "/auth/login"
)

// The login route's query parameters, frozen because six CLIs will spell them.
// A CLI asks for the one-time-code flow with ?flow=cli, and adds ?port=N when
// it is listening on loopback for the code — the pattern gh auth login uses.
//
// StateParam carries the CLI's own nonce, which is echoed back on the loopback
// redirect as `state`. Without it the listener accepts any callback bearing a
// code, so a local process that guesses the ephemeral port can hand the CLI a
// credential of its choosing. It is optional only so that CLIs already in the
// wild keep working; a CLI that sends it must require it back.
const (
	FlowParam  = "flow"
	FlowCLI    = "cli"
	PortParam  = "port"
	StateParam = "cli_state"
)

// SessionCookieName is the browser session cookie. Courrier and Agenda already
// ship this exact name; adopting theirs keeps two apps from having to log
// everyone out twice.
const SessionCookieName = "session"

// CSRFHeaderName is the second lock on the cookie transport. SameSite=Lax
// stops cross-site form posts; a header a browser will not attach to a simple
// request stops the rest. Any non-empty value counts — the header's presence
// is the whole signal, so there is no token to distribute or rotate.
const CSRFHeaderName = "X-Facile-CSRF"

// Defaults, each taken from what the apps already do rather than invented.
const (
	// DefaultSessionTTL matches the 30 days every app hardcodes today.
	DefaultSessionTTL = 30 * 24 * time.Hour

	// DefaultLoginCodeTTL matches Plume's existing 60 second window.
	DefaultLoginCodeTTL = 60 * time.Second

	// DefaultClaimsTTL is deliberately the same number as the profile sync
	// rate limit below. One refresh cadence, not two: a role revoked in the
	// IdP stops mattering within five minutes, and the IdP sees at most one
	// refresh per user per five minutes rather than one per request.
	DefaultClaimsTTL = 5 * time.Minute

	// DefaultProfileSyncInterval is Nuage's existing profile_synced_at rate
	// limit, unchanged.
	DefaultProfileSyncInterval = 5 * time.Minute

	// DefaultSessionIdleTTL retires a browser session nobody has used for a
	// week, inside the thirty-day absolute lifetime. No app has an idle
	// window today, so this is the one default porte does not inherit: a
	// month-long credential that nothing can age out is the difference
	// between a borrowed laptop being a bad afternoon and a bad month.
	// Active users never meet it, and Config.SessionIdleTTL turns it off.
	DefaultSessionIdleTTL = 7 * 24 * time.Hour
)

// DefaultAppName is what porte's own pages say when Config.AppName is empty
// and SuccessURL carries no name to derive one from, an IP address or a bare
// path. It is deliberately generic: inventing "127" out of a loopback address
// would be worse than admitting the page does not know whose it is.
const DefaultAppName = "Sign-in"

// Config is the environment contract. The variable names are the existing
// suite convention and do not change: OIDC_ISSUER, OIDC_CLIENT_ID,
// OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL, OIDC_SUCCESS_URL, SSO_ONLY.
//
// OIDC_APP_NAME and OIDC_LOGO_URL join them and are the only two that need no
// deployment to set them: both derive from OIDC_SUCCESS_URL, so an app that
// says nothing still gets its own name and its own logo on the pages porte
// serves itself.
type Config struct {
	// Issuer is the OIDC issuer URL. Its presence is what enables OIDC —
	// the existing convention, kept.
	Issuer       string
	ClientID     string
	ClientSecret string

	// RedirectURL must match the redirect URI registered on the provider.
	RedirectURL string

	// SuccessURL is where the browser lands after a successful callback.
	SuccessURL string

	// FailureURL is where the browser lands when a login fails, with the
	// reason in an `error` query parameter. Empty means /login on
	// SuccessURL's origin, which is where every adopter's login page is.
	FailureURL string

	// AppName is what porte calls this app on the two pages it renders
	// itself: the CLI code page, and a CLI's own loopback page. Empty
	// derives it from SuccessURL's first DNS label, so
	// https://courrier.facile.studio/inbox reads as Courrier without a
	// deployment having to say so, and DefaultAppName covers a SuccessURL
	// with no name in it.
	//
	// It exists because those two pages are the only ones porte does not
	// hand back to the app. Everything else is a redirect to SuccessURL or
	// to LoginFailure, and an app owns its own chrome there. A page that
	// names nobody is one a human cannot tell from any other local process
	// asking them to sign in, which on the loopback page is the whole
	// question being asked.
	AppName string

	// LogoURL is the image shown beside that name. Empty means
	// SuccessURL's origin plus /logo.svg, which every suite app already
	// serves.
	//
	// A wrong guess costs nothing: the page removes an image that fails to
	// load and keeps the name, so an app that serves no logo shows no gap
	// and no broken icon. A CLI's loopback page carries no logo at all,
	// deliberately, because it is served from 127.0.0.1 by a binary that
	// must not need the network to draw its own page.
	LogoURL string

	// SSOOnly suppresses local password routes entirely. They are not
	// registered rather than rejected, so there is no endpoint to probe.
	SSOOnly bool

	// MachineAudience enables offline verification of bearer tokens shaped
	// as JWTs against the provider's JWKS — machine tokens minted by another
	// suite service, keyed to this audience. Empty disables it entirely,
	// which is every app today.
	//
	// It requires OIDC_ISSUER: discovery and the JWKS come from the same
	// provider the browser flow federates to. A bearer that parses as three
	// dot-separated segments is verified and never falls through to the
	// hashed-session lookup; anything else is authenticated exactly as
	// before.
	//
	// Its value is this app's own client id, because that is what a service
	// account's token is addressed to. Registre's suite-ci account declares
	// `audiences: [courrier]`, so courrier sets `courrier` here and a token
	// minted for any other app does not open it. That is the whole point of
	// the claim, and it is why CLIAudience below is a separate field rather
	// than a second use of this one.
	MachineAudience string

	// CLIAudience enables POST /auth/oidc/device/exchange: a CLI trades an
	// access token the provider's device grant issued for this app's own
	// session token. Empty means the route is not mounted at all. It
	// requires OIDC_ISSUER for the same reason MachineAudience does.
	//
	// Its value is the CLI's client id, not this app's. Registre issues the
	// suite CLI's token to `facile-cli` and that client declares no
	// audiences of its own, so the token carries `aud: ["facile-cli"]` and
	// every app that wants one login for every terminal sets exactly that.
	//
	// It is deliberately not MachineAudience, and the two hold different
	// values by construction: a service-account token is addressed to one
	// app, and the CLI's token is addressed to the CLI and presented at all
	// of them. Folding them into one setting would force an app to choose
	// between service accounts and CLI login, and an app that chose CLI
	// login would silently start rejecting every service-account token it
	// used to accept.
	//
	// The separation is also a boundary, not only a naming fix. Setting
	// this does not put the CLI's token on the bearer path: it verifies
	// tokens for this route and nothing else, so a facile-cli token cannot
	// be used directly as a credential on every RequireAuth route. It buys
	// a session here only by being exchanged for one, which leaves a
	// session row that an app can list and revoke.
	CLIAudience string

	// TrustEmailWithoutVerifiedClaim lets a callback whose token carries no
	// email_verified claim match an existing account by address.
	//
	// Off, because an absent claim is not a verification: porte cannot tell
	// a provider that omits a claim it checks anyway from one where any
	// visitor can register any address, and matching an existing account on
	// the second is an account takeover with no exploit in it. Set it only
	// for a provider whose registration you control — and note it never
	// applies to an explicit `email_verified: false`, which is a provider
	// saying no and is not an operator's to overrule.
	TrustEmailWithoutVerifiedClaim bool

	// ClaimsScope carries the roles claim. Empty disables claims handling
	// altogether, which is the state every app is in today, so leaving it
	// unset regresses nothing.
	ClaimsScope string

	// SessionTTL, ClaimsTTL and LoginCodeTTL fall back to the Default
	// constants above when zero.
	SessionTTL   time.Duration
	ClaimsTTL    time.Duration
	LoginCodeTTL time.Duration

	// SessionIdleTTL retires a browser session that has gone unused for this
	// long, independently of SessionTTL. Zero means DefaultSessionIdleTTL;
	// a negative value disables the idle window and restores the behaviour
	// the apps have today, which is an absolute expiry and nothing else.
	//
	// Labelled sessions — named API tokens — are never idled out. A token
	// wired into a nightly job is idle by design.
	SessionIdleTTL time.Duration

	// AcceptLegacyCookie makes porte also read the session cookie under its
	// unprefixed name over https, so an app adopting porte does not log out
	// everyone holding its own pre-porte `session` cookie.
	//
	// It is off by default and it is not free: while it is on, a cookie
	// named `session` scoped to the parent domain is accepted, which is the
	// cookie a compromised sibling host plants and precisely what the
	// __Host- prefix exists to refuse. Turn it on for one SessionTTL — after
	// that every surviving session was issued by porte and carries the
	// prefix — then turn it off. There is no shorter honest migration: a
	// reader cannot tell a legacy cookie from a forged one, which is the
	// whole reason the prefix is worth having.
	AcceptLegacyCookie bool
}

// HTTPS reports whether this app is served over TLS, according to its own
// configuration rather than a proxy header. It decides the Secure attribute on
// porte's cookies, so that a proxy which stops sending X-Forwarded-Proto
// downgrades nothing.
func (c Config) HTTPS() bool {
	return strings.HasPrefix(strings.ToLower(c.RedirectURL), "https://") ||
		strings.HasPrefix(strings.ToLower(c.SuccessURL), "https://")
}

// LoginFailure returns the URL a failed browser login lands on, carrying reason
// in an `error` query parameter.
//
// /auth/oidc and its callback are browser navigations, not API calls. Writing a
// JSON error body to them puts `{"code":"invalid_argument"...}` in the user's
// window, which is the app asking a human to read a wire format. Every failure
// a browser can reach goes here instead; the CLI's own endpoints keep JSON,
// because there the caller really is a program.
//
// The default keeps SuccessURL's origin and replaces its path, rather than
// appending: an app whose SuccessURL is /dashboard has its login page at
// /login, not at /dashboard/login. FailureURL covers anything else.
func (c Config) LoginFailure(reason string) string {
	target := c.FailureURL
	if target == "" {
		origin, err := url.Parse(c.SuccessURL)
		if err != nil {
			return "/login?error=" + url.QueryEscape(reason)
		}
		origin.Path = "/login"
		origin.RawQuery = ""
		origin.Fragment = ""
		target = origin.String()
	}
	dest, err := url.Parse(target)
	if err != nil {
		return "/login?error=" + url.QueryEscape(reason)
	}
	query := dest.Query()
	query.Set("error", reason)
	dest.RawQuery = query.Encode()
	return dest.String()
}

// AppLabel returns the name to print on the pages porte renders itself.
//
// AppName answers it when a deployment set one. When it did not, SuccessURL
// does: every app in the suite is served from its own subdomain, so the first
// DNS label of https://courrier.facile.studio/inbox is the app's name already
// and capitalising it is the whole derivation.
//
// The fallback is DefaultAppName rather than something cleverer. A SuccessURL
// that is an IP address or a relative path holds no name, and a page headed
// "127" would be a worse answer than a page that admits it.
func (c Config) AppLabel() string {
	if name := strings.TrimSpace(c.AppName); name != "" {
		return name
	}
	target, err := url.Parse(c.SuccessURL)
	if err != nil {
		return DefaultAppName
	}
	label, _, _ := strings.Cut(target.Hostname(), ".")
	first, size := utf8.DecodeRuneInString(label)
	if !unicode.IsLetter(first) {
		return DefaultAppName
	}
	return string(unicode.ToUpper(first)) + label[size:]
}

// Logo returns the image URL for those pages, or empty when there is none to
// derive.
//
// The default is SuccessURL's origin plus /logo.svg, the path every suite app
// already serves its mark from. Guessing is safe here in a way it usually is
// not: the page drops an image that fails to load, so an app without that file
// shows its name alone rather than a broken icon, and no deployment has to
// opt out of a default that costs it nothing.
//
// Like LoginFailure, this replaces SuccessURL's path instead of appending to
// it. An app landing on /inbox serves its logo at /logo.svg, not at
// /inbox/logo.svg.
func (c Config) Logo() string {
	if logo := strings.TrimSpace(c.LogoURL); logo != "" {
		return logo
	}
	origin, err := url.Parse(c.SuccessURL)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return ""
	}
	origin.Path = "/logo.svg"
	origin.RawQuery = ""
	origin.Fragment = ""
	return origin.String()
}

// IdleTimeout returns the idle window, or zero when it is disabled.
func (c Config) IdleTimeout() time.Duration {
	if c.SessionIdleTTL < 0 {
		return 0
	}
	if c.SessionIdleTTL == 0 {
		return DefaultSessionIdleTTL
	}
	return c.SessionIdleTTL
}

// Enabled reports whether OIDC is configured at all.
func (c Config) Enabled() bool { return c.Issuer != "" }

// validateWithoutIssuer checks the settings that are meaningless without a
// provider. Both audiences name a claim in a token this app never sees when
// OIDC is off, and each one enables a different surface, so each says which
// variable is the one to remove.
func (c Config) validateWithoutIssuer() error {
	for name, audience := range map[string]string{
		"OIDC_MACHINE_AUDIENCE": c.MachineAudience,
		"OIDC_CLI_AUDIENCE":     c.CLIAudience,
	} {
		if audience != "" {
			return fmt.Errorf("porte: %s is set but OIDC_ISSUER is empty — verifying a token audience needs a provider", name)
		}
	}
	return nil
}

// ClaimsEnabled reports whether porte should request and verify the roles
// claim. Off by default: no app reads claims today.
func (c Config) ClaimsEnabled() bool { return c.ClaimsScope != "" }

// Scopes returns the scopes to request. The first four are what all six apps
// already request; offline_access is what makes silent claim refresh possible
// without a second login.
func (c Config) Scopes() []string {
	scopes := []string{"openid", "email", "profile", "offline_access"}
	if c.ClaimsEnabled() {
		scopes = append(scopes, c.ClaimsScope)
	}
	return scopes
}

// Validate rejects a half-configured provider at startup. A missing client
// secret must not become a 500 on the first login attempt three days later.
func (c Config) Validate() error {
	if !c.Enabled() {
		return c.validateWithoutIssuer()
	}
	issuer, err := url.Parse(c.Issuer)
	switch {
	case err != nil:
		return fmt.Errorf("porte: OIDC_ISSUER is not a valid URL: %w", err)
	case issuer.Scheme != "https" && issuer.Scheme != "http", issuer.Host == "":
		// url.Parse accepts almost any string, so the parse error alone
		// catches nothing: "sso.facile.studio" parses fine as a relative
		// path and then fails discovery with an opaque error at boot.
		return fmt.Errorf("porte: OIDC_ISSUER must be an absolute http(s) URL, got %q", c.Issuer)
	}
	missing := []string{}
	for name, value := range map[string]string{
		"OIDC_CLIENT_ID":     c.ClientID,
		"OIDC_CLIENT_SECRET": c.ClientSecret,
		"OIDC_REDIRECT_URL":  c.RedirectURL,
		"OIDC_SUCCESS_URL":   c.SuccessURL,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("porte: OIDC_ISSUER is set but %s %s empty",
			strings.Join(sorted(missing), ", "), plural(len(missing), "is", "are"))
	}
	return nil
}

// Resolved returns a copy with zero durations replaced by their defaults, so
// the rest of the implementation never repeats the fallback.
func (c Config) Resolved() Config {
	if c.SessionTTL == 0 {
		c.SessionTTL = DefaultSessionTTL
	}
	if c.ClaimsTTL == 0 {
		c.ClaimsTTL = DefaultClaimsTTL
	}
	if c.LoginCodeTTL == 0 {
		c.LoginCodeTTL = DefaultLoginCodeTTL
	}
	return c
}

// ConfigResponse is the body of GET /auth/config, byte-identical to what all
// six apps serve today. A frontend reads it to decide whether to show a
// password form at all.
type ConfigResponse struct {
	SSOOnly     bool `json:"sso_only"`
	OIDCEnabled bool `json:"oidc_enabled"`
}

// CredentialsRequest is the body of POST /auth/register and POST /auth/login.
// Name is ignored by the login.
type CredentialsRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Password string `json:"password"`
}

// ExchangeRequest is the body of POST /auth/oidc/exchange: a CLI trading its
// one-time login code for a bearer token.
type ExchangeRequest struct {
	Code string `json:"code"`
}

// DeviceExchangeRequest is the body of POST /auth/oidc/device/exchange: a CLI
// trading the access token it already holds from the provider's device grant
// for this app's own session token.
//
// The field name is the shipped caller's, and the token in it is a live
// credential. Nothing that handles this struct may log it.
type DeviceExchangeRequest struct {
	AccessToken string `json:"access_token"`
}

// ExchangeResponse keeps Plume's existing wire shape, including user_id as a
// string. The Go side uses int64 throughout, but changing the JSON type would
// break the CLIs for no gain — the field is an opaque identifier to them.
type ExchangeResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// LogoutResponse is the body of POST /auth/logout.
type LogoutResponse struct {
	LoggedOut bool `json:"logged_out"`
}

// SyncProfileResponse is the body of POST /auth/sync-profile. Synced is false
// when the call was a no-op because the rate limit had not elapsed.
type SyncProfileResponse struct {
	Synced bool `json:"synced"`
}

func sorted(values []string) []string {
	out := append([]string(nil), values...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
