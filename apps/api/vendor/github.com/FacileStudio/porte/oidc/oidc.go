// Package oidc is porte's engine: the OIDC flow, the seven routes and the
// session middleware.
//
// It is separate from the contract package on purpose. An app's stores and
// domain code import porte and compile against plain types; only main.go
// imports this package and inherits go-oidc, oauth2 and chi with it.
//
// Wiring is one call:
//
//	manager, err := session.New(cfg, session.Deps{Sessions: store.Sessions()})
//	kit, err := oidc.New(ctx, cfg, oidc.Deps{Sessions: manager, Users: store, Identities: store, Codes: store})
//	manager.Mount(router)
//	kit.Mount(router)
//	router.Group(func(r chi.Router) { r.Use(manager.RequireAuth); ... })
package oidc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Deps are the stores the kit writes through, plus the session manager it
// issues into. Sessions is always required; Users, Identities and Codes become
// required once OIDC is enabled, and Avatars is optional.
//
// porte/pg implements all four stores over the default identity tables, so an
// app that has no reason to be exotic passes the same value three times.
type Deps struct {
	Users      porte.UserStore
	Identities porte.IdentityStore
	Codes      porte.LoginCodeStore
	Avatars    porte.AvatarStore
	Logger     *slog.Logger

	// Sessions is the manager that issues and verifies the credential. It
	// is required, and it is passed in rather than built here because an
	// app with a local login shares one manager between porte/oidc and
	// porte/local — two managers over the same table would each maintain
	// their own idea of the cookie and the clock.
	Sessions *session.Manager

	// ConfigExtra adds fields to GET /auth/config. Every app in the suite
	// serves a superset of porte's two keys there — one carries
	// allow_registration, another a legacy password_auth — and porte owns
	// the route, so without this an adopting app either loses its key or
	// registers the path a second time and chi panics at boot.
	//
	// porte's own sso_only and oidc_enabled are written over the returned
	// map. An app cannot use this to claim SSO is optional when it is
	// mandatory: the frontend decides whether to draw a password form on
	// those two keys, and they answer to the configuration alone.
	ConfigExtra func() map[string]any
}

// Kit serves porte's OIDC routes.
type Kit struct {
	cfg      porte.Config
	deps     Deps
	sessions *session.Manager
	provider *gooidc.Provider
	verifier *gooidc.IDTokenVerifier
	oauth    oauth2.Config
	logger   *slog.Logger
	now      func() time.Time

	// cliTokens verifies the CLI's device-grant tokens offline. It is
	// non-nil exactly when Config.CLIAudience is set, and its presence is
	// what mounts RouteDeviceExchange. An app with no CLI audience to check
	// against has no way to verify such a token, so it does not serve the
	// route at all rather than serving one that can only refuse.
	//
	// It is a second verifier, not the one the session manager holds for
	// Config.MachineAudience. The two audiences name different token
	// populations and the kit must never confuse them, so nothing here
	// reaches session.Manager.WithJWT.
	cliTokens session.JWTVerifier
}

// New performs discovery and returns a kit. It is the boot path, so every
// misconfiguration it can detect is detected here: a half-filled environment,
// an unreachable issuer, and a roles scope the provider does not offer.
//
// A disabled config (no OIDC_ISSUER) is not an error. New returns a kit that
// serves only RouteConfig and authenticates sessions, which is what an app
// running without SSO needs and what every app does today.
func New(ctx context.Context, cfg porte.Config, deps Deps) (*Kit, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if deps.Sessions == nil {
		return nil, fmt.Errorf("porte/oidc: Deps.Sessions is required; build one with session.New")
	}
	if err := agreesWith(cfg, deps.Sessions.Config()); err != nil {
		return nil, err
	}
	kit := &Kit{cfg: cfg.Resolved(), deps: deps, sessions: deps.Sessions, logger: logger, now: time.Now}

	if !cfg.Enabled() {
		return kit, nil
	}

	for name, missing := range map[string]bool{
		"UserStore":      deps.Users == nil,
		"IdentityStore":  deps.Identities == nil,
		"LoginCodeStore": deps.Codes == nil,
	} {
		if missing {
			return nil, fmt.Errorf("porte/oidc: OIDC is enabled but Deps.%s is nil", name)
		}
	}

	provider, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("porte/oidc: discovery failed for %s: %w", cfg.Issuer, err)
	}
	kit.provider = provider
	kit.verifier = provider.Verifier(&gooidc.Config{ClientID: cfg.ClientID})
	kit.oauth = oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes(),
	}

	if err := kit.guardClaimsScope(); err != nil {
		return nil, err
	}
	if cfg.MachineAudience != "" {
		if err := attachBearerVerifier(ctx, cfg, deps); err != nil {
			return nil, err
		}
	}
	if cfg.CLIAudience != "" {
		cliTokens, err := newBearerVerifier(ctx, cfg, cfg.CLIAudience, deps.Identities)
		if err != nil {
			return nil, err
		}
		kit.cliTokens = cliTokens
	}
	// The manager fills Identity.Roles through this kit, and this kit
	// needs the manager to issue sessions. Something has to be built first.
	if cfg.ClaimsEnabled() {
		deps.Sessions.WithClaims(kit)
	}
	return kit, nil
}

