package oidc

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"

	"github.com/FacileStudio/porte"
	jwtpkg "github.com/FacileStudio/porte/oidc/jwt"
	"github.com/FacileStudio/tronc/errors"
)

// ErrProvisionRefused wraps every reason ProvisionBearer declines to create an
// account that is a verdict on the token or the person, as opposed to a store
// that failed. A resource server answers the first with 401 and the second
// with 500.
var ErrProvisionRefused = stderrors.New("porte/oidc: the account cannot be provisioned from this token")

// ProvisionBearer resolves the claims of an already verified bearer token to
// the local account, creating it when the subject has none yet. It is the
// login callback's own write path, persist, fed from an access token instead
// of an ID token: the same avatar fetch, the same UserStore.UpsertFromOIDC
// with its linking rules and refusals, the same identity row. Nothing else in
// porte creates an account.
//
// Three refusals come before the store is asked, each wrapping
// ErrProvisionRefused. The token must carry an email, as the callback
// requires. That email must be verified: an unattended creation path never
// trusts an address the provider did not vouch for, so an explicit false and
// silence (unless Config.TrustEmailWithoutVerifiedClaim) both refuse. And
// with a roles scope configured, the token must carry roles, as the callback
// requires.
//
// The issuer decides who may use the app, assigned-only apps included, before
// it mints the token, exactly as it does before a login, so nothing here
// re-checks the access policy. Concurrent first calls for one subject share a
// single provisioning; an existing account is returned untouched.
func (k *Kit) ProvisionBearer(ctx context.Context, claims jwtpkg.Claims) (porte.Identity, error) {
	if !k.Enabled() {
		return porte.Identity{}, fmt.Errorf("%w: OIDC is not configured", ErrProvisionRefused)
	}
	identity, err := ResolveBearer(ctx, k.deps.Identities, k.cfg.Issuer, claims)
	if !stderrors.Is(err, porte.ErrNotFound) {
		return identity, err
	}
	provisioned, err, _ := k.provisioning.Do(claims.Subject, func() (any, error) {
		return k.provisionOnce(ctx, claims)
	})
	if err != nil {
		return porte.Identity{}, err
	}
	identity, _ = provisioned.(porte.Identity)
	return identity, nil
}

// provisionOnce looks again under the single flight, so a caller that waited
// behind a finished provisioning finds the account instead of making another.
func (k *Kit) provisionOnce(ctx context.Context, token jwtpkg.Claims) (porte.Identity, error) {
	identity, err := ResolveBearer(ctx, k.deps.Identities, k.cfg.Issuer, token)
	if !stderrors.Is(err, porte.ErrNotFound) {
		return identity, err
	}
	claims, err := k.claimsFromAccessToken(token)
	if err != nil {
		return porte.Identity{}, err
	}
	if _, err := k.persist(ctx, claims, porte.TokenSet{}); err != nil {
		return porte.Identity{}, classifyPersist(err)
	}
	k.logger.Info("porte: provisioned an account from a bearer token", slog.String("subject", token.Subject))
	return ResolveBearer(ctx, k.deps.Identities, k.cfg.Issuer, token)
}

// claimsFromAccessToken is claimsFromIDToken for a verified access token,
// with the callback's email requirement and the bearer path's stricter one.
func (k *Kit) claimsFromAccessToken(token jwtpkg.Claims) (porte.Claims, error) {
	var verified any
	if token.Profile.EmailVerified != nil {
		verified = *token.Profile.EmailVerified
	}
	claims := porte.Claims{
		Provider:          k.cfg.Issuer,
		Subject:           token.Subject,
		Email:             normalizeEmail(token.Email),
		EmailVerified:     k.emailTrusted(verified),
		Name:              token.Name,
		PreferredUsername: token.Profile.PreferredUsername,
		Picture:           token.Profile.Picture,
		Locale:            localeClaim(token.Profile.Locale),
	}
	switch {
	case claims.Email == "":
		return porte.Claims{}, fmt.Errorf("%w: the token carries no email", ErrProvisionRefused)
	case !claims.EmailVerified:
		return porte.Claims{}, fmt.Errorf("%w: the token's email is not verified", ErrProvisionRefused)
	}
	if k.cfg.ClaimsEnabled() {
		if token.Roles == nil {
			return porte.Claims{}, fmt.Errorf("%w: scope %q is configured but the token carries no roles",
				ErrProvisionRefused, k.cfg.ClaimsScope)
		}
		claims.Roles = token.Roles
	}
	return claims, nil
}

// classifyPersist marks a store refusal, the kinds the callback shows the user
// rather than hiding, as a refusal, and leaves every other failure as one.
func classifyPersist(err error) error {
	if _, ok := callerError(err); ok {
		return fmt.Errorf("%w: %w", ErrProvisionRefused, err)
	}
	return err
}

// callerError returns the API error inside err when it is the caller's
// problem: a refusal whose message is safe to show, not an internal failure.
func callerError(err error) (*errors.Error, bool) {
	var apiErr *errors.Error
	if !stderrors.As(err, &apiErr) {
		return nil, false
	}
	switch apiErr.Code {
	case "invalid_argument", "already_exists", "unauthenticated", "permission_denied":
		return apiErr, true
	}
	return nil, false
}
