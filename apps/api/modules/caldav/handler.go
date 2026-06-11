package caldav

import (
	"net/http"

	"api/internal/authcrypto"
	"api/schemas"

	gocaldav "github.com/emersion/go-webdav/caldav"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RegisterRoutes(router chi.Router, db *gorm.DB) {
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

	auth := davAuthMiddleware(db)

	// RFC 6764: /.well-known/caldav must be reachable without auth so clients
	// can bootstrap discovery before they have a session.
	router.HandleFunc("/.well-known/caldav", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, davPrefix+"/", http.StatusPermanentRedirect)
	})
	router.With(auth).HandleFunc(davPrefix, handler.ServeHTTP)
	router.With(auth).HandleFunc(davPrefix+"/*", handler.ServeHTTP)
}

func davAuthMiddleware(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := resolveUser(r, db)
			if user == nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Agenda CalDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
		})
	}
}

func resolveUser(r *http.Request, db *gorm.DB) *schemas.User {
	ctx := r.Context()

	// Session cookie (web UI / browser testing)
	if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
		hashed := authcrypto.HashToken(cookie.Value)
		var sess schemas.Session
		if db.WithContext(ctx).Where("token = ? AND expires_at > NOW()", hashed).First(&sess).Error == nil {
			var u schemas.User
			if db.WithContext(ctx).First(&u, sess.UserID).Error == nil {
				return &u
			}
		}
	}

	// HTTP Basic Auth — two accepted credential forms:
	//   1. email + account password  (password-based accounts)
	//   2. email + API token         (SSO users who have no password)
	if email, password, ok := r.BasicAuth(); ok && email != "" && password != "" {
		var u schemas.User
		if db.WithContext(ctx).Where("email = ?", email).First(&u).Error == nil {
			// Try password first
			if u.PasswordHash != "" && authcrypto.VerifyPassword(password, u.PasswordHash) {
				return &u
			}
			// Try API token (for SSO users)
			hashed := authcrypto.HashToken(password)
			var tok schemas.ApiToken
			if db.WithContext(ctx).Where("token = ? AND user_id = ?", hashed, u.ID).First(&tok).Error == nil {
				return &u
			}
		}
	}

	return nil
}
