package mcp

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/tronc/errors"
)

// Execer runs one statement: a *sql.DB, a *sql.Tx, or the ConnPool of a
// GORM transaction's Statement.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Erase deletes every action of userIDs.
func (p PostgresPending) Erase(ctx context.Context, userIDs ...string) error {
	return ErasePending(ctx, p.DB, userIDs...)
}

// ErasePending deletes every action of userIDs from the mcp_actions table
// through exec, so an account erasure can run it inside its own
// transaction. One statement per person keeps it free of driver-specific
// array types.
func ErasePending(ctx context.Context, exec Execer, userIDs ...string) error {
	for _, userID := range userIDs {
		if _, err := exec.ExecContext(ctx, `DELETE FROM mcp_actions WHERE user_id = $1`, userID); err != nil {
			return errors.Internal("failed to erase pending actions", err)
		}
	}
	return nil
}