// agreesWith refuses a kit and a manager built from different configurations.
//
// They share one Config type but are constructed separately, and the fields
// that decide cookie behaviour are the manager's while the fields that decide
// the flow are the kit's. A manager built with a different RedirectURL or
// SuccessURL reaches a different HTTPS() verdict, which silently changes
// whether the session cookie is Secure and carries the __Host- prefix — a
// security property, decided by a typo, with nothing failing until an
// attacker notices. Cheaper to refuse at boot.
func agreesWith(kit, manager porte.Config) error {
	resolved := kit.Resolved()
	for name, pair := range map[string][2]string{
		"OIDC_REDIRECT_URL": {resolved.RedirectURL, manager.RedirectURL},
		"OIDC_SUCCESS_URL":  {resolved.SuccessURL, manager.SuccessURL},
		"SessionTTL":        {resolved.SessionTTL.String(), manager.SessionTTL.String()},
		"SessionIdleTTL":    {resolved.IdleTimeout().String(), manager.IdleTimeout().String()},
	} {
		if pair[0] != pair[1] {
			return fmt.Errorf(
				"porte/oidc: the kit and its session manager disagree about %s (%q vs %q) — build both from the same porte.Config",
				name, pair[0], pair[1])
		}
	}
	return nil
}

// Sessions returns the manager this kit issues through, so an app that wired
// the kit does not have to keep a second reference to the manager.
func (k *Kit) Sessions() *session.Manager { return k.sessions }

// RequireAuth rejects unauthenticated requests. It is the manager's middleware,
// re-exported here so wiring reads the same as it did in v0.1.
func (k *Kit) RequireAuth(next http.Handler) http.Handler { return k.sessions.RequireAuth(next) }

// Optional attaches an identity when there is one and lets the request through
// either way.
func (k *Kit) Optional(next http.Handler) http.Handler { return k.sessions.Optional(next) }

// guardClaimsScope refuses to boot when the roles scope is configured but the
// provider does not advertise it.
//
// authentik's own documentation carries the trap: group-based authorization
// needs the scope attached to the provider's property mappings *and* requested
// by the client, and when either half is missing the claim simply never
// arrives — every rule denies, silently, with no error anywhere. A silent deny
// in the auth path is the worst failure mode available, so half of it is
// caught here and the other half on the first callback, where the claim's
// absence is observable.
func (k *Kit) guardClaimsScope() error {
	if !k.cfg.ClaimsEnabled() {
		return nil
	}
	var discovery struct {
		ScopesSupported []string `json:"scopes_supported"`
	}
	if err := k.provider.Claims(&discovery); err != nil {
		return fmt.Errorf("porte/oidc: cannot read the discovery document: %w", err)
	}
	if len(discovery.ScopesSupported) == 0 {
		k.logger.Warn("porte: provider advertises no scopes_supported, cannot verify the roles scope",
			slog.String("scope", k.cfg.ClaimsScope))
		return nil
	}
	for _, scope := range discovery.ScopesSupported {
		if scope == k.cfg.ClaimsScope {
			return nil
		}
	}
	return fmt.Errorf(
		"porte/oidc: the roles scope %q is not offered by %s — attach the scope mapping to the provider's property mappings, or unset it here; leaving it half-configured makes every role check deny silently",
		k.cfg.ClaimsScope, k.cfg.Issuer)
}

