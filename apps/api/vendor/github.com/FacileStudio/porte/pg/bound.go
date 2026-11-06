package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FacileStudio/porte"
)

// querier is what a write needs, from a pool or inside a transaction.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// lockUser takes the transaction-scoped advisory lock porte keys on one
// account: exclusive for a purge or a federated Save, shared for a bound
// insert. It names no table, so it works whatever table an adopter keeps its
// accounts in, and every writer takes it before touching a row, so the
// writers cannot deadlock on each other.
func lockUser(ctx context.Context, q querier, userID int64, shared bool) error {
	return lockAdvisory(ctx, q, "porte/user:", userID, shared)
}

// lockAdvisory takes the transaction-scoped advisory lock for namespace and
// id. Each namespace is its own lock space, so serialising one kind of write
// never stalls another.
func lockAdvisory(ctx context.Context, q querier, namespace string, id int64, shared bool) error {
	function := "pg_advisory_xact_lock"
	if shared {
		function += "_shared"
	}
	if _, err := q.ExecContext(ctx,
		`SELECT `+function+`(hashtextextended($1 || $2::bigint::text, 0))`, namespace, id); err != nil {
		return fmt.Errorf("porte/pg: lock %s%d: %w", namespace, id, err)
	}
	return nil
}

// withUserLock runs write in a transaction that first takes the account's
// advisory lock.
func withUserLock(ctx context.Context, db *sql.DB, userID int64, shared bool, write func(querier) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("porte/pg: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := lockUser(ctx, tx, userID, shared); err != nil {
		return err
	}
	if err := write(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("porte/pg: commit: %w", err)
	}
	return nil
}

// whileBound runs insert directly when binding vouches for nothing, and
// otherwise only while what vouched for it still exists: the identity
// (Provider, Subject) of binding.UserID, or the session IssuedBy of that
// user. The check runs under the account's shared advisory lock, which a
// purge's exclusive one excludes, so a purge either sees the inserted row and
// deletes it or commits first and makes the check fail. It answers
// porte.ErrNotFound when the voucher is gone.
func whileBound(ctx context.Context, db *sql.DB, binding porte.Session, insert func(querier) error) error {
	if binding.Provider == "" && binding.IssuedBy == 0 {
		return insert(db)
	}
	return withUserLock(ctx, db, binding.UserID, true, func(q querier) error {
		exists, err := vouched(ctx, q, binding)
		if err != nil {
			return fmt.Errorf("porte/pg: check the binding: %w", err)
		}
		if !exists {
			return porte.ErrNotFound
		}
		return insert(q)
	})
}

// vouched reports whether the identity or the session a binding names still
// belongs to its user.
func vouched(ctx context.Context, q querier, binding porte.Session) (bool, error) {
	var exists bool
	if binding.Provider != "" {
		err := q.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM porte_identities WHERE provider = $1 AND subject = $2 AND user_id = $3)`,
			binding.Provider, binding.Subject, binding.UserID).Scan(&exists)
		return exists, err
	}
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM porte_sessions WHERE id = $1 AND user_id = $2)`,
		binding.IssuedBy, binding.UserID).Scan(&exists)
	return exists, err
}
