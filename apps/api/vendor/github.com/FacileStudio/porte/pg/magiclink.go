package pg

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/FacileStudio/porte"
)

// MagicLinkStore implements [porte.MagicLinkStore] over porte_magic_links.
type MagicLinkStore struct{ db *sql.DB }

// Create stores the link and ends the user's older unconsumed links in one
// transaction.
//
// The transaction takes two advisory locks first, both keyed on the account
// id and naming no table, so it works whatever table an adopter keeps its
// accounts in. The account's shared lock orders the link against a purge, as
// a bound session insert is ordered: a purge either deletes the link or
// commits first. An exclusive lock in its own magic-link namespace serialises
// link creation per account without stalling a sign-in. Without it two
// requests for one account, a double click or a retried form, would each find
// nothing to end, each insert, and leave two live links where the contract
// promises one: under READ COMMITTED neither statement sees the other's
// insert.
func (s *MagicLinkStore) Create(ctx context.Context, link porte.MagicLink) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("porte/pg: create magic link: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, link.UserID, true); err != nil {
		return err
	}
	if err := lockAdvisory(ctx, tx, "porte/magiclink:", link.UserID, false); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE porte_magic_links SET consumed_at = $2
		 WHERE user_id = $1 AND consumed_at IS NULL`, link.UserID, link.CreatedAt); err != nil {
		return fmt.Errorf("porte/pg: supersede magic links: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO porte_magic_links (token_hash, user_id, ip, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		link.TokenHash, link.UserID, link.IP, link.CreatedAt, link.ExpiresAt); err != nil {
		return fmt.Errorf("porte/pg: insert magic link: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("porte/pg: create magic link: %w", err)
	}
	return nil
}

// Consume spends the link in one conditional UPDATE, so two consumes racing
// each other cannot both win. Every failure is porte.ErrNotFound: unlike a login
// code, a link does not distinguish a replay, because the answer the caller
// gets must not say which of the dead states the token is in.
func (s *MagicLinkStore) Consume(ctx context.Context, tokenHash string, now time.Time) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `
		UPDATE porte_magic_links SET consumed_at = $2
		 WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $2
		RETURNING user_id`, tokenHash, now).Scan(&userID)
	if stderrors.Is(err, sql.ErrNoRows) {
		return 0, porte.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("porte/pg: consume magic link: %w", err)
	}
	return userID, nil
}

// CountSince counts the links matching key created at or after since. A zero
// key matches nothing.
//
// Each key shape has its own statement. One query with `$1 = 0 OR user_id = $1`
// reads well and plans badly: with a generic plan neither index is usable, and
// the partial index on ip cannot be proven applicable at all.
func (s *MagicLinkStore) CountSince(ctx context.Context, key porte.LinkKey, since time.Time) (int, error) {
	var (
		query string
		args  []any
	)
	switch {
	case key.UserID != 0 && key.IP != "":
		query = `WHERE user_id = $1 AND ip = $2 AND created_at >= $3`
		args = []any{key.UserID, key.IP, since}
	case key.UserID != 0:
		query = `WHERE user_id = $1 AND created_at >= $2`
		args = []any{key.UserID, since}
	case key.IP != "":
		query = `WHERE ip = $1 AND ip <> '' AND created_at >= $2`
		args = []any{key.IP, since}
	default:
		return 0, nil
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM porte_magic_links `+query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("porte/pg: count magic links: %w", err)
	}
	return count, nil
}

// DeleteExpired sweeps links that expired before cutoff.
func (s *MagicLinkStore) DeleteExpired(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM porte_magic_links WHERE expires_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: delete expired magic links: %w", err)
	}
	return result.RowsAffected()
}
