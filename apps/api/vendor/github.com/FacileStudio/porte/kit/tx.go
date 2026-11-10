package kit

import (
	"context"
	"database/sql"
	stderrors "errors"

	"github.com/FacileStudio/tronc/errors"
)

const accountLockKey int64 = 0x706F7274656B6974

func (s *Store) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return errors.Internal("failed to open a transaction", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", accountLockKey); err != nil {
		return errors.Internal("failed to acquire the account lock", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.Internal("failed to commit the account change", err)
	}
	return nil
}

func scanID(row *sql.Row) (int64, error) {
	var id int64
	if err := row.Scan(&id); err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return id, nil
}
