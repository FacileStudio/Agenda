package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
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
	// It is ignored when OnRefuse is set.
	StatusCode int
	// MaxKeys bounds how many keys the limiter remembers, for a limiter that
	// faces public traffic where every new address is a new key. When the
	// bound is reached, the keys that have gone longest without a request are
	// forgotten first, and a forgotten key starts again from a full bucket.
	// Zero or negative keeps every key forever. Values of one count as two.
	MaxKeys int
	// OnRefuse writes the refusal instead of the plain-text default, for an
	// app whose clients already parse a JSON error envelope on 429. Retry-After
	// is set before it runs; it writes the status and the body. Nil keeps
	// StatusCode and RateLimitExceededBody.
	OnRefuse func(http.ResponseWriter, *http.Request)
}

// Ratelimit admits at most Limit requests per second per key, tolerating
// bursts up to Burst, and answers anything beyond with StatusCode carrying
// RateLimitExceededBody and a Retry-After estimate.
//
// State lives in a process-local map guarded by one mutex. Unless MaxKeys is
// set, entries are never evicted. That default is deliberate: every Facile
// deployment sits behind Traefik with a bounded set of users, so the map grows
// with distinct keys and stops growing when the user base does. A service
// taking raw IPs from a hostile public range must set MaxKeys, or that shape
// of traffic turns the map into an unbounded memory tax paid per attacker
// packet.
//
// The bucket refills continuously rather than on a ticker, so a quiet minute
// restores capacity exactly and no background goroutine exists to start,
// leak, or schedule around.
func Ratelimit(cfg RatelimitConfig) func(http.Handler) http.Handler {
	limiter := NewLimiter(cfg)
	keyOf := cfg.KeyFunc
	if keyOf == nil {
		keyOf = remoteAddrKey
	}
	status := orDefaultStatus(cfg.StatusCode)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			allowed, waitFor := limiter.take(keyOf(request), time.Now())
			if allowed {
				next.ServeHTTP(w, request)
				return
			}
			w.Header().Set("Retry-After", strconv.Itoa(waitFor))
			if cfg.OnRefuse != nil {
				cfg.OnRefuse(w, request)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(status)
			_, _ = fmt.Fprint(w, RateLimitExceededBody)
		})
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
