package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// FixedWindowConfig describes the budget FixedWindow enforces per key.
type FixedWindowConfig struct {
	// Requests is how many requests a key may make per window. Zero or
	// negative means DefaultRatelimitLimit.
	Requests int
	// Window is the length of one window, which starts at the key's first
	// request. Zero or negative means one minute.
	Window time.Duration
	// KeyFunc derives the bucket identity. Nil, or an empty result, falls
	// back to the host of RemoteAddr, which is the visitor only below RealIP.
	KeyFunc func(*http.Request) string
	// MaxKeys bounds how many keys are remembered. Expired windows are
	// dropped first; when the map is still full, it starts over and every
	// key gets a fresh budget. Zero or negative keeps every live key.
	MaxKeys int
	// OnRefuse writes the refusal. Retry-After is set before it runs. Nil
	// answers 429 with the suite error envelope, code rate_limited.
	OnRefuse func(http.ResponseWriter, *http.Request)
}

type fixedWindowEntry struct {
	count int
	end   time.Time
}

type fixedWindow struct {
	mu        sync.Mutex
	entries   map[string]*fixedWindowEntry
	requests  int
	window    time.Duration
	maxKeys   int
	nextSweep time.Time
}

func (f *fixedWindow) take(key string, now time.Time) (bool, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sweep(now)
	entry, ok := f.entries[key]
	if !ok || now.After(entry.end) {
		entry = &fixedWindowEntry{end: now.Add(f.window)}
		f.entries[key] = entry
	}
	if entry.count >= f.requests {
		return false, int(entry.end.Sub(now).Seconds()) + 1
	}
	entry.count++
	return true, 0
}

func (f *fixedWindow) sweep(now time.Time) {
	full := f.maxKeys > 0 && len(f.entries) >= f.maxKeys
	if !full && now.Before(f.nextSweep) {
		return
	}
	f.nextSweep = now.Add(f.window)
	for key, entry := range f.entries {
		if now.After(entry.end) {
			delete(f.entries, key)
		}
	}
	if full && len(f.entries) >= f.maxKeys {
		f.entries = make(map[string]*fixedWindowEntry)
	}
}

// FixedWindow admits at most Requests per Window per key and answers the rest
// with 429 and a Retry-After counting down to the end of the window.
//
// Use it instead of Ratelimit when clients depend on the window shape: the
// full budget at once, then refusals until the window ends, as CalDAV and
// other sync clients see it. A fixed window admits up to twice the budget
// across a window boundary; Ratelimit's token bucket never does. It starts no
// goroutine: expired windows are dropped during requests.
func FixedWindow(cfg FixedWindowConfig) func(http.Handler) http.Handler {
	state := newFixedWindow(cfg)
	keyOf := cfg.KeyFunc
	if keyOf == nil {
		keyOf = remoteAddrKey
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			key := keyOf(request)
			if key == "" {
				key = remoteAddrKey(request)
			}
			allowed, retryAfter := state.take(key, time.Now())
			if allowed {
				next.ServeHTTP(w, request)
				return
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			if cfg.OnRefuse != nil {
				cfg.OnRefuse(w, request)
				return
			}
			httpjson.WriteError(w, errors.RateLimited("too many requests, try again later"))
		})
	}
}

func newFixedWindow(cfg FixedWindowConfig) *fixedWindow {
	state := &fixedWindow{
		entries:  make(map[string]*fixedWindowEntry),
		requests: cfg.Requests,
		window:   cfg.Window,
		maxKeys:  cfg.MaxKeys,
	}
	if state.requests <= 0 {
		state.requests = DefaultRatelimitLimit
	}
	if state.window <= 0 {
		state.window = time.Minute
	}
	return state
}
