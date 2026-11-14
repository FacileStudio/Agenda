package mcp

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

// Defaults of KeepPruned.
const (
	DefaultPruneEvery = time.Hour
	DefaultPruneAfter = 30 * 24 * time.Hour
)

// Pruner is a store that drops the actions it no longer needs, such as
// PostgresPending.
type Pruner interface {
	Prune(ctx context.Context, olderThan time.Duration) (int64, error)
}

// Prune deletes the actions resolved, or expired, more than olderThan ago,
// and reports how many went.
func (p PostgresPending) Prune(ctx context.Context, olderThan time.Duration) (int64, error) {
	return PrunePending(ctx, p.DB, olderThan)
}

// PrunePending is Prune through exec, in one statement.
func PrunePending(ctx context.Context, exec Execer, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).UTC()
	result, err := exec.ExecContext(ctx, `DELETE FROM mcp_actions
		WHERE (resolved_at IS NOT NULL AND resolved_at < $1) OR expires_at < $1`, cutoff)
	if err != nil {
		return 0, errors.Internal("failed to prune pending actions", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Internal("failed to prune pending actions", err)
	}
	return n, nil
}

// KeepPruned prunes store at once, then every interval, until ctx ends. A
// zero every means DefaultPruneEvery and a zero olderThan DefaultPruneAfter;
// a nil logger means slog.Default(). Run it in its own goroutine.
func KeepPruned(ctx context.Context, store Pruner, every, olderThan time.Duration, logger *slog.Logger) {
	if every <= 0 {
		every = DefaultPruneEvery
	}
	if olderThan <= 0 {
		olderThan = DefaultPruneAfter
	}
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if n, err := store.Prune(ctx, olderThan); err != nil {
			logger.WarnContext(ctx, "mcp: pruning pending actions failed", "error", err)
		} else if n > 0 {
			logger.InfoContext(ctx, "mcp: pruned pending actions", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
