package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	defaultInterval   = 2 * time.Second
	defaultBatchSize  = 100
	defaultRetention  = 7 * 24 * time.Hour
	defaultMaxBackoff = time.Minute
)

// Drainer sends queued outbox rows in id order and prunes old sent rows.
type Drainer struct {
	db         *sql.DB
	emit       func(channel string, payload any) error
	interval   time.Duration
	batchSize  int
	retention  time.Duration
	maxBackoff time.Duration
	onError    func(error)
}

// NewDrainer builds a Drainer over db that sends through emit. Pass the
// client's EmitNow, never Emit: Emit buffers in memory while disconnected, so
// the row would be marked sent while the event only sits in RAM.
func NewDrainer(db *sql.DB, emit func(channel string, payload any) error, opts ...Option) *Drainer {
	d := &Drainer{
		db:         db,
		emit:       emit,
		interval:   defaultInterval,
		batchSize:  defaultBatchSize,
		retention:  defaultRetention,
		maxBackoff: defaultMaxBackoff,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Run drains and prunes on every interval until ctx is done. A failed pass
// doubles the pause, up to the maximum backoff; a clean pass resets it.
func (d *Drainer) Run(ctx context.Context) {
	wait := d.interval
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = d.nextWait(wait, d.pass(ctx))
	}
}

func (d *Drainer) pass(ctx context.Context) error {
	_, drainErr := d.Drain(ctx)
	_, pruneErr := d.Prune(ctx)
	err := errors.Join(drainErr, pruneErr)
	if err != nil && d.onError != nil {
		d.onError(err)
	}
	return err
}

func (d *Drainer) nextWait(wait time.Duration, err error) time.Duration {
	if err == nil {
		return d.interval
	}
	return min(wait*2, max(d.maxBackoff, d.interval))
}

// Prune deletes rows sent longer ago than the retention period and returns
// how many it removed. Unsent rows are never pruned.
func (d *Drainer) Prune(ctx context.Context) (int64, error) {
	result, err := d.db.ExecContext(ctx,
		`DELETE FROM antenne_outbox WHERE sent_at < now() - make_interval(secs => $1)`,
		d.retention.Seconds())
	if err != nil {
		return 0, fmt.Errorf("outbox: prune: %w", err)
	}
	return result.RowsAffected()
}
