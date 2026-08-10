// Package httpx assembles the chi router every Facile API starts from.
package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/FacileStudio/tronc/middleware"
)

// Config describes the standard chain.
type Config struct {
	// Logger receives the per-request records and any recovered panic.
	Logger *slog.Logger
	// CORS is applied when AllowedOrigins is non-empty.
	CORS middleware.CORSConfig
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

// trustedProxies resolves the configured set, distinguishing "unset" from
// "deliberately empty". A nil slice means the app said nothing and gets the
// default; a non-nil empty slice means it said none, and is honoured.
func (c Config) realIP() middleware.RealIPConfig {
	return middleware.RealIPConfig{
		Trusted: c.trustedProxies(),
		CDN:     c.CDNProxies,
		Header:  c.CDNHeader,
	}
}

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
	handler = middleware.RequestLogger(logger)(handler)
	if len(cfg.CORS.AllowedOrigins) > 0 {
		handler = middleware.CORS(cfg.CORS)(handler)
	}
	handler = middleware.RealIPWith(cfg.realIP())(handler)
	handler = middleware.Recoverer(logger)(handler)
	handler = chimiddleware.RequestID(handler)
	return handler
}

// NewRouter returns a chi router with the suite's middleware already applied,
// in this order:
//
//	RequestID -> Recoverer -> RealIP -> CORS -> RequestLogger
//
// RealIP is tronc's, not chi's. chi's rewrites RemoteAddr from
// X-Forwarded-For whatever the peer, which makes every per-IP rate limit
// downstream bypassable by rotating a header. Ours believes the header only
// from a trusted peer — see Config.TrustedProxies.
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
	router.Use(chimiddleware.RequestID)
	router.Use(middleware.Recoverer(logger))
	router.Use(middleware.RealIPWith(cfg.realIP()))
	if len(cfg.CORS.AllowedOrigins) > 0 {
		router.Use(middleware.CORS(cfg.CORS))
	}
	router.Use(middleware.RequestLogger(logger))
	return router
}
