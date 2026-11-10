// Package outbox makes Antenne events durable. A caller enqueues an event in
// the same database transaction as the business write that caused it, so the
// two commit or roll back together, and a Drainer later sends the queued rows
// in id order through the live client. Delivery is at least once: consumers
// deduplicate on the envelope's idempotency key.
//
// It depends on database/sql only. A GORM application passes gormDB.DB().
package outbox

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FacileStudio/antenne-client/go/catalog"
)

// Migration is the goose migration, Up and Down, that creates the
// antenne_outbox table. Copy it verbatim into the application's own ordered
// migrations directory under the next number.
//
//go:embed migration.sql
var Migration string

// ErrDuplicate is returned by Enqueue when the idempotency key is already
// queued. The caller's transaction stays usable.
var ErrDuplicate = errors.New("outbox: idempotency key already enqueued")

// ErrEmptyKey is returned by Enqueue when the idempotency key is empty.
var ErrEmptyKey = errors.New("outbox: idempotency key is required")

// Enqueue queues payload for channel inside tx, the caller's business
// transaction. A duplicate idempotency key returns ErrDuplicate without
// aborting tx.
func Enqueue(ctx context.Context, tx *sql.Tx, channel catalog.Channel, payload any, idempotencyKey string) error {
	if idempotencyKey == "" {
		return ErrEmptyKey
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: marshal payload: %w", err)
	}
	result, err := tx.ExecContext(ctx,
		`INSERT INTO antenne_outbox (channel, payload, idempotency_key) VALUES ($1, $2::jsonb, $3)
		 ON CONFLICT (idempotency_key) DO NOTHING`,
		string(channel), string(raw), idempotencyKey)
	if err != nil {
		return fmt.Errorf("outbox: enqueue: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("outbox: enqueue: %w", err)
	}
	if inserted == 0 {
		return ErrDuplicate
	}
	return nil
}
