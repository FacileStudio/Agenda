package mcptest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ClaudeCodeClient is the client_id Claude Code registers with through
// CIMD, the default client of a minted token.
const ClaudeCodeClient = "https://claude.ai/oauth/claude-code-client-metadata"

// Issuer is a fake authorization server for tests: OpenID discovery, one
// RSA key in its JWKS, and a minter of RS256 access tokens.
type Issuer struct {
	URL string
	key *rsa.PrivateKey
}

// NewIssuer starts an Issuer that stops with the test.
func NewIssuer(t testing.TB) *Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := &Issuer{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", issuer.discovery)
	mux.HandleFunc("/jwks", issuer.jwks)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer.URL = srv.URL
	return issuer
}

// discovery serves the OpenID configuration.
func (i *Issuer) discovery(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{"issuer": i.URL, "jwks_uri": i.URL + "/jwks"})
}

// jwks serves the public key.
func (i *Issuer) jwks(w http.ResponseWriter, _ *http.Request) {
	key := &i.key.PublicKey
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
}

// Token mints an access token for subject on resource with scopes, issued
// to Claude Code and valid for ten minutes.
func (i *Issuer) Token(t testing.TB, resource, subject string, scopes ...string) string {
	return i.Mint(t, map[string]any{"aud": []string{resource}, "sub": subject, "scope": strings.Join(scopes, " ")})
}

// Mint signs an access token. claims override the defaults (iss, sub, aud,
// client_id, exp, iat, jti, scope); a nil value removes a claim.
func (i *Issuer) Mint(t testing.TB, claims map[string]any) string {
	t.Helper()
	now := time.Now()
	body := map[string]any{
		"iss": i.URL, "sub": "subject", "client_id": ClaudeCodeClient,
		"exp": now.Add(10 * time.Minute).Unix(), "iat": now.Unix(), "jti": rand.Text(),
	}
	for name, value := range claims {
		body[name] = value
		if value == nil {
			delete(body, name)
		}
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "at+jwt", "kid": "k1"})
	payload, _ := json.Marshal(body)
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
