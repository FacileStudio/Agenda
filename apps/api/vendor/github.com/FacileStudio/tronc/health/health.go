// Package health serves the suite's liveness and readiness endpoints.
//
// /health answers {"status":"ok"} as soon as the process is serving, and
// touches no dependency. /ready runs every registered check and answers
// {"status":"ready"} or 503 {"status":"not_ready"}.
package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/tronc/httpjson"
)

// DefaultTimeout bounds a readiness probe.
const DefaultTimeout = 2 * time.Second

// Check reports whether one dependency is usable.
type Check func(context.Context) error

// DB returns a Check that pings a database.
func DB(db *sql.DB) Check {
	return func(ctx context.Context) error {
		return db.PingContext(ctx)
	}
}

// Live handles /health.
func Live() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Ready handles /ready, running checks with DefaultTimeout.
func Ready(checks ...Check) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), DefaultTimeout)
		defer cancel()

		for _, check := range checks {
			if err := check(ctx); err != nil {
				httpjson.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
				return
			}
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// Mount registers /health and /ready on router, and again under /api so the
// same probe answers whether or not the public edge proxies only /api/*.
//
// That split is why /api/health returns an SPA shell on some apps today and a
// 404 on others: the route existed at the root while the edge only forwards
// /api. Mounting both ends it.
func Mount(router chi.Router, checks ...Check) {
	live, ready := Live(), Ready(checks...)
	for _, prefix := range []string{"", "/api"} {
		router.Get(prefix+"/health", live)
		router.Get(prefix+"/ready", ready)
	}
}
