package users

import (
	"net/http"

	"api/internal/httpjson"
	"api/internal/middleware"
	"api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/users", func(router chi.Router) {
		router.With(middleware.RequireAuth(authService)).Get("/", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.list(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Get("/me", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.me(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Patch("/me", func(w http.ResponseWriter, request *http.Request) {
			var req UpdateRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}

			resp, err := service.controller.updateMe(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Post("/me/avatar", func(w http.ResponseWriter, request *http.Request) {
			request.Body = http.MaxBytesReader(w, request.Body, 6<<20)
			resp, err := service.controller.uploadAvatar(request.Context(), request)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Delete("/me/avatar", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.deleteAvatar(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Get("/me/api-token", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.getApiToken(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Post("/me/api-token", func(w http.ResponseWriter, request *http.Request) {
			var req CreateApiTokenRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			resp, err := service.controller.createApiToken(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusCreated, resp)
		})

		router.With(middleware.RequireAuth(authService)).Delete("/me/api-token", func(w http.ResponseWriter, request *http.Request) {
			if err := service.controller.deleteApiToken(request.Context()); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
		})

		router.With(middleware.RequireAuth(authService)).Get("/{id}", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.get(request.Context(), chi.URLParam(request, "id"))
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
