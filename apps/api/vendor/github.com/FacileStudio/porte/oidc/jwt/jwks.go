package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
)

// audience is the aud claim, which OIDC permits as either one string or an
// array of them. A verifier should not care which shape arrived.
type audience []string

func (a audience) contains(want string) bool {
	for _, value := range a {
		if value == want {
			return true
		}
	}
	return false
}

func (a *audience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = audience{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

// jwk is one RSA key of a JWKS document. Only the fields RS256 verification
// needs are read; anything else in the set is ignored.
type jwk struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// rsaKey converts the JWK to a public key, or returns nil when it is not a
// usable RSA key. A malformed entry is skipped rather than fatal: one bad key
// in the provider's set must not take down verification of the others.
func (k jwk) rsaKey() *rsa.PublicKey {
	if k.Kid == "" || k.N == "" || k.E == "" {
		return nil
	}
	n, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil || len(n) == 0 {
		return nil
	}
	e, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil || len(e) == 0 || len(e) > 4 {
		return nil
	}
	exponent := 0
	for _, b := range e {
		exponent = exponent<<8 | int(b)
	}
	if exponent < 2 {
		return nil
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
}

// discover fetches the discovery document, checks that it speaks for the
// configured issuer, and returns the jwks_uri to cache.
func (v *Verifier) discover(ctx context.Context) (string, error) {
	discoveryURL := strings.TrimSuffix(v.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	var document struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := v.get(ctx, discoveryURL, &document); err != nil {
		return "", fmt.Errorf("porte/jwt: discovery failed for %s: %w", v.cfg.Issuer, err)
	}
	if document.Issuer != "" && document.Issuer != v.cfg.Issuer {
		return "", fmt.Errorf("porte/jwt: %s publishes issuer %q, not %q",
			discoveryURL, document.Issuer, v.cfg.Issuer)
	}
	if document.JWKSURI == "" {
		return "", fmt.Errorf("porte/jwt: %s advertises no jwks_uri", v.cfg.Issuer)
	}
	return document.JWKSURI, nil
}

// get fetches url and decodes the JSON body, capped at a mebibyte — a
// discovery document or a key set is never larger, and an uncapped read turns
// a misbehaving provider into a memory problem.
func (v *Verifier) get(ctx context.Context, url string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, into)
}

// payload is the decoded token body, before the verifier's checks run.
type payload struct {
	Issuer    string   `json:"iss"`
	Audience  audience `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	IssuedAt  int64    `json:"iat"`
	Subject   string   `json:"sub"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
}

// parseClaims decodes the verified payload. It lives apart from Verify so the
// wire struct stays out of the control flow.
func parseClaims(raw []byte) (payload, error) {
	var claims payload
	err := json.Unmarshal(raw, &claims)
	return claims, err
}
