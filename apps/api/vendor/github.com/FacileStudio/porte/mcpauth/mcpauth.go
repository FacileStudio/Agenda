// Package mcpauth makes a Facile app an OAuth 2.1 resource server for its MCP
// endpoint, on top of the go-sdk auth package.
//
// Registre is the authorization server. It mints RS256 access tokens whose aud
// is the app's canonical MCP resource URI (RFC 8707) and whose scope claim
// carries `<app>.read` and `<app>.write`. Verifier checks them offline against
// Registre's JWKS through porte/oidc/jwt, maps (iss, sub) to the local account
// exactly as porte's Authorization header path does, and hands go-sdk a
// TokenInfo. Require mounts it with the RFC 6750 error codes MCP clients need
// for step-up, ProtectedResourceMetadata feeds the SDK's RFC 9728 handler, and
// PlanGate answers the suite's 402 to a third-party client on a free plan.
//
// With Config.Provision, a person the app has never seen gets their account on
// the first MCP call, through the app's porte OIDC kit: the web login's own
// write path, with its linking rules and refusals.
//
// Tokens Bulle obtains by RFC 8693 exchange today name the app's own client id
// as aud rather than the resource URI. Config.Audience keeps those verifying,
// for delegated tokens only, until Registre binds every token to a resource.
package mcpauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/oidc/jwt"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

var scopePrefixPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// errUnavailable is what the SDK middleware writes, as a 500, when a token
// could not be checked at all: Registre's keys or the identity store were
// unreachable. Its text is fixed because the middleware sends it to the client.
var errUnavailable = errors.New("token verification is unavailable")

// Config describes the resource server.
type Config struct {
	// Issuer is Registre's issuer URL. It must equal the token's iss and is
	// the provider column of porte_identities.
	Issuer string

	// Resource is the canonical URI of the MCP endpoint, such as
	// https://courrier.facile.studio/mcp. A token is accepted when its aud
	// names it byte for byte.
	Resource string

	// ScopePrefix is the app's scope namespace, such as "courrier".
	ScopePrefix string

	// Audience is the app's OIDC client id, the aud of the tokens Bulle
	// exchanges today. When set, a token naming it instead of Resource is
	// accepted only if it is a delegation (it carries act). Leave it empty
	// once every token is resource-bound.
	Audience string

	// Identities resolves (issuer, sub) to the local account.
	Identities porte.IdentityStore

	// Provision creates the local account on a subject's first MCP call,
	// through Accounts, instead of refusing a person who never signed in to
	// the app on the web. Off, such a token is refused with 401 as before.
	Provision bool

	// Accounts is the app's porte OIDC kit, *oidc.Kit, required when
	// Provision is set. Its ProvisionBearer is the web login's own write
	// path, so an MCP-created account is the account a web login would have
	// created: same linking rules, same refusals.
	Accounts Provisioner

	// Client fetches discovery and JWKS. Nil means http.DefaultClient.
	Client *http.Client

	// Logger receives the reason a token was refused, at debug level, never
	// the token. Nil means slog.Default().
	Logger *slog.Logger
}

