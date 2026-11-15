// Package activities serves GET /api/activities: the signed-in user's own
// Sablier sessions and Echo calls, drawn beside the calendar.
package activities

import (
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// RegisterRoutes mounts the activity list behind the session guard.
func RegisterRoutes(router chi.Router, db *gorm.DB, authService *auth.Service) {
	c := &controller{service: &Service{db: db}}
	router.With(authService.RequireAuth).Get("/activities", c.list)
}
