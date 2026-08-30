package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// DefaultRatelimitLimit is what RatelimitConfig.Limit becomes when left at
// zero. Ten requests per second is generous for an interactive app behind a
// single user's browser, which is where a chassis default belongs; routes that
// protect something expensive, like a login or a mail send, are expected to
// configure their own tighter budget rather than inherit one.
const DefaultRatelimitLimit = 10

// RateLimitExceededBody is the plain-text answer sent on a refusal. It is
// deliberately not the suite JSON envelope: a rate limiter runs before routing,
// often in front of routes whose clients are scripts hammering an endpoint, and
// a fixed byte string costs nothing to produce while being honest about who is
// refusing and why.
const RateLimitExceededBody = "rate limit exceeded"

// RatelimitConfig describes the budget Ratelimit enforces per key.
type RatelimitConfig struct {
	// Limit is the sustained refill rate in requests per second. Zero or
	// negative means DefaultRatelimitLimit.
	Limit float64
	// Burst is how many requests a key may make at once before refusals
	// start. It absorbs the small clusters every real page load produces.
	// Zero or negative means Limit rounded up.
	Burst int
	// KeyFunc derives the bucket identity from a request, typically an API
	// token or a user id, so paying customers are not throttled together
	// with anonymous traffic. Nil falls back to the host of RemoteAddr,
	// which is only the caller's real address when RealIP has rewritten it
	// first; mount this middleware below RealIP or provide a KeyFunc.
	KeyFunc func(*http.Request) string
	// StatusCode is the status answered with on refusal. Zero means 429.
	StatusCode int
}

type ratelimitBucket struct {
	tokens float64
	last   time.Time
}

// Ratelimit admits at most Limit requests per second per key, tolerating
// bursts up to Burst, and answers anything beyond with StatusCode carrying
// RateLimitExceededBody and a Retry-After estimate.
//
// State lives in a process-local map guarded by one mutex and entries are
// never evicted. The tradeoff is deliberate: eviction policy needs a second
// clock or a heap per entry, and every Facile deployment sits behind Traefik
// with a bounded set of users, so the map grows with distinct keys and stops
// growing when the user base does. A service taking raw IPs from a hostile
// public range must not use this as-is — that shape of traffic turns the map
// into an unbounded memory tax paid per attacker packet.
//
// The bucket refills continuously rather than on a ticker, so a quiet minute
// restores capacity exactly and no background goroutine exists to start,
// leak, or schedule around.
func Ratelimit(cfg RatelimitConfig) func(http.Handler) http.Handler {
	state := newRatelimitState(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			allowed, waitFor := state.allow(state.keyOf(request), time.Now())
			if allowed {
				next.ServeHTTP(w, request)
				return
			}
			w.Header().Set("Retry-After", strconv.Itoa(waitFor))
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(state.status)
			_, _ = fmt.Fprint(w, RateLimitExceededBody)
		})
	}
}

func newRatelimitState(cfg RatelimitConfig) *ratelimitState {
	limit := cfg.Limit
	if limit <= 0 {
		limit = DefaultRatelimitLimit
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = int(math.Ceil(limit))
	}
	keyOf := cfg.KeyFunc
	if keyOf == nil {
		keyOf = remoteAddrKey
	}
	return &ratelimitState{
		limit:  limit,
		burst:  burst,
		bucket: make(map[string]*ratelimitBucket),
		keyOf:  keyOf,
		status: orDefaultStatus(cfg.StatusCode),
	}
}

func orDefaultStatus(status int) int {
	if status == 0 {
		return http.StatusTooManyRequests
	}
	return status
}

// remoteAddrKey strips the port so two connections from one address share a
// bucket instead of getting a fresh budget per connection.
func remoteAddrKey(request *http.Request) string {
	if host, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
		return host
	}
	return request.RemoteAddr
}

type ratelimitState struct {
	mu     sync.Mutex
	limit  float64
	burst  int
	bucket map[string]*ratelimitBucket
	keyOf  func(*http.Request) string
	status int
}

func (state *ratelimitState) allow(key string, now time.Time) (bool, int) {
	state.mu.Lock()
	defer state.mu.Unlock()

	bucket, ok := state.bucket[key]
	if !ok {
		bucket = &ratelimitBucket{tokens: float64(state.burst), last: now}
		state.bucket[key] = bucket
	}

	bucket.tokens = math.Min(float64(state.burst), bucket.tokens+now.Sub(bucket.last).Seconds()*state.limit)
	bucket.last = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}
	wait := int(math.Ceil((1 - bucket.tokens) / state.limit))
	return false, max(wait, 1)
}
