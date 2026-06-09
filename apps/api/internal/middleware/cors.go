package middleware

import (
	"net/http"
	"slices"
	"strings"
)

var corsAllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
var corsAllowedHeaders = []string{"Accept", "Authorization", "Content-Type"}

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if len(allowedOrigins) == 0 {
				next.ServeHTTP(w, request)
				return
			}

			origin := request.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, request)
				return
			}

			if !isAllowedOrigin(origin, allowedOrigins) {
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
			header.Set("Access-Control-Allow-Origin", origin)
			header.Set("Access-Control-Allow-Methods", strings.Join(corsAllowedMethods, ", "))
			header.Set("Access-Control-Allow-Headers", strings.Join(corsAllowedHeaders, ", "))
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Access-Control-Max-Age", "600")

			if request.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, request)
		})
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	return slices.Contains(allowedOrigins, "*") || slices.Contains(allowedOrigins, origin)
}
