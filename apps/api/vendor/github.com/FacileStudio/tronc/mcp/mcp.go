// Package mcp mounts a remote MCP server on a chi router: the go-sdk
// streamable HTTP transport in stateless mode (spec 2026-07-28), behind
// access tokens bound to the server's resource URI, with the RFC 9728
// metadata, an Origin check, a write-scope gate and a per-person rate limit.
//
// An app hands Mount its curated tools, built with NewTool, and the token
// verifier its auth kit provides (porte/mcpauth.Verifier in the suite).
// tronc never imports the auth kit. Tool names carry the app prefix, every
// tool declares its annotations, and tools/list comes out sorted by name.
//
// Sensitive actions go through Confirmations: an MRTR input_required question
// when the client can elicit, otherwise a pending action and a confirm link,
// with HMAC-signed state, single-use claims and a changed-target guard.
// Pagination and truncation helpers keep results under about 25k tokens, and
// Failure turns service errors into ones a model can act on.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"time"

	"github.com/FacileStudio/tronc/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultListTTL is how long a client may keep the tool list when
// Options.ListTTL is zero.
const DefaultListTTL = 10 * time.Minute

// Default rate limit per person when Options.RateLimit.Limit is zero: two
// calls a second sustained, with room for a burst of thirty.
const (
	defaultRate  = 2
	defaultBurst = 30
)

// namePattern is what an app name, and so a tool prefix and a scope prefix,
// may look like.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// Options describe one app's MCP server.
type Options struct {
	// Name is the app, such as "agenda". It is the server name, the prefix
	// every tool name must start with ("agenda_"), and the default scope
	// prefix.
	Name string

	// Title is the human name of the app, such as "Agenda".
	Title string

	// Version is the server version reported to clients.
	Version string

	// Instructions is what a client shows its model about this server.
	Instructions string

	// Resource is the canonical URI people paste into their MCP client, such
	// as https://agenda.facile.studio/mcp. Tokens are bound to it, and its
	// path is where Mount serves the endpoint.
	Resource string

	// Issuer is the authorization server named in the metadata.
	Issuer string

	// Verifier checks access tokens. It must refuse tokens issued for any
	// other resource.
	Verifier auth.TokenVerifier

	// Scopes are the read and write scopes. Zero means Name.read and
	// Name.write. A server whose tools are all read-only may set Read alone,
	// and then advertises only that scope.
	Scopes Scopes

	// RateLimit caps calls per person. A zero Limit means two a second with a
	// burst of thirty; a nil KeyFunc keys on the token's user.
	RateLimit middleware.RatelimitConfig

	// Audit receives one record per write tool call and per confirmation
	// event. It never sees arguments. Nil means slog.Default().
	Audit *slog.Logger

	// Origins are browser origins allowed besides the resource's own.
	// Requests without an Origin header, from native clients, always pass.
	Origins []string

	// Tools are the curated tools, built with NewTool.
	Tools []Tool

	// ListTTL is how long a client may cache list results. Zero means
	// DefaultListTTL. The scope is always private, because results are
	// served under one person's token.
	ListTTL time.Duration
}

// Scopes are an app's MCP scopes. A token with Write may call every tool; a
// token with only Read may call the read-only ones. Write is empty on a
// read-only server.
type Scopes struct {
	Read  string
	Write string
}

// List returns the scopes, read first, as scopes_supported wants them.
func (s Scopes) List() []string {
	if s.Write == "" {
		return []string{s.Read}
	}
	return []string{s.Read, s.Write}
}

// Mount builds the server and mounts it on router; see New and
// Server.Mount. It returns an error, and mounts nothing, when the options or
// a tool are invalid.
func Mount(router chi.Router, opts Options) error {
	server, err := New(opts)
	if err != nil {
		return err
	}
	server.Mount(router)
	return nil
}

// Server is a validated MCP server, ready to mount.
type Server struct {
	opts        Options
	path        string
	origin      string
	metadataURL string
	writes      map[string]bool
	sdk         *sdk.Server
}

// New validates opts and registers the tools, so an app can fail at boot
// and mount later.
func New(opts Options) (*Server, error) {
	return build(opts)
}

