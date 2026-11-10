package middleware

import (
	"math"
	"net/http"
	"net/netip"
	"sync"
	"time"
)

// Limiter is the token bucket behind Ratelimit, for callers that have a key
// but no request to wrap: a login handler that limits inside its body, a chat
// bot counting messages per sender, a job spending an outbound budget.
//
// It reads Limit, Burst and MaxKeys from the RatelimitConfig it was built
// from and ignores the fields that only make sense for HTTP. A Limiter is
// safe for concurrent use and starts no goroutine.
type Limiter struct {
	mu         sync.Mutex
	limit      float64
	burst      int
	generation int
	current    map[string]*ratelimitBucket
	previous   map[string]*ratelimitBucket
}

type ratelimitBucket struct {
	tokens float64
	last   time.Time
}

// NewLimiter builds a Limiter with the same defaults Ratelimit applies: a
// Limit of zero or less becomes DefaultRatelimitLimit, a Burst of zero or less
// becomes Limit rounded up, and a MaxKeys of zero keeps every key forever.
func NewLimiter(cfg RatelimitConfig) *Limiter {
	limit := cfg.Limit
	if limit <= 0 {
		limit = DefaultRatelimitLimit
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = int(math.Ceil(limit))
	}
	generation := 0
	if cfg.MaxKeys > 0 {
		generation = max(cfg.MaxKeys, 2) / 2
	}
	return &Limiter{
		limit:      limit,
		burst:      burst,
		generation: generation,
		current:    make(map[string]*ratelimitBucket),
	}
}

// Allow spends one token from key's bucket now and reports whether there was
// one to spend.
func (l *Limiter) Allow(key string) bool {
	allowed, _ := l.take(key, time.Now())
	return allowed
}

// AllowAt spends one token from key's bucket at now. On a refusal it also
// reports how long the caller should wait before trying again, in whole
// seconds and never less than one, which is the value Ratelimit sends as
// Retry-After. Passing the clock in keeps tests free of sleeps.
func (l *Limiter) AllowAt(key string, now time.Time) (bool, time.Duration) {
	allowed, wait := l.take(key, now)
	return allowed, time.Duration(wait) * time.Second
}

func (l *Limiter) take(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.bucketFor(key, now)
	bucket.tokens = math.Min(float64(l.burst), bucket.tokens+now.Sub(bucket.last).Seconds()*l.limit)
	bucket.last = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}
	wait := int(math.Ceil((1 - bucket.tokens) / l.limit))
	return false, max(wait, 1)
}

func (l *Limiter) bucketFor(key string, now time.Time) *ratelimitBucket {
	if bucket, ok := l.current[key]; ok {
		return bucket
	}
	bucket, ok := l.previous[key]
	if ok {
		delete(l.previous, key)
	} else {
		bucket = &ratelimitBucket{tokens: float64(l.burst), last: now}
	}
	if l.generation > 0 && len(l.current) >= l.generation {
		l.previous, l.current = l.current, make(map[string]*ratelimitBucket)
	}
	l.current[key] = bucket
	return bucket
}

// RemoteAddrKey64 is a KeyFunc that buckets IPv6 callers by their /64 and
// IPv4 callers by their address. One IPv6 subscriber usually holds a whole
// /64 and can pick a fresh address from it for every request, so keying on
// the full address hands them an unlimited supply of budgets. Like the
// default key, it trusts RemoteAddr and belongs below RealIP. A RemoteAddr
// that is not an IP address is returned as its host part, unchanged.
func RemoteAddrKey64(request *http.Request) string {
	host := remoteAddrKey(request)
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.WithZone("").Prefix(64)
	if err != nil {
		return host
	}
	return prefix.String()
}
