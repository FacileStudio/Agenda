package settings

import (
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/api/settings", func(r chi.Router) {
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
