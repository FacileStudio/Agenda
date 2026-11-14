package mcp

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/url"
	"time"

	"github.com/FacileStudio/tronc/errors"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Statuses of an Outcome. A tool puts the status in its structured result,
// next to confirm_url when the status is StatusPending.
const (
	StatusPending  = "pending_confirmation"
	StatusDeclined = "declined"
	StatusDone     = "done"
)

// The states of a pending action on a confirm page.
const (
	StateReady   = "ready"
	StateChanged = "changed"
	StateMissing = "missing"
)

// Default lifetimes of a pending action and of the signed question.
const (
	DefaultPendingTTL = 7 * 24 * time.Hour
	DefaultAskTTL     = 15 * time.Minute
)

// ErrChanged is the refusal for a target that changed after the person was
// asked: they confirmed other content. The action closes.
var ErrChanged = errors.Failed("this changed after the user was asked, so nothing happened. Call the tool again to ask about the current version")

// errUnknownKind is the refusal for a kind no Kind was registered for.
var errUnknownKind = stderrors.New("mcp: unknown confirmation kind")

// Kind runs one kind of sensitive action.
type Kind struct {
	// Tool is the tool that asks for this kind, for the audit.
	Tool string

	// Fingerprint returns a digest of the target as it is now. Confirmations
	// stores it when asking and compares it before running, so an edit in
	// between closes the action instead of acting on content nobody saw.
	// A target that is gone returns its not_found error.
	Fingerprint func(ctx context.Context, a Action) (string, error)

	// Run performs the action.
	Run func(ctx context.Context, a Action) error
}

// ConfirmConfig describes the confirmations of one app.
type ConfirmConfig struct {
	// Store keeps pending actions; PostgresPending in production.
	Store Pending

	// Key signs the question state. It must stay secret and stable across
	// restarts and replicas.
	Key []byte

	// ConfirmURL is the app page where the person confirms, such as
	// https://agenda.facile.studio/pending. The action id is added as the
	// action query parameter.
	ConfirmURL string

	// Kinds maps each kind to how it runs.
	Kinds map[string]Kind

	// Audit receives one record per request, answer and run. Nil means
	// slog.Default().
	Audit *slog.Logger

	// PendingTTL is how long an action waits. Zero means DefaultPendingTTL.
	PendingTTL time.Duration

	// AskTTL is how long a signed question stays valid. Zero means
	// DefaultAskTTL.
	AskTTL time.Duration
}

// Outcome is where a sensitive action stands after a call.
type Outcome struct {
	// Status is StatusPending, StatusDeclined or StatusDone.
	Status string

	// Action is the pending action.
	Action Action

	// ConfirmURL is the link to give the person when Status is
	// StatusPending.
	ConfirmURL string

	// Message is the standard text for the model when Status is
	// StatusPending with a link, or StatusDeclined. A tool puts it in its
	// output's message.
	Message string
}

// Confirmations asks a person before a sensitive action runs: through MRTR
// input_required when the client can elicit, otherwise through a pending
// action and a confirm link the client hands over. Nothing runs without the
// person's answer.
type Confirmations struct {
	cfg    ConfirmConfig
	signer signer
}

// NewConfirmations checks cfg and fills its defaults.
func NewConfirmations(cfg ConfirmConfig) (*Confirmations, error) {
	link, err := url.Parse(cfg.ConfirmURL)
	switch {
	case cfg.Store == nil || len(cfg.Key) == 0 || len(cfg.Kinds) == 0:
		return nil, stderrors.New("mcp: confirmations need a store, a key and at least one kind")
	case err != nil || link.Scheme == "" || link.Host == "":
		return nil, stderrors.New("mcp: the confirm URL must be absolute")
	}
	for name, kind := range cfg.Kinds {
		if kind.Tool == "" || kind.Fingerprint == nil || kind.Run == nil {
			return nil, stderrors.New("mcp: kind " + name + " needs a tool, a fingerprint and a run")
		}
	}
	if cfg.Audit == nil {
		cfg.Audit = slog.Default()
	}
	if cfg.PendingTTL == 0 {
		cfg.PendingTTL = DefaultPendingTTL
	}
	if cfg.AskTTL == 0 {
		cfg.AskTTL = DefaultAskTTL
	}
	return &Confirmations{cfg: cfg, signer: signer{key: cfg.Key, ttl: cfg.AskTTL}}, nil
}

