package main

import (
	"log/slog"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/modules/ideas"

	"github.com/FacileStudio/porte/directory"
	porteKit "github.com/FacileStudio/porte/kit"

	"github.com/go-chi/chi/v5"
)

// invitationsConfig is what kit.MountInvitations needs to answer the user
// menu's pending space invitations from Registre: the directory credentials
// the deletion receiver also uses, and the identity store that holds each
// person's Registre subject. Without SSO there is nothing to ask, and nil.
func invitationsConfig(
	appEnv env.Config, identities porteKit.IdentityLister, appLogger *slog.Logger,
) *porteKit.InvitationsConfig {
	if appEnv.OIDC == nil {
		return nil
	}
	return &porteKit.InvitationsConfig{
		Directory: directory.Client{
			Issuer:       appEnv.OIDC.Issuer,
			ClientID:     appEnv.OIDC.ClientID,
			ClientSecret: appEnv.OIDC.ClientSecret,
		},
		Identities: identities,
		Logger:     appLogger,
	}
}

// mountOptional mounts the /api routes that exist only when their service is
// configured: the user menu's invitations and the Ideas page.
func mountOptional(r chi.Router, m mounts) error {
	if m.invitations != nil {
		if err := porteKit.MountInvitations(r.With(m.auth.RequireAuth), *m.invitations); err != nil {
			return err
		}
	}
	if m.ideas == nil {
		return nil
	}
	return ideas.RegisterRoutes(r, m.ideas, m.auth)
}
