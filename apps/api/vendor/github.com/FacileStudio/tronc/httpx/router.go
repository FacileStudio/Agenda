// Package httpx assembles the chi router every Facile API starts from.
package httpx

import (
	"log/slog"
	"net/http"

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
	handler = chimiddleware.RealIP(handler)
	handler = middleware.Recoverer(logger)(handler)
	handler = chimiddleware.RequestID(handler)
	return handler
}

// NewRouter returns a chi router with the suite's middleware already applied,
// in this order:
//
//	RequestID -> Recoverer -> RealIP -> CORS -> RequestLogger
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
	router.Use(chimiddleware.RealIP)
	if len(cfg.CORS.AllowedOrigins) > 0 {
		router.Use(middleware.CORS(cfg.CORS))
	}
	router.Use(middleware.RequestLogger(logger))
	return router
}
