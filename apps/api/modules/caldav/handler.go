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
	backend := NewBackend(db)
	handler := &gocaldav.Handler{
		Backend: backend,
		Prefix:  davPrefix,
	}

	auth := davAuthMiddleware(db)

	router.With(auth).HandleFunc("/.well-known/caldav", handler.ServeHTTP)
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

	// HTTP Basic Auth (native CalDAV clients: Apple Calendar, Thunderbird, DAVx⁵)
	if email, password, ok := r.BasicAuth(); ok && email != "" {
		var u schemas.User
		if db.WithContext(ctx).Where("email = ?", email).First(&u).Error == nil {
			if authcrypto.VerifyPassword(password, u.PasswordHash) {
				return &u
			}
		}
	}

	return nil
}
