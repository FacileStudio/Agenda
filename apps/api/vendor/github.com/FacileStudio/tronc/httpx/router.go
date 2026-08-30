// Package httpx assembles the chi router every Facile API starts from.
package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/tronc/middleware"
)

// Config describes the standard chain.
type Config struct {
	// Logger receives the per-request records and any recovered panic.
	Logger *slog.Logger
	// CORS is applied when AllowedOrigins is non-empty.
	CORS middleware.CORSConfig
	// APIPrefix is where this app's API lives, for request-log
	// classification. nil means /api.
	//
	// It is a pointer because "" is a real answer — an app whose routes sit
	// at the root, because a proxy in front strips the prefix, has an empty
	// prefix — and a plain string could not tell that apart from unset. Get
	// it wrong and every request is classified as static and logged at the
	// quiet level, which reads exactly like an app that logs nothing at all.
	// Use httpx.RootAPI for that case.
	APIPrefix *string
	// CDNProxies and CDNHeader recover the visitor when the proxy in front
	// replaced the forwarded chain rather than extending it. env.Core fills
	// them from TRUSTED_PROXIES; leaving them zero disables the case.
	CDNProxies []netip.Prefix
	CDNHeader  string
	// TrustedProxies are the peers whose X-Forwarded-For is believed.
	// Leave it nil for middleware.DefaultTrustedProxies — loopback and the
	// private ranges, which is every Facile deployment behind Traefik.
	// Pass an explicitly empty, non-nil slice to trust no proxy at all and
	// key everything on the connection address.
	TrustedProxies []netip.Prefix
}

// RootAPI is Config.APIPrefix for an app serving its API from the root.
var RootAPI = middleware.RootAPI

func (c Config) requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if c.APIPrefix == nil {
		return middleware.RequestLogger(logger)
	}
	return middleware.RequestLoggerWith(logger, middleware.RequestLoggerConfig{APIPrefix: c.APIPrefix})
}

func (c Config) realIP() middleware.RealIPConfig {
	return middleware.RealIPConfig{
		Trusted: c.trustedProxies(),
		CDN:     c.CDNProxies,
		Header:  c.CDNHeader,
	}
}

// trustedProxies resolves the configured set, distinguishing "unset" from
// "deliberately empty". A nil slice means the app said nothing and gets the
// default; a non-nil empty slice means it said none, and is honoured.
func (c Config) trustedProxies() []netip.Prefix {
	if c.TrustedProxies == nil {
		return middleware.DefaultTrustedProxies
	}
	return c.TrustedProxies
}

// Chain applies the standard middleware stack to any handler, in the same order
// NewRouter uses. It exists for apps that route with something other than chi —
// Go 1.22's http.ServeMux, for instance — so they get the same request logging,
// panic recovery and CORS without being rewritten onto a router they do not use.
func Chain(cfg Config, next http.Handler) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	handler := next
	handler = cfg.requestLogger(logger)(handler)
	if len(cfg.CORS.AllowedOrigins) > 0 {
		handler = middleware.CORS(cfg.CORS)(handler)
	}
	handler = middleware.RealIPWith(cfg.realIP())(handler)
	handler = middleware.Recoverer(logger)(handler)
	handler = middleware.RequestID(handler)
	return handler
}

// NewRouter returns a chi router with the suite's middleware already applied,
// in this order:
//
//	RequestID -> Recoverer -> RealIP -> CORS -> RequestLogger
//
// RealIP and RequestID are tronc's, not chi's. chi's RealIP rewrites
// RemoteAddr from X-Forwarded-For whatever the peer, which makes every per-IP
// rate limit downstream bypassable by rotating a header; ours believes the
// header only from a trusted peer — see Config.TrustedProxies. chi's RequestID
// takes X-Request-Id verbatim, never echoes it, and mints ids containing the
// container's hostname; ours bounds what it accepts, echoes what it settled on,
// and mints something opaque — see middleware.RequestID.
//
// Recoverer sits second, not last. The apps currently run it innermost, so a
// panic raised in CORS or in the request logger escapes to net/http and is
// answered with a bare connection error; putting it directly under RequestID
// covers the whole chain while still having an ID to log.
func NewRouter(cfg Config) *chi.Mux {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer(logger))
	router.Use(middleware.RealIPWith(cfg.realIP()))
	if len(cfg.CORS.AllowedOrigins) > 0 {
		router.Use(middleware.CORS(cfg.CORS))
	}
	router.Use(cfg.requestLogger(logger))
	return router
}
