package settings

import (
	"net/http"

	"api/internal/httpjson"
	"api/internal/middleware"
	"api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/settings", func(r chi.Router) {
		r.With(middleware.RequireAuth(authService)).Get("/", func(w http.ResponseWriter, req *http.Request) {
			resp, err := service.controller.getSettings(req.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		r.With(middleware.RequireAuth(authService)).Put("/", func(w http.ResponseWriter, req *http.Request) {
			var body UpdateRequest
			if err := httpjson.DecodeJSON(w, req, &body); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			resp, err := service.controller.updateSettings(req.Context(), &body)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
