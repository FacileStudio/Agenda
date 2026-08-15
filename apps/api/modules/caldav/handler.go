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

// RegisterRoutes exposes the CalDAV server under /dav, plus the RFC 6764
// discovery endpoint.
//
// chi only knows standard HTTP methods by default, so the WebDAV/CalDAV verbs
// (PROPFIND, PROPPATCH, MKCALENDAR, REPORT, …) are registered first or
// HandleFunc would never match them. /.well-known/caldav must be reachable
// without auth so clients can bootstrap discovery before they have a session,
// and it redirects with 302 rather than 308 because iOS has known issues with
// 308 on well-known redirects. Requests are rate-limited to 100 per minute per
// IP to blunt Basic Auth brute force while letting normal sync traffic through
// (a typical client stays under 10 req/min). go-webdav has no MKCALENDAR
// support, only MKCOL, so Apple Calendar's calendar creation would 405; the
// davHandler intercepts it and routes it to the backend.
func RegisterRoutes(router chi.Router, db *gorm.DB, authService *auth.Service) {
	for _, m := range []string{"PROPFIND", "PROPPATCH", "MKCALENDAR", "REPORT", "COPY", "MOVE", "LOCK", "UNLOCK"} {
		chi.RegisterMethod(m)
	}

	backend := NewBackend(db)
	handler := &gocaldav.Handler{
		Backend: backend,
		Prefix:  davPrefix,
	}

	davAuth := davAuthMiddleware(db, authService)
	rateLimiter := middleware.RateLimit(100, time.Minute)

	router.HandleFunc("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, davPrefix+"/", http.StatusFound)
	})
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
//
// The cookie path covers both the current cookie and the legacy `session`
// cookie this app issued before it adopted porte, which porte still reads.
// The Basic Auth username is percent-decoded because iOS 18.4+ encodes '@' as
// '%40'. An API token is verified for what it is — a labelled porte session —
// by handing it to porte as a bearer token, so expiry and idle rules are the
// ones porte applies everywhere else; the address must still match the account
// the token belongs to, or a valid token would authenticate a request that
// claims to be somebody else.
func resolveUser(w http.ResponseWriter, r *http.Request, db *gorm.DB, authService *auth.Service) *schemas.User {
	ctx := r.Context()

	load := func(id int64) *schemas.User {
		var u schemas.User
		if db.WithContext(ctx).First(&u, id).Error != nil {
			return nil
		}
		return &u
	}

	if id, err := authService.AuthenticateRequest(w, r); err == nil {
		if user := load(id); user != nil {
			return user
		}
	}

	rawEmail, secret, ok := r.BasicAuth()
	if !ok || rawEmail == "" || secret == "" {
		return nil
	}
	email, _ := url.PathUnescape(rawEmail)
	if email == "" {
		email = rawEmail
	}

	if id, err := authService.VerifyPassword(ctx, email, secret); err == nil {
		if user := load(id); user != nil {
			return user
		}
	}

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
