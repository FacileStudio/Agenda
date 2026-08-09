package auth

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts what porte does not own.
//
// /auth/config, /auth/logout, /auth/sync-profile and the whole OIDC flow come
// from porte's session manager and its OIDC kit, mounted in main.go. What is
// left here is the local password path, which keeps Agenda's own
// {user_id, token} response shape.
//
// Under SSO_ONLY the credential routes are not registered rather than
// rejected, so there is no endpoint left to probe for an account. That is the
// behaviour this app already had.
func RegisterRoutes(router chi.Router, service *Service, appEnv env.Config) {
	if appEnv.SSOOnly {
		return
	}
	router.Route("/auth", func(router chi.Router) {
		router.With(middleware.RateLimit(3, time.Minute)).Post("/register", func(w http.ResponseWriter, request *http.Request) {
			var req RegisterRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			resp, err := service.controller.register(w, request, &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusCreated, resp)
		})

		router.With(middleware.RateLimit(10, time.Minute)).Post("/login", func(w http.ResponseWriter, request *http.Request) {
			var req LoginRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			resp, err := service.controller.login(w, request, &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