// Config reports the resolved configuration, mostly so an app can read the
// TTLs it did not set.
func (k *Kit) Config() porte.Config { return k.cfg }

// Enabled reports whether the OIDC routes are live.
func (k *Kit) Enabled() bool { return k.provider != nil }

// tokenSource returns a source that refreshes the stored tokens as needed.
func (k *Kit) tokenSource(ctx context.Context, tokens porte.TokenSet) oauth2.TokenSource {
	return k.oauth.TokenSource(ctx, &oauth2.Token{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.Expiry,
	})
}

func toTokenSet(token *oauth2.Token) porte.TokenSet {
	return porte.TokenSet{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

// claimsFromIDToken normalises a verified ID token into the one type an app's
// UserStore has to understand.
func (k *Kit) claimsFromIDToken(idToken *gooidc.IDToken) (porte.Claims, error) {
	var raw struct {
		Email             string `json:"email"`
		EmailVerified     any    `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Picture           string `json:"picture"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return porte.Claims{}, fmt.Errorf("porte/oidc: cannot parse ID token claims: %w", err)
	}

	var roles []string
	if k.cfg.ClaimsEnabled() {
		var rolesOnly struct {
			Roles []string `json:"roles"`
		}
		if err := idToken.Claims(&rolesOnly); err != nil {
			return porte.Claims{}, fmt.Errorf("porte/oidc: cannot parse the roles claim: %w", err)
		}
		if rolesOnly.Roles == nil {
			// The other half of the startup guard. The scope was granted,
			// so the mapping is what is missing, and the app would
			// otherwise deny every role check without a single error.
			return porte.Claims{}, fmt.Errorf(
				"porte/oidc: scope %q was granted but no roles claim arrived — the provider is missing its scope mapping",
				k.cfg.ClaimsScope)
		}
		roles = rolesOnly.Roles
	}

	return porte.Claims{
		Provider:          k.cfg.Issuer,
		Subject:           idToken.Subject,
		Email:             raw.Email,
		EmailVerified:     k.emailTrusted(raw.EmailVerified),
		Name:              raw.Name,
		PreferredUsername: raw.PreferredUsername,
		GivenName:         raw.GivenName,
		FamilyName:        raw.FamilyName,
		Picture:           raw.Picture,
		Roles:             roles,
	}, nil
}

// emailTrusted reports whether the email in a token may be used to match an
// existing account. Matching on a mutable, unproven email is how an identity
// provider that lets a user set any address becomes an account takeover
// primitive, so an address the provider says is unverified never matches one.
//
// An absent claim is the interesting case, because it is not a "no" — the
// provider asserted nothing, and porte cannot tell "this provider omits a
// claim it verifies anyway" from "anyone can register this address here".
// Not being able to verify is a reason to refuse, so it refuses, and
// Config.TrustEmailWithoutVerifiedClaim is how an operator who knows their
// provider says otherwise. The flag cannot override an explicit false: that
// one is an assertion, and the whole point of the guard.
func (k *Kit) emailTrusted(value any) bool {
	verified, asserted := emailClaimAsserted(value)
	if asserted {
		return verified
	}
	if !k.cfg.TrustEmailWithoutVerifiedClaim {
		k.logger.Debug("porte: the provider asserted nothing about the address, so it will not match an existing account",
			slog.String("issuer", k.cfg.Issuer))
	}
	return k.cfg.TrustEmailWithoutVerifiedClaim
}

// emailClaimAsserted reads the three states an email_verified claim has: a
// verification, a refusal to verify, and silence. A malformed value is silence
// rather than a refusal — a provider sending 42 has not said no, it has failed
// to say anything, and the policy for that is the operator's to set.
func emailClaimAsserted(value any) (verified, asserted bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch typed {
		case "true", "True", "TRUE", "1":
			return true, true
		case "false", "False", "FALSE", "0":
			return false, true
		}
	}
	return false, false
}