// Request opens a pending action of kind on target and asks the person
// question. When the person already has an open action of that kind on
// that target, with the same digest, asked by the same OAuth client, it
// asks about that one instead of opening another, so an agent that loops on
// a tool opens one action. Another client of the same person opens its own
// action. The reuse is best effort: two concurrent identical requests can
// still open two. A reused action keeps its first expiry, which the Waiting
// text counts down from. When the client can elicit, it returns the
// input_required result for the tool to return as is. Otherwise the result
// is nil and the Outcome holds StatusPending, the confirm link and the
// Waiting text. Either way nothing has run yet.
func (c *Confirmations) Request(ctx context.Context, req *sdk.CallToolRequest, kind, target, question string) (*sdk.CallToolResult, Outcome, error) {
	who, err := CallerOf(req)
	if err != nil {
		return nil, Outcome{}, err
	}
	runner, ok := c.cfg.Kinds[kind]
	if !ok {
		return nil, Outcome{}, errUnknownKind
	}
	action := Action{UserID: who.UserID, Kind: kind, Target: target, Summary: question, ClientID: who.ClientID}
	if action.Digest, err = runner.Fingerprint(ctx, action); err != nil {
		return nil, Outcome{}, err
	}
	if action, err = c.reuseOrCreate(ctx, action); err != nil {
		return nil, Outcome{}, err
	}
	c.audit(ctx, runner.Tool, "requested", who, target)
	if CanElicit(req) {
		return c.signer.ask(who.UserID, action.ID, question), Outcome{Status: StatusPending, Action: action}, nil
	}
	return nil, Outcome{Status: StatusPending, Action: action, ConfirmURL: c.Link(action.ID), Message: waiting(time.Until(action.ExpiresAt).Round(time.Hour))}, nil
}

// Resume handles the retry that carries the person's answer to a question
// about kind: a decline closes the action, with MessageDeclined in the
// Outcome, and an accept runs it once.
func (c *Confirmations) Resume(ctx context.Context, req *sdk.CallToolRequest, kind string) (Outcome, error) {
	who, err := CallerOf(req)
	if err != nil {
		return Outcome{}, err
	}
	id, answer, err := c.signer.answer(req, who.UserID)
	if err != nil {
		return Outcome{}, err
	}
	if answer == declined {
		if err := c.cfg.Store.Resolve(ctx, who.UserID, id, OutcomeDeclined); err != nil {
			return Outcome{}, err
		}
		c.audit(ctx, c.cfg.Kinds[kind].Tool, OutcomeDeclined, who, "")
		return Outcome{Status: StatusDeclined, Action: Action{ID: id, Kind: kind}, Message: MessageDeclined}, nil
	}
	action, err := c.run(ctx, who, id, kind)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{Status: StatusDone, Action: action}, nil
}

// Confirm runs one of who's actions from the confirm page. The page must
// refuse a delegated token first: an agent never confirms its own request.
func (c *Confirmations) Confirm(ctx context.Context, who Caller, id string) (Outcome, error) {
	action, err := c.run(ctx, who, id, "")
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{Status: StatusDone, Action: action}, nil
}

// Dismiss closes one of who's actions without running it.
func (c *Confirmations) Dismiss(ctx context.Context, who Caller, id string) error {
	if err := c.cfg.Store.Resolve(ctx, who.UserID, id, OutcomeDismissed); err != nil {
		return err
	}
	c.audit(ctx, "pending", OutcomeDismissed, who, id)
	return nil
}

// Open lists the person's waiting actions, newest first, at most
// OpenLimit of them.
func (c *Confirmations) Open(ctx context.Context, userID string) ([]Action, error) {
	actions, _, err := c.List(ctx, userID)
	return actions, err
}

// List is Open that also reports whether the person has more than
// OpenLimit waiting actions, which it leaves out.
func (c *Confirmations) List(ctx context.Context, userID string) ([]Action, bool, error) {
	actions, err := c.cfg.Store.Open(ctx, userID)
	if err != nil || len(actions) <= OpenLimit {
		return actions, false, err
	}
	return actions[:OpenLimit], true, nil
}
