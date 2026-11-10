package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/modules/mcp"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/porte/oidc"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// newMCP builds the MCP server when MCP_RESOURCE is set, SSO is on and
// ENCRYPTION_KEY signs the confirmations, and nothing otherwise. The
// verifier runs Registre discovery here, so a wrong issuer fails at boot like
// the OIDC kit does. A person's first MCP call creates their account through
// the OIDC kit, as a first web login would.
func newMCP(
	ctx context.Context, sqlDB *sql.DB, appEnv env.Config, calendarService *calendars.Service,
	eventService *events.Service, identities porte.IdentityStore, kit *oidc.Kit, appLogger *slog.Logger,
) (*mcp.Service, error) {
	if appEnv.OIDC == nil || appEnv.MCP.Resource == "" || len(appEnv.MCP.StateKey) == 0 {
		return nil, nil
	}
	verifier, err := mcpauth.Verifier(ctx, mcpauth.Config{
		Issuer:      appEnv.OIDC.Issuer,
		Resource:    appEnv.MCP.Resource,
		ScopePrefix: "agenda",
		Identities:  identities,
		Provision:   true,
		Accounts:    kit,
		Logger:      appLogger,
	})
	if err != nil {
		return nil, err
	}
	appLogger.Info("mcp server on", slog.String("resource", appEnv.MCP.Resource))
	return mcp.NewService(calendarService, eventService, mcp.Config{
		Resource: appEnv.MCP.Resource,
		Issuer:   appEnv.OIDC.Issuer,
		Version:  version,
		Origins:  appEnv.CORSAllowedOrigins,
		StateKey: appEnv.MCP.StateKey,
		Verifier: verifier,
		Pending:  mcpkit.PostgresPending{DB: sqlDB},
		Logger:   appLogger,
	})
}
