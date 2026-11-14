package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"net/url"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

// State tells a confirm page whether an action still matches what the
// person was asked about: StateReady, StateChanged or StateMissing.
func (c *Confirmations) State(ctx context.Context, a Action) string {
	runner, ok := c.cfg.Kinds[a.Kind]
	if !ok {
		return StateMissing
	}
	digest, err := runner.Fingerprint(ctx, a)
	switch {
	case err != nil:
		return StateMissing
	case digest != a.Digest:
		return StateChanged
	default:
		return StateReady
	}
}

// Link is the confirm page URL for action id.
func (c *Confirmations) Link(id string) string {
	link, _ := url.Parse(c.cfg.ConfirmURL)
	query := link.Query()
	query.Set("action", id)
	link.RawQuery = query.Encode()
	return link.String()
}

// reuseOrCreate returns the person's open action of the same kind on the
// same target with the same digest, or stores action as a new one.
func (c *Confirmations) reuseOrCreate(ctx context.Context, action Action) (Action, error) {
	open, err := c.cfg.Store.Open(ctx, action.UserID)
	if err != nil {
		return action, err
	}
	for _, existing := range open {
		if existing.Kind == action.Kind && existing.Target == action.Target && existing.Digest == action.Digest {
			return existing, nil
		}
	}
	return c.create(ctx, action)
}

// create stores action under a fresh random id.
func (c *Confirmations) create(ctx context.Context, action Action) (Action, error) {
	token := make([]byte, 18)
	if _, err := rand.Read(token); err != nil {
		return action, errors.Internal("failed to create a pending action", err)
	}
	action.ID = base64.RawURLEncoding.EncodeToString(token)
	action.CreatedAt = time.Now().UTC()
	action.ExpiresAt = action.CreatedAt.Add(c.cfg.PendingTTL)
	return action, c.cfg.Store.Create(ctx, action)
}

// run claims one of who's actions, checks it still matches, runs it and
// records how it ended. A wrong kind or a failed run reopens it; a changed
// target closes it. kind empty accepts any kind.
func (c *Confirmations) run(ctx context.Context, who Caller, id, kind string) (Action, error) {
	action, err := c.cfg.Store.Claim(ctx, who.UserID, id)
	if err != nil {
		return action, err
	}
	runner, ok := c.cfg.Kinds[action.Kind]
	if !ok || (kind != "" && action.Kind != kind) {
		_ = c.cfg.Store.Release(ctx, id)
		return action, ErrConfirmation
	}
	who.ClientID = action.ClientID
	err = c.execute(ctx, runner, action)
	switch {
	case stderrors.Is(err, ErrChanged):
		_ = c.cfg.Store.Finish(ctx, id, OutcomeStale)
	case err != nil:
		_ = c.cfg.Store.Release(ctx, id)
	default:
		_ = c.cfg.Store.Finish(ctx, id, OutcomeDone)
	}
	c.audit(ctx, runner.Tool, runOutcome(err), who, action.Target)
	return action, err
}

// execute runs action if its target still has the digest it was asked
// with.
func (c *Confirmations) execute(ctx context.Context, runner Kind, action Action) error {
	digest, err := runner.Fingerprint(ctx, action)
	if err != nil {
		return err
	}
	if digest != action.Digest {
		return ErrChanged
	}
	return runner.Run(ctx, action)
}

// runOutcome is the audit outcome of a run that returned err.
func runOutcome(err error) string {
	switch {
	case err == nil:
		return OutcomeDone
	case stderrors.Is(err, ErrChanged):
		return OutcomeStale
	default:
		return "failed"
	}
}

// audit records one confirmation event.
func (c *Confirmations) audit(ctx context.Context, tool, outcome string, who Caller, target string) {
	Audit(ctx, c.cfg.Audit, Event{Tool: tool, Outcome: outcome, Caller: who, Target: target})
}
