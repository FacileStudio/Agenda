package directory

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	jwksRefetchGap = time.Minute
	maxKeyDocument = 1 << 20
)

var errUnknownKey = errors.New("directory: no signing key with that kid")

// keySet is the issuer's RSA signing keys, found through its discovery
// document and refetched when a token names a kid it does not hold, at most
// once a minute so a forged kid cannot make the app hammer Registre. A cached
// kid is answered under a read lock, never behind a fetch in flight.
type keySet struct {
	mu      sync.RWMutex
	fetch   sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func (k *keySet) cached(kid string) (*rsa.PublicKey, time.Time) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.keys[kid], k.fetched
}

// lookup returns the key for kid, refreshing the set when kid is new. An
// unknown kid inside the refetch gap is ErrUnavailable: the key may exist and
// the answer is a retry, not a verdict.
func (k *keySet) lookup(ctx context.Context, c Client, kid string) (*rsa.PublicKey, error) {
	if key, _ := k.cached(kid); key != nil {
		return key, nil
	}
	k.fetch.Lock()
	defer k.fetch.Unlock()
	key, fetched := k.cached(kid)
	if key != nil {
		return key, nil
	}
	if time.Since(fetched) < jwksRefetchGap {
		return nil, fmt.Errorf("%w: unknown kid, keys fetched less than a minute ago", ErrUnavailable)
	}
	keys, err := fetchKeys(ctx, c)
	k.mu.Lock()
	k.fetched = time.Now()
	if err == nil {
		k.keys = keys
	}
	k.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if key, ok := keys[kid]; ok {
		return key, nil
	}
	return nil, errUnknownKey
}

// getJSON reads a small unauthenticated JSON document without following
// redirects: a key document must come from where the issuer said it is.
func getJSON(ctx context.Context, c Client, target string, into any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	client := *c.httpClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s answered %d", ErrUnavailable, target, response.StatusCode)
	}
	data, err := readAll(response.Body, maxKeyDocument)
	if err != nil {
		return fmt.Errorf("%w: read %s: %w", ErrUnavailable, target, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		return fmt.Errorf("%w: decode %s: %w", ErrUnavailable, target, err)
	}
	return nil
}

// jwksURI reads the discovery document and returns where the keys are. The
// document must name the configured issuer, and the keys must be served over
// https unless the issuer itself is plain http (local development, tests).
func jwksURI(ctx context.Context, c Client) (string, error) {
	var discovery struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := getJSON(ctx, c, trimIssuer(c.Issuer)+"/.well-known/openid-configuration", &discovery); err != nil {
		return "", err
	}
	if trimIssuer(discovery.Issuer) != trimIssuer(c.Issuer) {
		return "", fmt.Errorf("%w: discovery names issuer %q", ErrUnavailable, discovery.Issuer)
	}
	if !strings.HasPrefix(discovery.JWKSURI, "https://") && !strings.HasPrefix(c.Issuer, "http://") {
		return "", fmt.Errorf("%w: jwks_uri %q is not https", ErrUnavailable, discovery.JWKSURI)
	}
	return discovery.JWKSURI, nil
}

// fetchKeys reads the issuer's JWKS through its discovery document.
func fetchKeys(ctx context.Context, c Client) (map[string]*rsa.PublicKey, error) {
	location, err := jwksURI(ctx, c)
	if err != nil {
		return nil, err
	}
	var document struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := getJSON(ctx, c, location, &document); err != nil {
		return nil, err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, jwk := range document.Keys {
		if jwk.Kty != "RSA" || (jwk.Use != "" && jwk.Use != "sig") || (jwk.Alg != "" && jwk.Alg != "RS256") {
			continue
		}
		if key := rsaKey(jwk.N, jwk.E); key != nil {
			keys[jwk.Kid] = key
		}
	}
	return keys, nil
}

// rsaKey builds a public key from base64url modulus and exponent, or nil.
func rsaKey(n, e string) *rsa.PublicKey {
	modulus, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(n, "="))
	if err != nil {
		return nil
	}
	exponent, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(e, "="))
	if err != nil || len(exponent) == 0 || len(exponent) > 4 {
		return nil
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(new(big.Int).SetBytes(exponent).Int64())}
}
