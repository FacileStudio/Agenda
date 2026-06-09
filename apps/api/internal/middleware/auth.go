package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"api/internal/authcontext"
	"api/internal/errors"
	"api/internal/httpjson"
)

type Authenticator interface {
	Authenticate(context context.Context, authorization string) (string, any, error)
}

func extractToken(r *http.Request) string {
	if c, err := r.Cookie("session"); err == nil && c.Value != "" {
		return c.Value
	}

	if h := r.Header.Get("Authorization"); h != "" {
		return h
	}

	if t := r.URL.Query().Get("token"); t != "" {
		slog.Warn("auth via query param is deprecated, use cookie or header")
		return t
	}

	return ""
}

func RequireAuth(authService Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token := extractToken(request)
			if token == "" {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}

			userID, rawData, err := authService.Authenticate(request.Context(), token)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			data, ok := rawData.(interface{ GetEmail() string })
			if !ok || data == nil {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}

			authContext := authcontext.WithIdentity(request.Context(), authcontext.Identity{
				UserID: userID,
				Email:  data.GetEmail(),
			})
			next.ServeHTTP(w, request.WithContext(authContext))
		})
	}
}
