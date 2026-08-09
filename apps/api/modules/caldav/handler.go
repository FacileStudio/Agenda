package caldav

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/schemas"

	gocaldav "github.com/emersion/go-webdav/caldav"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RegisterRoutes(router chi.Router, db *gorm.DB, authService *auth.Service) {
	// chi only knows standard HTTP methods by default.
	// WebDAV/CalDAV uses PROPFIND, PROPPATCH, MKCALENDAR, REPORT — register them
	// so chi includes them in its mALL bitmask and HandleFunc matches them.
	for _, m := range []string{"PROPFIND", "PROPPATCH", "MKCALENDAR", "REPORT", "COPY", "MOVE", "LOCK", "UNLOCK"} {
		chi.RegisterMethod(m)
	}

	backend := NewBackend(db)
	handler := &gocaldav.Handler{
		Backend: backend,
		Prefix:  davPrefix,
	}

	davAuth := davAuthMiddleware(db, authService)
	// 100 requests/minute per IP — prevents Basic Auth brute force while
	// allowing normal sync traffic (typical client: <10 req/min).
	rateLimiter := middleware.RateLimit(100, time.Minute)

	// RFC 6764: /.well-known/caldav must be reachable without auth so clients
	// can bootstrap discovery before they have a session.
	// Use 302 (not 308) — iOS has known issues with 308 on well-known redirects.
	router.HandleFunc("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, davPrefix+"/", http.StatusFound)
	})
	// go-webdav has no MKCALENDAR support (only MKCOL), so Apple Calendar's
	// calendar creation 405s. Intercept it and route to the backend.
	davHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "MKCALENDAR" {
			backend.HandleMkcalendar(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	}
	router.With(rateLimiter, davAuth).HandleFunc(davPrefix, davHandler)
	router.With(rateLimiter, davAuth).HandleFunc(davPrefix+"/*", davHandler)
}

func davAuthMiddleware(db *gorm.DB, authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := resolveUser(w, r, db, authService)
			if user == nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Agenda CalDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
		})
	}
}

// resolveUser authenticates a CalDAV request against porte.
//
// Three credentials reach this endpoint and all three are now porte's: the
// browser's session cookie, an email plus password, and an email plus API
// token — the last one being how an SSO user, who has no password at all,
// connects a calendar client.
//
// The password check is Verify rather than Login on purpose. A CalDAV client
// re-sends Basic credentials on every single request, so signing in here would
// write a session row per PROPFIND.
func resolveUser(w http.ResponseWriter, r *http.Request, db *gorm.DB, authService *auth.Service) *schemas.User {
	ctx := r.Context()

	load := func(id int64) *schemas.User {
		var u schemas.User
		if db.WithContext(ctx).First(&u, id).Error != nil {
			return nil
		}
		return &u
	}

	// The cookie or a bearer header, which is what a browser and the web UI
	// send. porte reads both, including the legacy `session` cookie this app
	// issued before it adopted porte.
	if id, err := authService.AuthenticateRequest(w, r); err == nil {
		if user := load(id); user != nil {
			return user
		}
	}

	rawEmail, secret, ok := r.BasicAuth()
	if !ok || rawEmail == "" || secret == "" {
		return nil
	}
	// iOS 18.4+ percent-encodes '@' as '%40' in Basic Auth usernames.
	email, _ := url.PathUnescape(rawEmail)
	if email == "" {
		email = rawEmail
	}

	if id, err := authService.VerifyPassword(ctx, email, secret); err == nil {
		if user := load(id); user != nil {
			return user
		}
	}

	// An API token, presented in the password field. It is verified as the
	// credential it is — a labelled porte session — by handing it to porte
	// as a bearer token, so the expiry and idle rules are the ones porte
	// applies everywhere else rather than a second implementation here.
	//
	// The address still has to match the account the token belongs to. A
	// valid token for one user must not authenticate a request that claims
	// to be another.
	bearer := r.Clone(ctx)
	bearer.Header.Set("Authorization", "Bearer "+secret)
	id, err := authService.AuthenticateRequest(w, bearer)
	if err != nil {
		return nil
	}
	user := load(id)
	if user == nil || !strings.EqualFold(user.Email, email) {
		return nil
	}
	return user
}
