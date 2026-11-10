package main

import (
	"context"
	"log/slog"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/internal/plans"
	"github.com/FacileStudio/Agenda/apps/api/modules/ideas"
	"github.com/FacileStudio/Agenda/apps/api/modules/spaces"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/exchange"

	"gorm.io/gorm"
)

// newIdeas builds the Ideas service when BULLE_URL is set and SSO is on, and
// nothing otherwise. Agenda acts as the signed-in user through a Registre token
// exchange for audience bulle, over the identity store that seals the Registre
// tokens. The plan gate follows every directory snapshot from then on.
func newIdeas(
	ctx context.Context, db *gorm.DB, appEnv env.Config, identities porte.IdentityStore,
	directory *spaces.Directory, appLogger *slog.Logger,
) (*ideas.Service, error) {
	if appEnv.OIDC == nil || strings.TrimSpace(appEnv.BulleURL) == "" {
		return nil, nil
	}
	exchanger, err := exchange.New(exchange.Config{
		Issuer:       appEnv.OIDC.Issuer,
		ClientID:     appEnv.OIDC.ClientID,
		ClientSecret: appEnv.OIDC.ClientSecret,
		Identities:   identities,
		Logger:       appLogger,
	})
	if err != nil {
		return nil, err
	}
	gate := plans.New(db, appEnv.OIDC.Issuer, appLogger)
	if err := gate.Load(ctx); err != nil {
		return nil, err
	}
	if directory != nil {
		directory.OnSnapshot(gate.Apply)
	}
	source := exchanger.Source(exchange.Target{Audience: "bulle", Scopes: []string{"bulle.read", "bulle.write"}})
	return ideas.NewService(db, ideas.Config{
		URL:        strings.TrimSpace(appEnv.BulleURL),
		UpgradeURL: strings.TrimRight(appEnv.OIDC.Issuer, "/") + "/spaces",
		Token:      source.Token,
		Forget:     source.Forget,
		Plans:      gate,
	}), nil
}