// Mount serves the endpoint at the resource's path on router, and its RFC
// 9728 metadata at the well-known path for that resource, plus the bare
// well-known path for clients that look only there. router is the root
// router, outside any /api prefix and outside CSRF middleware.
func (s *Server) Mount(router chi.Router) {
	router.Handle(s.path, s.endpoint())
	router.Handle(wellKnown+s.path, s.metadata())
	router.Handle(wellKnown, s.metadata())
}

// build validates opts and registers the tools.
func build(opts Options) (*Server, error) {
	if err := normalize(&opts); err != nil {
		return nil, err
	}
	resource, err := url.Parse(opts.Resource)
	if err != nil || resource.Scheme == "" || resource.Host == "" {
		return nil, fmt.Errorf("mcp: resource %q is not an absolute URI", opts.Resource)
	}
	s := &Server{
		opts:        opts,
		path:        resource.EscapedPath(),
		origin:      resource.Scheme + "://" + resource.Host,
		metadataURL: MetadataURL(resource),
		writes:      map[string]bool{},
	}
	if s.path == "" {
		s.path = "/"
	}
	s.sdk = sdk.NewServer(&sdk.Implementation{Name: opts.Name, Title: opts.Title, Version: opts.Version}, &sdk.ServerOptions{
		Instructions: opts.Instructions,
		SetCacheable: privateCache(opts.ListTTL),
	})
	if err := s.register(opts.Tools); err != nil {
		return nil, err
	}
	return s, nil
}

// normalize checks the required options and fills the defaults.
func normalize(opts *Options) error {
	if !namePattern.MatchString(opts.Name) {
		return fmt.Errorf("mcp: name %q must be lowercase letters and digits", opts.Name)
	}
	if opts.Verifier == nil {
		return errors.New("mcp: a token verifier is required")
	}
	if opts.Issuer == "" {
		return errors.New("mcp: an issuer is required")
	}
	if opts.Scopes == (Scopes{}) {
		opts.Scopes = Scopes{Read: opts.Name + ".read", Write: opts.Name + ".write"}
	}
	if opts.Scopes.Read == "" {
		return errors.New("mcp: a read scope is required")
	}
	if opts.ListTTL == 0 {
		opts.ListTTL = DefaultListTTL
	}
	if opts.Audit == nil {
		opts.Audit = slog.Default()
	}
	if opts.RateLimit.Limit <= 0 {
		opts.RateLimit.Limit, opts.RateLimit.Burst = defaultRate, max(opts.RateLimit.Burst, defaultBurst)
	}
	if opts.RateLimit.KeyFunc == nil {
		opts.RateLimit.KeyFunc = tokenUser
	}
	return nil
}

// privateCache marks every cacheable result private with ttl.
func privateCache(ttl time.Duration) func(context.Context, sdk.Request, *sdk.Cacheable) {
	return func(_ context.Context, _ sdk.Request, c *sdk.Cacheable) {
		c.CacheScope = "private"
		c.TTLMs = int(ttl.Milliseconds())
	}
}

// endpoint is the MCP handler: Origin check, bearer token, write scope,
// rate limit, then the stateless transport, which answers 405 to GET and
// DELETE.
func (s *Server) endpoint() http.Handler {
	transport := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s.sdk }, &sdk.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
	gated := middleware.Ratelimit(s.opts.RateLimit)(transport)
	if len(s.writes) > 0 {
		gated = writeGate(s.writes, s.opts.Scopes.Write, s.metadataURL)(gated)
	}
	authed := requireToken(s.opts.Verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.metadataURL,
		Scopes:              []string{s.opts.Scopes.Read},
	})(gated)
	return originGuard(slices.Compact(append([]string{s.origin}, s.opts.Origins...)))(authed)
}

// metadata serves the RFC 9728 document for the resource.
func (s *Server) metadata() http.Handler {
	return auth.ProtectedResourceMetadataHandler(ProtectedResourceMetadata(s.opts.Resource, s.opts.Issuer, s.opts.Scopes.List()))
}

// tokenUser keys the rate limit on the person the token speaks for, or on
// the remote address when there is none.
func tokenUser(r *http.Request) string {
	if info := auth.TokenInfoFromContext(r.Context()); info != nil && info.UserID != "" {
		return "user:" + info.UserID
	}
	return "addr:" + r.RemoteAddr
}
