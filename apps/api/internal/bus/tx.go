package bus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// transactionOf is the *sql.Tx under a GORM transaction. Enqueueing outside
// one would publish a change that might never commit, so it is refused.
func transactionOf(tx *gorm.DB) (*sql.Tx, error) {
	sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx)
	if !ok {
		return nil, fmt.Errorf("bus: a change must be queued inside its transaction")
	}
	return sqlTx, nil
}

// emailOf is the address of userID, empty for 0 or an unknown id.
func emailOf(ctx context.Context, tx *sql.Tx, userID int64) (string, error) {
	if userID == 0 {
		return "", nil
	}
	var email sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("bus: read the actor: %w", err)
	}
	return email.String, nil
}

// idOf is a row id as the decimal string facile_id carries.
func idOf(id int64) string {
	return strconv.FormatInt(id, 10)
}
