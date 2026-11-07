package pg

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/FacileStudio/porte"
)

// refusal says why claims that matched no account cannot take its address.
func refusal(claims porte.Claims) error {
	if !claims.EmailVerified {
		return fmt.Errorf(
			"porte/pg: %s is already registered under a different identity, and the provider did not verify the address, so porte will not link them",
			claims.Email)
	}
	return fmt.Errorf(
		"porte/pg: %s already belongs to another identity of this provider, so porte will not link a second one",
		claims.Email)
}

// byVerifiedEmail finds the row holding the address, takes that account's
// exclusive advisory lock, then re-reads the row and its identities in fresh
// statements, so it sees whatever a concurrent purge or link committed while
// it waited. A row whose address was never verified matches even when it
// holds an identity of this provider: its holder never proved the address,
// and resolveOrCreate purges it.
func byVerifiedEmail(ctx context.Context, tx *sql.Tx, claims porte.Claims) (match, bool, error) {
	found := match{byEmail: true}
	err := tx.QueryRowContext(ctx, `SELECT id FROM porte_users WHERE email = $1`, claims.Email).Scan(&found.userID)
	if stderrors.Is(err, sql.ErrNoRows) {
		return found, false, nil
	}
	if err == nil {
		err = lockUser(ctx, tx, found.userID, false)
	}
	var verified, linked bool
	if err == nil {
		err = tx.QueryRowContext(ctx, `
			SELECT u.email_verified,
			       EXISTS (SELECT 1 FROM porte_identities i WHERE i.user_id = u.id AND i.provider = $3)
			  FROM porte_users u WHERE u.id = $1 AND u.email = $2`,
			found.userID, claims.Email, claims.Provider).Scan(&verified, &linked)
	}
	if stderrors.Is(err, sql.ErrNoRows) {
		return match{}, false, nil
	}
	if err != nil {
		return found, false, fmt.Errorf("porte/pg: resolve user by email: %w", err)
	}
	found.unverified = !verified
	return found, found.unverified || !linked, nil
}

// purge removes everything a holder of an unverified row could still use.
func purge(ctx context.Context, tx *sql.Tx, tables *tableSet, userID int64) error {
	list, err := tables.userTables(ctx, tx, "porte_identities", "porte_sessions")
	if err != nil {
		return err
	}
	for _, table := range list {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id = $1", userID); err != nil {
			return fmt.Errorf("porte/pg: purge %s: %w", table, err)
		}
	}
	return nil
}
