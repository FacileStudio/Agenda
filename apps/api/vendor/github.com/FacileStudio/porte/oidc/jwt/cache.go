package jwt

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"
)

// minRefetchInterval is the floor between two JWKS fetches provoked by a kid
// the cache does not hold.
//
// The kid is attacker-controlled: it is read out of the token header before
// any signature has been checked, so without a floor every unauthenticated
// request carrying a made-up kid costs one outbound fetch, serialized behind
// the same mutex every real verification waits on. Eleven apps pointed at one
// issuer turn that into an amplifier aimed at the provider, which is the
// failure registre's own SPEC calls out about per-request round trips.
//
// It is short on purpose. A rotation the provider published before signing
// with the new key is already in the cache; this path is the safety net for
// one that was not, and half a minute of refusals is the price of the floor.
const minRefetchInterval = 30 * time.Second

// key returns the signing key for kid, refreshing the cache when it is stale
// and once more when the kid is simply unknown — a key rotated in since the
// last fetch must not wait out the TTL.
//
// The unknown-kid refetch is rate limited by minRefetchInterval, because the
// kid arrives unauthenticated and would otherwise buy a stranger one fetch per
// request. Inside that window an unknown kid is refused from cache.
//
// The lock is held across the fetch, so verifications serialize while the
// provider is slow and each queued caller retries the fetch on its own; a
// provider outage is a latency wall, never a fallback to stale keys. That is
// deliberate — fail closed — and worth revisiting only with singleflight or
// serve-stale, not by dropping the guard.
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	stale := v.keys == nil || v.now().Sub(v.fetchedAt) >= v.ttl()
	if stale {
		if err := v.fetch(ctx); err != nil {
			return nil, err
		}
	}
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	if !stale && v.takeRefetchSlot() {
		if err := v.fetch(ctx); err != nil {
			return nil, err
		}
		if key, ok := v.keys[kid]; ok {
			return key, nil
		}
	}
	return nil, fmt.Errorf("%w: no key for kid %q", ErrInvalid, kid)
}

// takeRefetchSlot reports whether an unknown kid may spend a JWKS fetch now,
// and consumes the slot when it may. The caller holds the lock.
func (v *Verifier) takeRefetchSlot() bool {
	now := v.now()
	if !v.refetchedAt.IsZero() && now.Sub(v.refetchedAt) < minRefetchInterval {
		return false
	}
	v.refetchedAt = now
	return true
}

func (v *Verifier) ttl() time.Duration {
	if v.cfg.CacheTTL == 0 {
		return DefaultCacheTTL
	}
	return v.cfg.CacheTTL
}

func (v *Verifier) fetch(ctx context.Context) error {
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := v.get(ctx, v.jwks, &set); err != nil {
		return fmt.Errorf("porte/jwt: JWKS fetch failed: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if key := jwk.rsaKey(); key != nil {
			keys[jwk.Kid] = key
		}
	}
	v.keys = keys
	v.fetchedAt = v.now()
	return nil
}
