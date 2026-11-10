package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type row struct {
	id      int64
	channel string
	payload []byte
}

// Drain makes one pass: it locks up to a batch of unsent rows in id order,
// skipping rows another drainer holds, sends them, and marks each sent. The
// first emit failure stops the pass and leaves that row and every later one
// unsent. It returns how many rows were sent.
func (d *Drainer) Drain(ctx context.Context) (int, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("outbox: begin: %w", err)
	}
	defer tx.Rollback()
	rows, err := lockBatch(ctx, tx, d.batchSize)
	if err != nil {
		return 0, err
	}
	sent, emitErr := d.send(ctx, tx, rows)
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("outbox: commit: %w", err)
	}
	return sent, emitErr
}

func (d *Drainer) send(ctx context.Context, tx *sql.Tx, rows []row) (int, error) {
	for i, r := range rows {
		if err := d.emit(r.channel, json.RawMessage(r.payload)); err != nil {
			return i, fmt.Errorf("outbox: emit row %d on %s: %w", r.id, r.channel, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE antenne_outbox SET sent_at = now() WHERE id = $1`, r.id); err != nil {
			return i, fmt.Errorf("outbox: mark row %d sent: %w", r.id, err)
		}
	}
	return len(rows), nil
}

func lockBatch(ctx context.Context, tx *sql.Tx, limit int) ([]row, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, channel, payload FROM antenne_outbox WHERE sent_at IS NULL
		 ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox: select: %w", err)
	}
	defer rows.Close()
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.channel, &r.payload); err != nil {
			return nil, fmt.Errorf("outbox: scan: %w", err)
		}
		batch = append(batch, r)
	}
	return batch, rows.Err()
}
