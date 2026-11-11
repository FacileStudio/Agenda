package mcptest

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/FacileStudio/tronc/mcp"
)

// Pending is an in-memory mcp.Pending for tests. Claims are atomic under one
// mutex, like the UPDATE of mcp.PostgresPending.
type Pending struct {
	mu      sync.Mutex
	actions map[string]*row
}

// row is one stored action and its resolution.
type row struct {
	action   mcp.Action
	resolved bool
	outcome  string
}

// NewPending returns an empty store.
func NewPending() *Pending {
	return &Pending{actions: map[string]*row{}}
}

// Create stores action.
func (p *Pending) Create(_ context.Context, action mcp.Action) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.actions[action.ID] = &row{action: action}
	return nil
}

// Open lists userID's open actions, newest first.
func (p *Pending) Open(_ context.Context, userID string) ([]mcp.Action, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	open := []mcp.Action{}
	for _, r := range p.actions {
		if r.open(userID, time.Now()) {
			open = append(open, r.action)
		}
	}
	slices.SortFunc(open, func(a, b mcp.Action) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(a.ID, b.ID))
	})
	return open, nil
}

// Claim marks one open action running.
func (p *Pending) Claim(_ context.Context, userID, id string) (mcp.Action, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.actions[id]
	if !ok || !r.open(userID, time.Now()) {
		return mcp.Action{}, mcp.ErrGone
	}
	r.resolved, r.outcome = true, mcp.OutcomeRunning
	return r.action, nil
}

// Finish records an outcome.
func (p *Pending) Finish(_ context.Context, id, outcome string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r, ok := p.actions[id]; ok {
		r.outcome = outcome
	}
	return nil
}

// Release reopens a running action.
func (p *Pending) Release(_ context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r, ok := p.actions[id]; ok && r.outcome == mcp.OutcomeRunning {
		r.resolved, r.outcome = false, ""
	}
	return nil
}

// Resolve closes an open action.
func (p *Pending) Resolve(_ context.Context, userID, id, outcome string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.actions[id]
	if !ok || r.action.UserID != userID || r.resolved {
		return mcp.ErrGone
	}
	r.resolved, r.outcome = true, outcome
	return nil
}

// Erase deletes every action of userIDs.
func (p *Pending) Erase(_ context.Context, userIDs ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, r := range p.actions {
		if slices.Contains(userIDs, r.action.UserID) {
			delete(p.actions, id)
		}
	}
	return nil
}

// Outcome is the recorded outcome of action id, for assertions.
func (p *Pending) Outcome(id string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r, ok := p.actions[id]; ok {
		return r.outcome
	}
	return ""
}

// open reports whether the row is userID's, unresolved and unexpired.
func (r *row) open(userID string, now time.Time) bool {
	return r.action.UserID == userID && !r.resolved && now.Before(r.action.ExpiresAt)
}
