// Package jwt verifies bearer tokens shaped as JSON Web Tokens against a
// federated provider's JWKS, without a round trip to the provider per request.
//
// It exists so an app can accept machine tokens minted elsewhere in the
// suite — Registre issues them — while keeping porte's one-lookup-per-request
// property: keys are fetched from the provider's jwks_uri, cached by kid for a
// configurable TTL, and refetched exactly once when a token arrives signed by
// a key the cache has never seen. A token that fails any check is refused with
// an error; the caller never falls through to another verifier on a JWT that
// merely failed.
//
// The package depends on the standard library only. Signature verification is
// RS256 over PKCS#1 v1.5, which is what Authentik and every provider in the
// suite issue.
package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultCacheTTL is how long fetched keys stay trusted when Config.CacheTTL
// is zero. It is roughly the lifetime of the tokens this verifier is meant to
// consume: a key revoked at the provider stops mattering within the hour
// without any single request paying a fetch.
const DefaultCacheTTL = time.Hour

// leeway absorbs small clock differences between the verifier and the issuer.
// An exp exactly one second in the future should not fail because the two
// machines disagree about what second it is.
const leeway = 30 * time.Second

// ErrInvalid is the sentinel every refusal wraps. A caller that only cares
// whether a token is acceptable can match on it; a caller debugging a
// deployment can unwrap for the reason.
var ErrInvalid = errors.New("porte/jwt: invalid token")

// Config is the verifier's configuration.
type Config struct {
	// Issuer is the provider's issuer URL. It must match the token's iss
	// claim exactly, and discovery is fetched from it.
	Issuer string

	// Audience is the value the token's aud claim must carry. Tokens minted
	// for another relying party are refused even when the signature is
	// good — an access token for the IdP's own API must not open this app.
	Audience string

	// CacheTTL is how long a JWKS fetch stays fresh. Zero means
	// DefaultCacheTTL.
	CacheTTL time.Duration

	// Client is the HTTP client discovery and JWKS fetches go through.
	// Nil means http.DefaultClient.
	Client *http.Client

	// Now defaults to time.Now. It is here so a test can age a token past
	// its expiry without sleeping.
	Now func() time.Time
}

// Claims are what a verified token asserts, reduced to the fields an app
// authenticating a machine or a user through a bearer token needs.
type Claims struct {
	Subject string
	Email   string
	Name    string
	Roles   []string
}

// Verifier verifies RS256 JWTs against a provider's JWKS.
type Verifier struct {
	cfg    Config
	client *http.Client
	now    func() time.Time
	jwks   string

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	fetchedAt   time.Time
	refetchedAt time.Time
}

// New performs discovery and returns a verifier. It is the boot path, so an
// issuer that does not serve a discovery document or a jwks_uri fails here
// rather than on the first request.
func New(ctx context.Context, cfg Config) (*Verifier, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("porte/jwt: an issuer is required")
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("porte/jwt: an audience is required")
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	v := &Verifier{cfg: cfg, client: client, now: now}

	jwks, err := v.discover(ctx)
	if err != nil {
		return nil, err
	}
	v.jwks = jwks
	return v, nil
}

// Verify checks the signature and the registered claims of rawToken and
// returns what it asserts. Every failure wraps ErrInvalid.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("%w: not three segments", ErrInvalid)
	}
	if err := v.verifySignature(ctx, parts); err != nil {
		return Claims{}, err
	}
	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: undecodable payload: %w", ErrInvalid, err)
	}
	payload, err := parseClaims(payloadBytes)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: unparsable payload: %w", ErrInvalid, err)
	}
	if payload.Issuer != v.cfg.Issuer {
		return Claims{}, fmt.Errorf("%w: issuer %q is not %q", ErrInvalid, payload.Issuer, v.cfg.Issuer)
	}
	if !payload.Audience.contains(v.cfg.Audience) {
		return Claims{}, fmt.Errorf("%w: audience is not %q", ErrInvalid, v.cfg.Audience)
	}
	if err := withinValidity(v.now(), payload); err != nil {
		return Claims{}, err
	}
	return Claims{
		Subject: payload.Subject,
		Email:   payload.Email,
		Name:    payload.Name,
		Roles:   payload.Roles,
	}, nil
}

// withinValidity checks the three time claims against now, each with the same
// leeway.
//
// exp is mandatory: an absent one decodes as zero, which is 1970, so a token
// that carries no expiry is refused rather than accepted forever. nbf and iat
// are optional in an OAuth 2.0 access token — RFC 9068 requires only iat — so
// a zero there means the claim was absent and not that the epoch was meant.
// A token that says it is not usable yet, or that says it was minted after
// this machine's clock, is refused either way: both are a signal that the
// token was not issued for this moment.
func withinValidity(now time.Time, claims payload) error {
	if now.After(time.Unix(claims.ExpiresAt, 0).Add(leeway)) {
		return fmt.Errorf("%w: token expired", ErrInvalid)
	}
	if claims.NotBefore != 0 && now.Add(leeway).Before(time.Unix(claims.NotBefore, 0)) {
		return fmt.Errorf("%w: token is not valid yet", ErrInvalid)
	}
	if claims.IssuedAt != 0 && now.Add(leeway).Before(time.Unix(claims.IssuedAt, 0)) {
		return fmt.Errorf("%w: token was issued in the future", ErrInvalid)
	}
	return nil
}

// verifySignature checks the header, resolves the signing key by kid and
// verifies the RS256 signature over the first two segments.
func (v *Verifier) verifySignature(ctx context.Context, parts []string) error {
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return fmt.Errorf("%w: undecodable header: %w", ErrInvalid, err)
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("%w: unparsable header: %w", ErrInvalid, err)
	}
	if header.Alg != "RS256" {
		return fmt.Errorf("%w: signing algorithm %q is not accepted", ErrInvalid, header.Alg)
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return err
	}
	signature, err := decodeSegment(parts[2])
	if err != nil {
		return fmt.Errorf("%w: undecodable signature: %w", ErrInvalid, err)
	}
	digest := sha256.Sum256([]byte(strings.Join(parts[:2], ".")))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("%w: bad signature: %w", ErrInvalid, err)
	}
	return nil
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
