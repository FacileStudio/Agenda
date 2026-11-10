package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type visitor struct {
	count     int
	windowEnd time.Time
}

// fixedWindow counts requests per key in windows that start at a key's first
// request. One mutex guards the whole map: every request reads and writes its
// visitor, so a lock per entry would buy nothing over one short critical
// section.
type fixedWindow struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	requests int
	window   time.Duration
}

// take spends one request for key at now. When the budget is spent it reports
// false and the whole seconds left in the window, plus one.
func (f *fixedWindow) take(key string, now time.Time) (bool, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.visitors[key]
	if !ok || now.After(v.windowEnd) {
		v = &visitor{windowEnd: now.Add(f.window)}
		f.visitors[key] = v
	}
	if v.count >= f.requests {
		return false, int(v.windowEnd.Sub(now).Seconds()) + 1
	}
	v.count++
	return true, 0
}

// sweep forgets every key whose window ended before now.
func (f *fixedWindow) sweep(now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, v := range f.visitors {
		if now.After(v.windowEnd) {
			delete(f.visitors, key)
		}
	}
}

// RateLimit returns middleware that allows at most requests per window per
// client IP, replying 429 with a Retry-After header once the budget is spent.
//
// It stays a fixed window rather than tronc's token bucket because CalDAV
// clients already see its exact shape: the full budget at once, then refusals
// until the window ends, with Retry-After counting down to that end.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	limiter := &fixedWindow{visitors: map[string]*visitor{}, requests: requests, window: window}

	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for now := range ticker.C {
			limiter.sweep(now)
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}
			allowed, retryAfter := limiter.take(ip, time.Now())
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				httpjson.WriteError(w, errors.RateLimited("too many requests, try again later"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
