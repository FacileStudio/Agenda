package mcp

import (
	"errors"
	"log/slog"
	"net/url"

	"github.com/FacileStudio/tronc/middleware"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// DefaultPerMinute is the tool calls per person and minute when
// AppConfig.PerMinute is zero.
const DefaultPerMinute = 120

// PendingPath is the path of the confirm page on the app's origin.
const PendingPath = "/pending"

// AppConfig is what an app's main hands its MCP module: the deployment
// side of the server.
type AppConfig struct {
	// Resource is the canonical URI people paste into their MCP client, such
	// as https://agenda.facile.studio/mcp. Registre binds tokens to it.
	Resource string

	// Issuer is Registre, the only authorization server.
	Issuer string

	// Version is the server version reported to clients.
	Version string

	// Origins are the browser origins allowed besides the resource's own.
	Origins []string

	// StateKey signs the confirmation state handed to clients.
	StateKey []byte

	// Verifier checks access tokens, normally mcpauth.Verifier.
	Verifier auth.TokenVerifier

	// Pending stores the actions waiting for a person, normally
	// PostgresPending.
	Pending Pending

	// PerMinute caps tool calls per person, with a burst of as many. Zero
	// means DefaultPerMinute.
	PerMinute int

	// Logger receives the write audit, which Journal collects. Nil means
	// slog.Default().
	Logger *slog.Logger
}

// AppSpec is what the app's MCP module itself declares.
type AppSpec struct {
	// Name, Title and Instructions are as in Options.
	Name         string
	Title        string
	Instructions string

	// Kinds are the sensitive actions. None means no confirmations.
	Kinds map[string]Kind

	// Tools are the curated tools.
	Tools []Tool

	// Scopes are as in Options. Zero means Name.read and Name.write; a
	// server whose tools are all read-only may set Read alone.
	Scopes Scopes
}

// App is one app's validated MCP server and the confirmations behind its
// confirm page.
type App struct {
	// Server is ready to mount on the root router.
	Server *Server

	// Confirmations asks before sensitive actions, with its confirm page at
	// URL + PendingPath. It is nil when the spec has no kinds.
	Confirmations *Confirmations

	// URL is the resource's origin, such as https://agenda.facile.studio.
	URL string
}

// NewApp builds the server and its confirmations from cfg and spec. Tools
// and kinds may close over a value the caller fills from the App afterwards:
// they run only once the server is mounted.
func NewApp(cfg AppConfig, spec AppSpec) (*App, error) {
	resource, err := url.Parse(cfg.Resource)
	if err != nil || resource.Scheme == "" || resource.Host == "" {
		return nil, errors.New("mcp: the resource must be an absolute URI")
	}
	if cfg.PerMinute == 0 {
		cfg.PerMinute = DefaultPerMinute
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	app := &App{URL: resource.Scheme + "://" + resource.Host}
	if len(spec.Kinds) > 0 {
		app.Confirmations, err = NewConfirmations(ConfirmConfig{
			Store: cfg.Pending, Key: cfg.StateKey, ConfirmURL: app.URL + PendingPath, Kinds: spec.Kinds, Audit: cfg.Logger,
		})
		if err != nil {
			return nil, err
		}
	}
	app.Server, err = New(Options{
		Name: spec.Name, Title: spec.Title, Version: cfg.Version, Instructions: spec.Instructions,
		Resource: cfg.Resource, Issuer: cfg.Issuer, Verifier: cfg.Verifier, Origins: cfg.Origins, Audit: cfg.Logger,
		RateLimit: middleware.RatelimitConfig{Limit: float64(cfg.PerMinute) / 60, Burst: cfg.PerMinute},
		Tools:     spec.Tools, Scopes: spec.Scopes,
	})
	if err != nil {
		return nil, err
	}
	return app, nil
}
