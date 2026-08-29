package oidc

import (
	"context"
	"fmt"

	"github.com/FacileStudio/porte"
	jwtpkg "github.com/FacileStudio/porte/oidc/jwt"
)

// newBearerVerifier builds an offline JWT verifier for one audience, against
// the same provider the browser flow federates to.
//
// It takes the audience as an argument rather than reading it off cfg because
// porte builds two of these from two different settings, and they must not be
// able to see each other's. A verifier's audience is baked in at construction
// by porte/oidc/jwt, so one verifier cannot serve two audiences and the second
// one costs a second discovery and JWKS fetch at boot. That is the price of
// keeping the two token populations apart, and it is paid once per process.
func newBearerVerifier(ctx context.Context, cfg porte.Config, audience string, identities porte.IdentityStore) (bearerVerifier, error) {
	verifier, err := jwtpkg.New(ctx, jwtpkg.Config{Issuer: cfg.Issuer, Audience: audience})
	if err != nil {
		return bearerVerifier{}, fmt.Errorf("porte/oidc: bearer-token verification for audience %q failed to configure: %w", audience, err)
	}
	return bearerVerifier{tokens: verifier, identities: identities, issuer: cfg.Issuer}, nil
}

// attachBearerVerifier gives the session manager the verifier for
// Config.MachineAudience, which is what puts a service account's JWT on the
// Authorization header path. It is called only when that field is set.
//
// It deliberately does not serve the device exchange. The exchange has its own
// audience and its own verifier, and returns nothing here, so enabling one
// never enables the other.
func attachBearerVerifier(ctx context.Context, cfg porte.Config, deps Deps) error {
	verifier, err := newBearerVerifier(ctx, cfg, cfg.MachineAudience, deps.Identities)
	if err != nil {
		return err
	}
	deps.Sessions.WithJWT(verifier)
	return nil
}

// tokenVerifier is the signature half of the bearer path. It is an interface
// rather than the concrete *jwtpkg.Verifier so that the resolution below can
// be exercised against every claim shape without minting a signed token for
// each one; oidc.New always passes the real verifier.
type tokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (jwtpkg.Claims, error)
}

// bearerVerifier adapts the offline JWT verifier to the manager's JWTVerifier
// interface, and resolves what the token asserts into the local account it
// speaks for.
//
// Resolution is the same one the login callback and back-channel logout use:
// (issuer, sub) against porte_identities, where the provider column is the
// issuer URL. There is deliberately no second lookup and no email fallback —
// email is mutable at the provider, and matching on one is the account
// takeover primitive v0.3.0 removed from the callback. A subject with no row
// has never signed in here and is refused; porte does not provision an account
// from a bearer token, because the callback is where account creation happens
// and it is the path that holds the verified email.
//
// Refusal is also the whole of the deactivation story on this path. A JWT
// carries no session row, so revoking sessions — which is all back-channel
// logout can do — does not reach one. What does reach one is IdentityStore
// Find: an app that deactivates an account by making Find answer ErrNotFound
// for it locks that account out of the bearer path on the next request. An app
// that leaves the row readable keeps admitting the token until it expires,
// which is why the issuer's access-token lifetime is the real bound here.
type bearerVerifier struct {
	tokens     tokenVerifier
	identities porte.IdentityStore
	issuer     string
}

// VerifyJWT verifies the token and returns the identity it resolves to. Every
// cryptographic and claim check runs first: the store is only consulted for a
// token already proven to be signed by the issuer and minted for this app, so
// a caller without one cannot time the difference between a subject that has
// an account here and one that does not.
//
// A row that names account zero is refused rather than returned. No account
// has that id, so the row is a store bug, and the two callers of this method
// both spend the UserID directly, one to authorize a request and one to mint a
// session. Zero would be the identity of everybody at once.
func (b bearerVerifier) VerifyJWT(ctx context.Context, rawToken string) (porte.Identity, error) {
	claims, err := b.tokens.Verify(ctx, rawToken)
	if err != nil {
		return porte.Identity{}, err
	}
	if claims.Subject == "" {
		return porte.Identity{}, fmt.Errorf("porte/oidc: the token asserts no subject")
	}
	stored, err := b.identities.Find(ctx, b.issuer, claims.Subject)
	if err != nil {
		return porte.Identity{}, fmt.Errorf("porte/oidc: no local identity for subject %q: %w", claims.Subject, err)
	}
	if stored.UserID == 0 {
		return porte.Identity{}, fmt.Errorf("porte/oidc: the identity row for subject %q names no account", claims.Subject)
	}
	return porte.Identity{
		UserID:  stored.UserID,
		Subject: claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Roles:   claims.Roles,
	}, nil
}
