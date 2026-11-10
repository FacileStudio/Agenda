// Package ideas serves Bulle's Ideas API to Agenda's own client.
//
// The routes proxy Bulle with a token exchanged for the signed-in user. The
// package decides the two things Bulle cannot know: which Registre space is
// active, taken from a space_id the caller is a member of, and whether that
// space or person has ai.access.
package ideas

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/middleware"
	"github.com/FacileStudio/tronc/bulle"

	"github.com/go-chi/chi/v5"
)

// App is Agenda's name in Bulle's idea enum.
const App = "agenda"

// Config wires the routes to Bulle. Plans decides ai.access; a nil Plans
// leaves the decision to Bulle.
type Config struct {
	URL        string
	UpgradeURL string
	Token      bulle.TokenSource
	Forget     func(context.Context)
	Plans      Plans
}

// RegisterRoutes serves /ideas behind authentication on router.
func RegisterRoutes(router chi.Router, service *Service, authService middleware.Authenticator) error {
	var mountErr error
	router.Group(func(group chi.Router) {
		group.Use(middleware.RequireAuth(authService))
		group.Use(service.resolve)
		mountErr = bulle.MountIdeas(group, service.options())
	})
	return mountErr
}

func (s *Service) options() bulle.Options {
	return bulle.Options{
		App:         App,
		URL:         s.cfg.URL,
		TokenSource: s.cfg.Token,
		Forget:      s.cfg.Forget,
		Space:       func(r *http.Request) string { return scopeOf(r).space },
		AIAccess:    func(r *http.Request) bool { return scopeOf(r).aiAccess },
		UpgradeURL:  s.cfg.UpgradeURL,
	}
}
