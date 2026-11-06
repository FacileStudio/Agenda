package pg

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/FacileStudio/porte"
)

// UpsertFromOIDC resolves the callback's claims to a user id, creating the
// user and the identity link when they are new.
//
// Matching is on (provider, subject) first. The email fallback runs only when
// the provider verified the address. Onto a verified row it links only when
// the row holds no identity from this provider yet: an account already linked
// to another subject of the same issuer is somebody else's, so a recycled
// address is refused rather than linked. The fallback exists at all so
// accounts created before the subject was recorded, or by a local
// registration, still link on their next login.
//
// A row whose address was never verified is taken over by the verified login,
// whatever identities it holds, and purged first in the same transaction:
// every identity on it (a password included), its sessions and API tokens,
// its login codes and its magic links. Whoever created that row never proved
// the address, so a squat becomes neither a takeover nor a lockout.
//
// The email is trimmed and lowercased, as porte/local does, and an empty one
// is refused. An unverified claim never overwrites the stored address, and
// never clears email_verified once a verified claim has set it.
//
// Concurrent first logins are settled by the INSERT's ON CONFLICT on the
// email: the loser waits for the winner's commit, inserts nothing, and then
// resolves again, where it finds the winner's identity by subject. The email
// fallback locks the user row before it checks for an identity, so two
// subjects cannot both link into one row.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	claims.Email = strings.ToLower(strings.TrimSpace(claims.Email))
	if claims.Email == "" {
		return 0, fmt.Errorf("porte/pg: the provider asserted no email for subject %q", claims.Subject)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := resolveOrCreate(ctx, tx, claims)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO porte_identities (user_id, provider, subject)
		VALUES ($1, $2, $3)
		ON CONFLICT (provider, subject) DO UPDATE SET user_id = EXCLUDED.user_id`,
		userID, claims.Provider, claims.Subject,
	); err != nil {
		return 0, fmt.Errorf("porte/pg: link identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("porte/pg: commit: %w", err)
	}
	return userID, nil
}

// match is the user an existing row resolved to, and how. A zero userID
// means nothing matched.
type match struct {
	userID     int64
	byEmail    bool
	unverified bool
	inserted   bool
}

// resolveOrCreate finds the user the claims belong to and refreshes its
// profile, or creates it.
func resolveOrCreate(ctx context.Context, tx *sql.Tx, claims porte.Claims) (int64, error) {
	found, err := findOrInsert(ctx, tx, claims)
	switch {
	case err != nil:
		return 0, err
	case found.inserted:
		return found.userID, nil
	case found.userID == 0:
		return 0, refusal(claims)
	case found.byEmail && found.unverified:
		if err := purge(ctx, tx, found.userID); err != nil {
			return 0, err
		}
	}
	return found.userID, updateUser(ctx, tx, found.userID, claims)
}

// findOrInsert resolves the claims to an existing row, or inserts one. An
// insert that loses a race resolves again, now that the winner committed.
func findOrInsert(ctx context.Context, tx *sql.Tx, claims porte.Claims) (match, error) {
	found, ok, err := existingUser(ctx, tx, claims)
	if err != nil || ok {
		return found, err
	}
	inserted, err := insertUser(ctx, tx, claims)
	if err != nil || inserted != 0 {
		return match{userID: inserted, inserted: true}, err
	}
	found, ok, err = existingUser(ctx, tx, claims)
	if !ok {
		return match{}, err
	}
	return found, err
}

// existingUser looks the claims up by (provider, subject), then by a verified
// email onto a row that holds no identity from this provider.
func existingUser(ctx context.Context, tx *sql.Tx, claims porte.Claims) (match, bool, error) {
	var found match
	err := tx.QueryRowContext(ctx,
		`SELECT user_id FROM porte_identities WHERE provider = $1 AND subject = $2`,
		claims.Provider, claims.Subject).Scan(&found.userID)
	switch {
	case err == nil:
		return found, true, nil
	case !stderrors.Is(err, sql.ErrNoRows):
		return found, false, fmt.Errorf("porte/pg: resolve user: %w", err)
	case !claims.EmailVerified:
		return found, false, nil
	}
	return byVerifiedEmail(ctx, tx, claims)
}

// insertUser creates the user row and returns its id, or 0 when the email is
// already taken, after waiting for any concurrent transaction that holds it.
func insertUser(ctx context.Context, tx *sql.Tx, claims porte.Claims) (int64, error) {
	avatarSource := ""
	if claims.AvatarURL != "" {
		avatarSource = "oidc"
	}
	var userID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO porte_users (email, email_verified, name, avatar_url, avatar_source, locale)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (email) DO NOTHING
		RETURNING id`,
		claims.Email, claims.EmailVerified, claims.DisplayName(), claims.AvatarURL, avatarSource,
		porte.NormalizeLocale(claims.Locale),
	).Scan(&userID)
	switch {
	case err == nil:
		return userID, nil
	case stderrors.Is(err, sql.ErrNoRows):
		return 0, nil
	}
	return 0, fmt.Errorf("porte/pg: insert user: %w", err)
}

// updateUser writes the claims' profile onto an existing user.
func updateUser(ctx context.Context, tx *sql.Tx, userID int64, claims porte.Claims) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE porte_users
		   SET email = CASE WHEN $3 THEN $2 ELSE email END,
		       email_verified = email_verified OR $3,
		       name = COALESCE(NULLIF($4, ''), name),
		       avatar_url = COALESCE(NULLIF($5, ''), avatar_url),
		       avatar_source = CASE WHEN $5 <> '' THEN 'oidc' ELSE avatar_source END,
		       locale = COALESCE(NULLIF($6, ''), locale),
		       updated_at = now()
		 WHERE id = $1`,
		userID, claims.Email, claims.EmailVerified, claims.DisplayName(), claims.AvatarURL, keptLocale(claims.Locale),
	); err != nil {
		return fmt.Errorf("porte/pg: update user: %w", err)
	}
	return nil
}
