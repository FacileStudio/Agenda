package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The TokenInfo.Extra keys Caller reads. They match porte/mcpauth's
// ExtraClientID and ExtraActor; a verifier from another kit sets the same
// keys, or leaves them empty.
const (
	ExtraClientID = "client_id"
	ExtraActor    = "actor"
)

// Audit outcomes of a tool call.
const (
	outcomeOK       = "ok"
	outcomeError    = "error"
	outcomeQuestion = "input_required"
)

// errNoToken is the refusal for a call that reached a tool without a
// verified token, which Mount's middleware should make impossible.
var errNoToken = errors.New("this request carries no valid access token. Reconnect the app")

// Annotations are a tool's hints for the client's permission UI. Every tool
// declares all four: clients treat a missing destructive or open-world hint
// as true. They are UX, never the gate: the scopes and Confirmations are.
type Annotations struct {
	// ReadOnly tools change nothing. Every other tool needs the write scope.
	ReadOnly bool

	// Destructive tools may delete or overwrite. A read-only tool is never
	// destructive.
	Destructive bool

	// Idempotent tools have no further effect when called again with the
	// same arguments.
	Idempotent bool

	// OpenWorld tools read or reach content from outside the app, such as
	// mail from other senders or web pages.
	OpenWorld bool
}

// ToolSpec describes one tool.
type ToolSpec struct {
	// Name is <app>_<verb> or <app>_<resource>_<verb>, lowercase.
	Name string

	// Title is the human name shown in clients.
	Title string

	// Description tells the model when to call the tool and what it returns.
	Description string

	// Annotations are required, so each tool states its hints on purpose.
	Annotations *Annotations
}

// Tool is a curated tool ready for Mount.
type Tool struct {
	spec     ToolSpec
	register func(s *sdk.Server, t *sdk.Tool, guard guard)
}

// guard is what a registered handler checks and reports around each call.
type guard struct {
	writeScope string
	audit      *slog.Logger
}

// NewTool builds a tool from spec and a typed handler. The input and output
// schemas come from In and Out. Mount checks the spec; the handler runs only
// for a caller with a verified token, and for a write tool only when the
// token holds the write scope.
func NewTool[In, Out any](spec ToolSpec, handler sdk.ToolHandlerFor[In, Out]) Tool {
	return Tool{spec: spec, register: func(s *sdk.Server, t *sdk.Tool, g guard) {
		sdk.AddTool(s, t, guarded(spec, g, handler))
	}}
}

// guarded wraps handler with the scope check and, for a write tool, one
// audit record per call.
func guarded[In, Out any](spec ToolSpec, g guard, handler sdk.ToolHandlerFor[In, Out]) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		var zero Out
		who, err := CallerOf(req)
		if err != nil {
			return nil, zero, err
		}
		if spec.Annotations.ReadOnly {
			return handler(ctx, req, in)
		}
		if !who.Has(g.writeScope) {
			return nil, zero, fmt.Errorf("this tool needs the %s scope. Reconnect the app and grant write access", g.writeScope)
		}
		result, out, err := handler(ctx, req, in)
		Audit(ctx, g.audit, Event{Tool: spec.Name, Outcome: callOutcome(result, err), Caller: who})
		return result, out, err
	}
}

// callOutcome is the audit outcome of a tool call.
func callOutcome(result *sdk.CallToolResult, err error) string {
	switch {
	case err != nil || (result != nil && result.IsError):
		return outcomeError
	case result != nil && len(result.InputRequests) > 0:
		return outcomeQuestion
	default:
		return outcomeOK
	}
}

// register checks every tool and adds it to the server. tools/list returns
// them sorted by name, whatever the order here.
func (s *Server) register(tools []Tool) error {
	prefix := s.opts.Name + "_"
	seen := map[string]bool{}
	for _, tool := range tools {
		if err := check(tool.spec, prefix, seen); err != nil {
			return err
		}
		a := tool.spec.Annotations
		if !a.ReadOnly {
			s.writes[tool.spec.Name] = true
		}
		destructive, openWorld := a.Destructive, a.OpenWorld
		tool.register(s.sdk, &sdk.Tool{
			Name: tool.spec.Name, Title: tool.spec.Title, Description: tool.spec.Description,
			Annotations: &sdk.ToolAnnotations{
				Title: tool.spec.Title, ReadOnlyHint: a.ReadOnly, DestructiveHint: &destructive,
				IdempotentHint: a.Idempotent, OpenWorldHint: &openWorld,
			},
		}, guard{writeScope: s.opts.Scopes.Write, audit: s.opts.Audit})
	}
	return nil
}

// check refuses a tool without the app prefix, a lowercase name, a title,
// a description and annotations, or with a name already taken.
func check(spec ToolSpec, prefix string, seen map[string]bool) error {
	rest, ok := strings.CutPrefix(spec.Name, prefix)
	switch {
	case !ok || rest == "" || strings.Trim(rest, "abcdefghijklmnopqrstuvwxyz0123456789_") != "":
		return fmt.Errorf("mcp: tool %q must be named %s<verb>, in lowercase letters, digits and _", spec.Name, prefix)
	case seen[spec.Name]:
		return fmt.Errorf("mcp: tool %q is registered twice", spec.Name)
	case spec.Title == "" || spec.Description == "":
		return fmt.Errorf("mcp: tool %q needs a title and a description", spec.Name)
	case spec.Annotations == nil:
		return fmt.Errorf("mcp: tool %q needs its annotations", spec.Name)
	case spec.Annotations.ReadOnly && spec.Annotations.Destructive:
		return fmt.Errorf("mcp: tool %q cannot be read-only and destructive", spec.Name)
	}
	seen[spec.Name] = true
	return nil
}

// Caller is the person a tool call acts for, and the client acting.
type Caller struct {
	// UserID is the app's id of the person, as the verifier resolved it.
	UserID string

	// ClientID is the OAuth client the token was issued to.
	ClientID string

	// Actor is the agent acting for the person on a delegated token, and
	// empty otherwise.
	Actor string

	// Scopes are the scopes the token grants.
	Scopes []string
}

// CallerOf reads the caller from the verified token of a tool call.
func CallerOf(req *sdk.CallToolRequest) (Caller, error) {
	if req == nil || req.Extra == nil {
		return Caller{}, errNoToken
	}
	return callerFrom(req.Extra.TokenInfo)
}

// callerFrom reads a caller from token info.
func callerFrom(info *auth.TokenInfo) (Caller, error) {
	if info == nil || info.UserID == "" {
		return Caller{}, errNoToken
	}
	clientID, _ := info.Extra[ExtraClientID].(string)
	actor, _ := info.Extra[ExtraActor].(string)
	return Caller{UserID: info.UserID, ClientID: clientID, Actor: actor, Scopes: info.Scopes}, nil
}

// Has reports whether the caller's token grants scope.
func (c Caller) Has(scope string) bool {
	return slices.Contains(c.Scopes, scope)
}
