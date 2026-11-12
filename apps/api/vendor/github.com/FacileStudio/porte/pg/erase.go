package pg

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
)

// EraseIdentity removes what porte holds for one erased identity: the
// identity row with its provider tokens, and every session, named token,
// login code and magic link of the account it belonged to. By design it
// removes only this one identity: the account's other identities, if any,
// and porte_users stay, because the app's own tables may point at them; the
// app removes the account in its OnPersonDeleted callback, which runs first.
//
// An identity that is already gone is not an error, so a retry after a
// failure further along is safe. The account's exclusive advisory lock orders
// it against a bound insert, as a purge is ordered. The lock is keyed on the
// account, so the identity is read once to learn it and again under the lock;
// if it moved to another account in between, the erase starts over.
func (s *Store) EraseIdentity(ctx context.Context, provider, subject string) error {
	for range 3 {
		userID, found, err := identityOwner(ctx, s.db, provider, subject)
		if err != nil || !found {
			return err
		}
		err = withUserLock(ctx, s.db, userID, false, func(q querier) error {
			return eraseLocked(ctx, q, s.tables, userID, provider, subject)
		})
		if !stderrors.Is(err, errMoved) {
			return err
		}
	}
	return fmt.Errorf("porte/pg: identity %s moved between accounts while erasing", subject)
}

// HoldsIdentity reports whether an account holds the (provider, subject)
// identity. It has the signature of porte/directory's Deletions.Holds and
// matches the provider exactly, as EraseIdentity does, so the receiver erases
// a handled person again when a sign-in or a provisioning call re-created
// the account after the erasure.
func (s *Store) HoldsIdentity(ctx context.Context, provider, subject string) (bool, error) {
	_, found, err := identityOwner(ctx, s.db, provider, subject)
	return found, err
}

var errMoved = stderrors.New("identity moved")

func identityOwner(ctx context.Context, q querier, provider, subject string) (int64, bool, error) {
	var userID int64
	err := q.QueryRowContext(ctx,
		`SELECT user_id FROM porte_identities WHERE provider = $1 AND subject = $2`, provider, subject).Scan(&userID)
	if stderrors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("porte/pg: find identity to erase: %w", err)
	}
	return userID, true, nil
}

func eraseLocked(ctx context.Context, q querier, tables *tableSet, userID int64, provider, subject string) error {
	owner, found, err := identityOwner(ctx, q, provider, subject)
	if err != nil || !found {
		return err
	}
	if owner != userID {
		return errMoved
	}
	list, err := tables.userTables(ctx, q, "porte_sessions")
	if err != nil {
		return err
	}
	for _, table := range list {
		if _, err := q.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id = $1", userID); err != nil {
			return fmt.Errorf("porte/pg: erase %s: %w", table, err)
		}
	}
	_, err = q.ExecContext(ctx, `DELETE FROM porte_identities WHERE provider = $1 AND subject = $2`, provider, subject)
	if err != nil {
		return fmt.Errorf("porte/pg: erase identity: %w", err)
	}
	return nil
}