// Verifier performs Registre discovery and returns the go-sdk TokenVerifier
// for cfg. Every refusal unwraps to auth.ErrInvalidToken with the fixed text
// "invalid token", so the SDK answers 401 and the client learns nothing about
// which check failed.
func Verifier(ctx context.Context, cfg Config) (auth.TokenVerifier, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	tokens, err := jwt.New(ctx, jwt.Config{
		Issuer:         cfg.Issuer,
		Audience:       cfg.Resource,
		ExtraAudiences: []string{cfg.Audience},
		Client:         cfg.Client,
	})
	if err != nil {
		return nil, fmt.Errorf("porte/mcpauth: %w", err)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	v := verifier{cfg: cfg, tokens: tokens, logger: logger}
	return v.verify, nil
}

func (c Config) validate() error {
	if c.Issuer == "" || c.Identities == nil {
		return errors.New("porte/mcpauth: an issuer and an identity store are required")
	}
	if err := c.validateProvision(); err != nil {
		return err
	}
	if !scopePrefixPattern.MatchString(c.ScopePrefix) {
		return fmt.Errorf("porte/mcpauth: scope prefix %q is not a lower-case app name", c.ScopePrefix)
	}
	resource, err := url.Parse(c.Resource)
	if err != nil || (resource.Scheme != "https" && resource.Scheme != "http") || resource.Host == "" ||
		resource.Fragment != "" || resource.RawQuery != "" {
		return fmt.Errorf("porte/mcpauth: resource %q is not an absolute URI without query or fragment", c.Resource)
	}
	return nil
}

type tokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (jwt.Claims, error)
}

type verifier struct {
	cfg    Config
	tokens tokenVerifier
	logger *slog.Logger
}

func (v verifier) verify(ctx context.Context, rawToken string, _ *http.Request) (*auth.TokenInfo, error) {
	claims, err := v.claims(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	identity, err := v.account(ctx, claims)
	if err != nil {
		return nil, err
	}
	return &auth.TokenInfo{
		UserID:     strconv.FormatInt(identity.UserID, 10),
		Scopes:     withImpliedRead(claims.Scopes, ScopesFor(v.cfg.ScopePrefix)),
		Expiration: claims.ExpiresAt,
		Extra: map[string]any{
			ExtraClientID: claims.ClientID,
			ExtraActor:    claims.Actor,
			ExtraSubject:  claims.Subject,
		},
	}, nil
}

// claims verifies the signature and registered claims, then applies the one
// audience rule jwt cannot express: the app's own client id is accepted only
// on a delegated token, so a login token Registre issued to the app itself is
// never an MCP credential.
func (v verifier) claims(ctx context.Context, rawToken string) (jwt.Claims, error) {
	claims, err := v.tokens.Verify(ctx, rawToken)
	if err != nil && !errors.Is(err, jwt.ErrInvalid) {
		return jwt.Claims{}, v.unavailable(ctx, err)
	}
	if err != nil {
		return jwt.Claims{}, v.refuse(ctx, err)
	}
	if !slices.Contains(claims.Audience, v.cfg.Resource) && claims.Actor == "" {
		return jwt.Claims{}, v.refuse(ctx, errors.New("the app audience is accepted on delegated tokens only"))
	}
	return claims, nil
}

// account resolves the claims to the local account through porte's bearer
// path, provisioning it first when the app opted in. A subject with no account
// is a refusal; a store that failed is not.
func (v verifier) account(ctx context.Context, claims jwt.Claims) (porte.Identity, error) {
	identity, err := oidc.ResolveBearer(ctx, v.cfg.Identities, v.cfg.Issuer, claims)
	if v.cfg.Provision && errors.Is(err, porte.ErrNotFound) {
		return v.provision(ctx, claims)
	}
	if err != nil && claims.Subject != "" && !errors.Is(err, porte.ErrNotFound) {
		return porte.Identity{}, v.unavailable(ctx, err)
	}
	if err != nil {
		return porte.Identity{}, v.refuse(ctx, err)
	}
	return identity, nil
}

// refuse logs why a token failed and returns the one error the client sees.
func (v verifier) refuse(ctx context.Context, reason error) error {
	v.logger.DebugContext(ctx, "porte/mcpauth: bearer token refused", slog.Any("reason", reason))
	return fmt.Errorf("%w", auth.ErrInvalidToken)
}

// unavailable logs a failure that is not a verdict on the token, such as
// Registre's keys or the identity store being unreachable.
func (v verifier) unavailable(ctx context.Context, cause error) error {
	v.logger.ErrorContext(ctx, "porte/mcpauth: bearer token could not be checked", slog.Any("error", cause))
	return errUnavailable
}
