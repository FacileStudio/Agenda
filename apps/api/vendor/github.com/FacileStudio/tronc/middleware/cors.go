package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// DefaultAllowedMethods, DefaultAllowedHeaders and DefaultExposedHeaders are
// what an app gets when CORSConfig leaves them empty.
//
// X-Request-Id is in both header lists on purpose. A cross-origin caller may
// send one so its own logs and the server's name the same request, and it may
// read the one that comes back — a response header is invisible to a script
// unless it is exposed, so without this the echo RequestID writes would reach
// the browser and be unreadable there, which is the same as not sending it.
var (
	DefaultAllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	DefaultAllowedHeaders = []string{"Accept", "Authorization", "Content-Type", RequestIDHeader}
	DefaultExposedHeaders = []string{RequestIDHeader}
)

// CORSConfig describes which cross-origin callers an app accepts.
type CORSConfig struct {
	// AllowedOrigins are matched exactly. "*" allows every origin and is
	// only legal when AllowCredentials is false. An empty slice denies all
	// cross-origin requests.
	AllowedOrigins []string
	// AllowCredentials sends Access-Control-Allow-Credentials: true, which
	// lets the browser attach cookies and read the response.
	AllowCredentials bool
	// AllowedHeaders defaults to DefaultAllowedHeaders. Apps with a custom
	// header, such as Capsule's X-Delete-Token, extend it here.
	AllowedHeaders []string
	// AllowedMethods defaults to DefaultAllowedMethods.
	AllowedMethods []string
	// ExposedHeaders are the response headers a script may read. It defaults
	// to DefaultExposedHeaders; pass a non-nil empty slice to expose none.
	ExposedHeaders []string
	// MaxAgeSeconds defaults to 600.
	MaxAgeSeconds int
}

// CORS answers preflights and decorates cross-origin responses.
//
// It panics when AllowedOrigins contains "*" and AllowCredentials is set.
// That pair means the response is readable by any site while carrying the
// caller's cookies, which is a credential leak rather than a configuration
// style; failing at startup is the only safe reading of it.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	wildcard := slices.Contains(cfg.AllowedOrigins, "*")
	if wildcard && cfg.AllowCredentials {
		panic(fmt.Sprintf(
			"tronc/middleware: CORS allows origin %q with AllowCredentials, which lets any site read authenticated responses; list the origins explicitly or turn credentials off",
			"*"))
	}

	methods := strings.Join(orDefault(cfg.AllowedMethods, DefaultAllowedMethods), ", ")
	headers := strings.Join(orDefault(cfg.AllowedHeaders, DefaultAllowedHeaders), ", ")
	// nil is "unset", an empty slice is "expose none" — the same distinction
	// TrustedProxies and APIPrefix make, for the same reason: a zero value that
	// stood for both would make one of the two answers unsayable.
	exposed := strings.Join(DefaultExposedHeaders, ", ")
	if cfg.ExposedHeaders != nil {
		exposed = strings.Join(cfg.ExposedHeaders, ", ")
	}
	maxAge := cfg.MaxAgeSeconds
	if maxAge == 0 {
		maxAge = 600
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, request)
				return
			}

			if !wildcard && !slices.Contains(cfg.AllowedOrigins, origin) {
				if request.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, request)
				return
			}

			header := w.Header()
			header.Add("Vary", "Origin")
			header.Add("Vary", "Access-Control-Request-Method")
			header.Add("Vary", "Access-Control-Request-Headers")
			if wildcard {
				header.Set("Access-Control-Allow-Origin", "*")
			} else {
				header.Set("Access-Control-Allow-Origin", origin)
			}
			header.Set("Access-Control-Allow-Methods", methods)
			header.Set("Access-Control-Allow-Headers", headers)
			header.Set("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
			if exposed != "" {
				header.Set("Access-Control-Expose-Headers", exposed)
			}
			if cfg.AllowCredentials {
				header.Set("Access-Control-Allow-Credentials", "true")
			}

			if request.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, request)
		})
	}
}

func orDefault(value, fallback []string) []string {
	if len(value) == 0 {
		return fallback
	}
	return value
}
