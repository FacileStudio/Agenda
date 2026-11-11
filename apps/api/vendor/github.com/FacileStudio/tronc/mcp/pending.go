package mcp

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

// Outcomes a pending action is resolved with.
const (
	OutcomeRunning   = "running"
	OutcomeDone      = "done"
	OutcomeDeclined  = "declined"
	OutcomeDismissed = "dismissed"
	OutcomeStale     = "stale"
)

// ErrGone is the refusal for an action that ran, was dismissed, expired or
// was never this person's.
var ErrGone = errors.NotFound("this action was already confirmed, dismissed or expired")

// Action is one sensitive action waiting for a person.
type Action struct {
	// ID is a random token; it appears in the confirm link.
	ID string `json:"id"`

	// UserID is the person who must confirm.
	UserID string `json:"user_id"`

	// Kind names the action, such as "delete_event"; it picks the Kind that
	// runs it.
	Kind string `json:"kind"`

	// Target is the app's reference to what the action touches, such as an
	// event id.
	Target string `json:"target"`

	// Digest fingerprints the target as it was when the person was asked.
	Digest string `json:"-"`

	// Summary is the question the person was asked, for the confirm page.
	Summary string `json:"summary"`

	// ClientID is the OAuth client that asked.
	ClientID string `json:"client_id"`

	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Pending stores actions waiting for a person. Claim must be atomic: of two
// claims of one open action, exactly one succeeds. That makes every
// confirmation single use, whether it comes from the client or from the
// confirm page.
type Pending interface {
	// Create stores a new action; ID, CreatedAt and ExpiresAt are set.
	Create(ctx context.Context, action Action) error

	// Open lists the person's unresolved, unexpired actions, newest first.
	Open(ctx context.Context, userID string) ([]Action, error)

	// Claim marks one of the person's open actions as running and returns
	// it, or ErrGone.
	Claim(ctx context.Context, userID, id string) (Action, error)

	// Finish records the outcome of a claimed action.
	Finish(ctx context.Context, id, outcome string) error

	// Release reopens a claimed action whose run failed.
	Release(ctx context.Context, id string) error

	// Resolve closes one of the person's open actions without running it,
	// or returns ErrGone.
	Resolve(ctx context.Context, userID, id, outcome string) error

	// Erase deletes every action of the people, resolved or not, for an
	// account erasure.
	Erase(ctx context.Context, userIDs ...string) error
}

// PendingSchema creates the table PostgresPending uses. It is idempotent and
// additive: an app runs it with its own migrations.
const PendingSchema = `
CREATE TABLE IF NOT EXISTS mcp_actions (
	id          text        PRIMARY KEY,
	user_id     text        NOT NULL,
	kind        text        NOT NULL,
	target      text        NOT NULL,
	digest      text        NOT NULL,
	summary     text        NOT NULL DEFAULT '',
	client_id   text        NOT NULL DEFAULT '',
	created_at  timestamptz NOT NULL DEFAULT now(),
	expires_at  timestamptz NOT NULL,
	resolved_at timestamptz,
	outcome     text        NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS mcp_actions_open_idx
	ON mcp_actions (user_id, created_at DESC) WHERE resolved_at IS NULL;
`

// actionColumns is the column list every read selects, in scan order.
const actionColumns = `id, user_id, kind, target, digest, summary, client_id, created_at, expires_at`

// PostgresPending is Pending on the mcp_actions table of PendingSchema,
// over any database/sql Postgres driver.
type PostgresPending struct {
	DB *sql.DB
}

// Create inserts action.
func (p PostgresPending) Create(ctx context.Context, a Action) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO mcp_actions (`+actionColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, a.UserID, a.Kind, a.Target, a.Digest, a.Summary, a.ClientID, a.CreatedAt, a.ExpiresAt)
	if err != nil {
		return errors.Internal("failed to create a pending action", err)
	}
	return nil
}

// Open lists userID's open actions, newest first.
func (p PostgresPending) Open(ctx context.Context, userID string) ([]Action, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT `+actionColumns+` FROM mcp_actions
		WHERE user_id = $1 AND resolved_at IS NULL AND expires_at > now() ORDER BY created_at DESC, id`, userID)
	if err != nil {
		return nil, errors.Internal("failed to list pending actions", err)
	}
	defer func() { _ = rows.Close() }()
	actions := []Action{}
	for rows.Next() {
		action, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Internal("failed to list pending actions", err)
	}
	return actions, nil
}

// Claim marks one open action running in a single UPDATE, which is what
// makes it single use.
func (p PostgresPending) Claim(ctx context.Context, userID, id string) (Action, error) {
	row := p.DB.QueryRowContext(ctx, `UPDATE mcp_actions SET resolved_at = now(), outcome = $3
		WHERE id = $1 AND user_id = $2 AND resolved_at IS NULL AND expires_at > now() RETURNING `+actionColumns,
		id, userID, OutcomeRunning)
	action, err := scanAction(row)
	if stderrors.Is(err, ErrGone) {
		return Action{}, ErrGone
	}
	return action, err
}

// Finish records the outcome of a claimed action.
func (p PostgresPending) Finish(ctx context.Context, id, outcome string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE mcp_actions SET outcome = $2 WHERE id = $1`, id, outcome)
	if err != nil {
		return errors.Internal("failed to finish a pending action", err)
	}
	return nil
}

// Release reopens a claimed action that is still running.
func (p PostgresPending) Release(ctx context.Context, id string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE mcp_actions SET resolved_at = NULL, outcome = ''
		WHERE id = $1 AND outcome = $2`, id, OutcomeRunning)
	if err != nil {
		return errors.Internal("failed to release a pending action", err)
	}
	return nil
}

// Resolve closes one of userID's open actions.
func (p PostgresPending) Resolve(ctx context.Context, userID, id, outcome string) error {
	result, err := p.DB.ExecContext(ctx, `UPDATE mcp_actions SET resolved_at = now(), outcome = $3
		WHERE id = $1 AND user_id = $2 AND resolved_at IS NULL`, id, userID, outcome)
	if err != nil {
		return errors.Internal("failed to resolve a pending action", err)
	}
	if n, err := result.RowsAffected(); err != nil || n == 0 {
		return ErrGone
	}
	return nil
}

// scanner is a *sql.Row or *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanAction reads one row of actionColumns. No row is ErrGone.
func scanAction(row scanner) (Action, error) {
	var a Action
	err := row.Scan(&a.ID, &a.UserID, &a.Kind, &a.Target, &a.Digest, &a.Summary, &a.ClientID, &a.CreatedAt, &a.ExpiresAt)
	switch {
	case stderrors.Is(err, sql.ErrNoRows):
		return Action{}, ErrGone
	case err != nil:
		return Action{}, errors.Internal("failed to read a pending action", err)
	}
	return a, nil
}
