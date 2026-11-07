// Package pg is porte's default storage: the identity tables and the four
// stores over them.
//
// It takes a *sql.DB, so it works with whatever driver and pool the app
// already has — GORM hands one over with db.DB(). It imports database/sql and
// nothing else, so adopting it costs no dependency and forces no ORM.
//
// An app with an exotic user model implements [porte.UserStore] over its own
// tables and ignores this package. That is the escape hatch, and it exists
// because the alternative — porte defining what a user is — is how an auth
// library turns into a framework.
package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/FacileStudio/porte"
)

// Schema creates the identity tables. Apply it through the app's own
// migrations — tronc/migrate, goose, whatever is already there — rather than
// at boot: a schema applied on startup races every other replica.
//
// The app keeps its business columns in its own table, keyed on
// porte_users(id), so the int64 foreign keys already in place keep pointing at
// the same thing.
//
// The one UPDATE in here is v0.3.0's re-keying of password identities off the
// email address and onto the account id — see [porte.LocalSubject] for why the
// address was the wrong key. It is idempotent, since after it runs the
// predicate is false for every row, and it is deliberately allowed to fail:
// the only way it can is a user holding two password identities at once, which
// the old key made reachable and which nothing should paper over by picking
// one. Refusing to migrate is the right answer to ambiguous credentials.
//
// It also makes the constraint free. subject is half the primary key, so once
// it holds the account id, "one password per account" is enforced by the table
// rather than by a check somebody has to remember to write.
//
// Columns added after a table first shipped are guarded by information_schema
// checks rather than ALTER TABLE ... IF NOT EXISTS, which takes an ACCESS
// EXCLUSIVE lock even when there is nothing to add: a boot behind a long read
// would otherwise queue, and every request behind the boot with it.
const Schema = `
CREATE TABLE IF NOT EXISTS porte_users (
	id             bigserial PRIMARY KEY,
	facile_id      text UNIQUE,
	email          text NOT NULL UNIQUE,
	email_verified boolean NOT NULL DEFAULT false,
	name           text NOT NULL DEFAULT '',
	avatar_url     text NOT NULL DEFAULT '',
	avatar_source  text NOT NULL DEFAULT '',
	created_at     timestamptz NOT NULL DEFAULT now(),
	updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS porte_identities (
	user_id         bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	provider        text NOT NULL,
	subject         text NOT NULL,
	password_hash   text NOT NULL DEFAULT '',
	access_token    text NOT NULL DEFAULT '',
	refresh_token   text NOT NULL DEFAULT '',
	token_expiry    timestamptz,
	roles           jsonb,
	roles_synced_at timestamptz,
	synced_at       timestamptz,
	created_at      timestamptz DEFAULT now(),
	PRIMARY KEY (provider, subject)
);
CREATE INDEX IF NOT EXISTS porte_identities_user_idx ON porte_identities (user_id);

UPDATE porte_identities SET subject = user_id::text
 WHERE provider = 'local' AND subject <> user_id::text;

CREATE TABLE IF NOT EXISTS porte_sessions (
	id           bigserial PRIMARY KEY,
	token_hash   text NOT NULL UNIQUE,
	user_id      bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	label        text NOT NULL DEFAULT '',
	created_at   timestamptz NOT NULL DEFAULT now(),
	last_used_at timestamptz NOT NULL DEFAULT now(),
	expires_at   timestamptz
);
CREATE INDEX IF NOT EXISTS porte_sessions_user_idx ON porte_sessions (user_id);
CREATE INDEX IF NOT EXISTS porte_sessions_expiry_idx ON porte_sessions (expires_at)
	WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS porte_login_codes (
	code_hash   text PRIMARY KEY,
	user_id     bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	expires_at  timestamptz NOT NULL,
	consumed_at timestamptz
);

CREATE TABLE IF NOT EXISTS porte_magic_links (
	token_hash  text PRIMARY KEY,
	user_id     bigint NOT NULL REFERENCES porte_users(id) ON DELETE CASCADE,
	ip          text NOT NULL DEFAULT '',
	created_at  timestamptz NOT NULL,
	expires_at  timestamptz NOT NULL,
	consumed_at timestamptz
);
CREATE INDEX IF NOT EXISTS porte_magic_links_user_idx ON porte_magic_links (user_id, created_at);
CREATE INDEX IF NOT EXISTS porte_magic_links_ip_idx ON porte_magic_links (ip, created_at)
	WHERE ip <> '';
CREATE INDEX IF NOT EXISTS porte_magic_links_expiry_idx ON porte_magic_links (expires_at);

DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM information_schema.columns
	                WHERE table_schema = current_schema() AND table_name = 'porte_users' AND column_name = 'locale') THEN
		ALTER TABLE porte_users ADD COLUMN locale text NOT NULL DEFAULT 'en';
	END IF;
	IF NOT EXISTS (SELECT 1 FROM information_schema.columns
	                WHERE table_schema = current_schema() AND table_name = 'porte_identities' AND column_name = 'created_at') THEN
		ALTER TABLE porte_identities ADD COLUMN created_at timestamptz;
		ALTER TABLE porte_identities ALTER COLUMN created_at SET DEFAULT now();
	ELSIF EXISTS (SELECT 1 FROM information_schema.columns
	               WHERE table_schema = current_schema() AND table_name = 'porte_identities'
	                 AND column_name = 'created_at' AND column_default IS NULL) THEN
		ALTER TABLE porte_identities ALTER COLUMN created_at SET DEFAULT now();
	END IF;
	IF NOT EXISTS (SELECT 1 FROM information_schema.columns
	                WHERE table_schema = current_schema() AND table_name = 'porte_login_codes' AND column_name = 'consumed_at') THEN
		ALTER TABLE porte_login_codes ADD COLUMN consumed_at timestamptz;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM information_schema.columns
	                WHERE table_schema = current_schema() AND table_name = 'porte_login_codes' AND column_name = 'provider') THEN
		ALTER TABLE porte_login_codes ADD COLUMN provider text NOT NULL DEFAULT '',
		                              ADD COLUMN subject text NOT NULL DEFAULT '';
	END IF;
END
$$;

` + KeySchema

// Store is the entry point: one value over a *sql.DB that hands out the four
// stores porte asks for.
//
// They are four types rather than one because two of the interfaces spell the
// same method differently — SessionStore.Find takes a token hash and
// IdentityStore.Find takes a provider and a subject — and Go has no way to
// carry both on one receiver. Renaming a method to dodge that would leak a
// storage accident into the contract.
type Store struct {
	db     *sql.DB
	tables *tableSet
}

// New returns a store over db.
func New(db *sql.DB) *Store { return &Store{db: db, tables: newTableSet()} }

// Users resolves OIDC callbacks to user ids.
func (s *Store) Users() *UserStore { return &UserStore{db: s.db, tables: s.tables} }

// Identities persists the authentication rows and the cached claim.
func (s *Store) Identities() *IdentityStore { return &IdentityStore{db: s.db} }

// Sessions persists issued credentials.
func (s *Store) Sessions() *SessionStore { return &SessionStore{db: s.db} }

// LoginCodes persists pending CLI login codes.
func (s *Store) LoginCodes() *LoginCodeStore { return &LoginCodeStore{db: s.db} }

// MagicLinks persists the links porte/magiclink emails.
func (s *Store) MagicLinks() *MagicLinkStore { return &MagicLinkStore{db: s.db} }

// UserStore implements [porte.UserStore] over porte_users.
type UserStore struct {
	db     *sql.DB
	tables *tableSet
}

// IdentityStore implements [porte.IdentityStore] over porte_identities.
type IdentityStore struct{ db *sql.DB }

// SessionStore implements [porte.SessionStore] over porte_sessions.
type SessionStore struct{ db *sql.DB }

// LoginCodeStore implements [porte.LoginCodeStore] over porte_login_codes.
type LoginCodeStore struct{ db *sql.DB }

var (
	_ porte.UserStore         = (*UserStore)(nil)
	_ porte.PasswordUserStore = (*UserStore)(nil)
	_ porte.IdentityStore     = (*IdentityStore)(nil)
	_ porte.SessionStore      = (*SessionStore)(nil)
	_ porte.LoginCodeStore    = (*LoginCodeStore)(nil)
	_ porte.MagicLinkStore    = (*MagicLinkStore)(nil)
)

// EnsureSchema applies [Schema]. It is here for tests and local development;
// production schema changes belong in the app's migrations.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, Schema); err != nil {
		return fmt.Errorf("porte/pg: ensure schema: %w", err)
	}
	return nil
}

// CreateFromPassword creates a user row for a local registration.
//
// It does not take a lock. The first-account-is-an-administrator rule and the
// registration gate are the app's, so the app holds the lock that makes
// counting and inserting atomic; the unique index on email is what stops a
// duplicate here regardless.
func (s *UserStore) CreateFromPassword(ctx context.Context, email, name string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO porte_users (email, email_verified, name)
		VALUES ($1, false, $2)
		RETURNING id`, email, name).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: create user from password: %w", err)
	}
	return userID, nil
}

// FindByEmail returns the user id for an address, or porte.ErrNotFound.
func (s *UserStore) FindByEmail(ctx context.Context, email string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM porte_users WHERE email = $1`, email).Scan(&userID)
	if stderrors.Is(err, sql.ErrNoRows) {
		return 0, porte.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("porte/pg: find user by email: %w", err)
	}
	return userID, nil
}

// Find returns the identity for a provider and subject.
func (s *IdentityStore) Find(ctx context.Context, provider, subject string) (porte.StoredIdentity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT user_id, provider, subject, password_hash, access_token, refresh_token,
		       token_expiry, roles, roles_synced_at, synced_at
		  FROM porte_identities
		 WHERE provider = $1 AND subject = $2`, provider, subject)
	identity, err := scanIdentity(row)
	if stderrors.Is(err, sql.ErrNoRows) {
		return porte.StoredIdentity{}, porte.ErrNotFound
	}
	if err != nil {
		return porte.StoredIdentity{}, fmt.Errorf("porte/pg: find identity: %w", err)
	}
	return identity, nil
}

// Save inserts or updates a password identity by (provider, subject). A
// federated identity it inserts when absent and updates when present, but
// never re-points onto another account, and never creates one on an account
// that already holds a different subject of the same provider.
//
// That second rule is what keeps a late Save after a purge from resurrecting
// the purged identity: the purge leaves the owner's identity of the same
// provider on the row. Save takes the account's advisory lock first, as the
// purge does, so it either runs before the purge and is purged with the rest,
// or after it and sees the owner's identity. The lock names no table, so Save
// works whatever table an adopter keeps its accounts in. Either refusal is porte.ErrNotFound.
func (s *IdentityStore) Save(ctx context.Context, identity porte.StoredIdentity) error {
	roles, err := marshalRoles(identity.Roles)
	if err != nil {
		return err
	}
	if identity.Provider != porte.ProviderLocal {
		return s.saveFederated(ctx, identity, roles)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO porte_identities
		       (user_id, provider, subject, password_hash, access_token, refresh_token,
		        token_expiry, roles, roles_synced_at, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (provider, subject) DO UPDATE SET
		       user_id         = EXCLUDED.user_id,
		       password_hash   = EXCLUDED.password_hash,
		       access_token    = EXCLUDED.access_token,
		       refresh_token   = EXCLUDED.refresh_token,
		       token_expiry    = EXCLUDED.token_expiry,
		       roles           = EXCLUDED.roles,
		       roles_synced_at = EXCLUDED.roles_synced_at,
		       synced_at       = EXCLUDED.synced_at`,
		identity.UserID, identity.Provider, identity.Subject, identity.PasswordHash,
		identity.Tokens.AccessToken, identity.Tokens.RefreshToken,
		nullTime(identity.Tokens.Expiry), roles,
		nullTime(identity.RolesSyncedAt), nullTime(identity.SyncedAt))
	if err != nil {
		return fmt.Errorf("porte/pg: save identity: %w", err)
	}
	return nil
}

// saveFederated is Save for a non-local identity, under the account's
// exclusive advisory lock, the one the purge takes.
func (s *IdentityStore) saveFederated(ctx context.Context, identity porte.StoredIdentity, roles []byte) error {
	return withUserLock(ctx, s.db, identity.UserID, false, func(q querier) error {
		result, err := q.ExecContext(ctx, `
			INSERT INTO porte_identities
			       (user_id, provider, subject, password_hash, access_token, refresh_token,
			        token_expiry, roles, roles_synced_at, synced_at)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
			 WHERE EXISTS (SELECT 1 FROM porte_identities WHERE provider = $2 AND subject = $3 AND user_id = $1)
			    OR NOT EXISTS (SELECT 1 FROM porte_identities WHERE provider = $2 AND user_id = $1)
			ON CONFLICT (provider, subject) DO UPDATE SET
			       password_hash = EXCLUDED.password_hash, access_token = EXCLUDED.access_token,
			       refresh_token = EXCLUDED.refresh_token, token_expiry = EXCLUDED.token_expiry,
			       roles = EXCLUDED.roles, roles_synced_at = EXCLUDED.roles_synced_at,
			       synced_at = EXCLUDED.synced_at
			 WHERE porte_identities.user_id = EXCLUDED.user_id`,
			identity.UserID, identity.Provider, identity.Subject, identity.PasswordHash,
			identity.Tokens.AccessToken, identity.Tokens.RefreshToken,
			nullTime(identity.Tokens.Expiry), roles,
			nullTime(identity.RolesSyncedAt), nullTime(identity.SyncedAt))
		if err != nil {
			return fmt.Errorf("porte/pg: save identity: %w", err)
		}
		if written, err := result.RowsAffected(); err != nil || written == 0 {
			return porte.ErrNotFound
		}
		return nil
	})
}

// MarkRolesSynced moves the roles_synced_at stamp and nothing else.
//
// One column, one statement: this runs on the path where a role refresh just
// failed, and the identity the caller is holding was read before that attempt.
// Writing it back through Save would restore whatever refresh token it had read
// over the one a concurrent request may have just rotated in, and a lost
// rotation means every later refresh for that user fails.
func (s *IdentityStore) MarkRolesSynced(ctx context.Context, provider, subject string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE porte_identities SET roles_synced_at = $3
		 WHERE provider = $1 AND subject = $2`, provider, subject, at)
	if err != nil {
		return fmt.Errorf("porte/pg: mark roles synced: %w", err)
	}
	return requireOne(result, "porte/pg: mark roles synced")
}

// ListByUser returns every way this human can authenticate.
func (s *IdentityStore) ListByUser(ctx context.Context, userID int64) ([]porte.StoredIdentity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, provider, subject, password_hash, access_token, refresh_token,
		       token_expiry, roles, roles_synced_at, synced_at
		  FROM porte_identities
		 WHERE user_id = $1
		 ORDER BY provider`, userID)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: list identities: %w", err)
	}
	defer func() { _ = rows.Close() }()

	identities := []porte.StoredIdentity{}
	for rows.Next() {
		identity, err := scanIdentity(rows)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: scan identity: %w", err)
		}
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("porte/pg: list identities: %w", err)
	}
	return identities, nil
}

// Create stores a session and returns it with its id filled in.
//
// A bound session is inserted only while what vouched for it still exists:
// its identity (Provider set) or the session that issued it (IssuedBy set).
// The check runs under the account's shared advisory lock, which a purge's
// exclusive lock excludes. The binding is checked, not stored, so the insert
// uses the column list every adopter's own DDL already has.
func (s *SessionStore) Create(ctx context.Context, session porte.Session) (porte.Session, error) {
	insert := func(q querier) error {
		return q.QueryRowContext(ctx, `
			INSERT INTO porte_sessions (token_hash, user_id, label, created_at, last_used_at, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,
			session.TokenHash, session.UserID, session.Label,
			session.CreatedAt, session.LastUsedAt, nullTime(session.ExpiresAt),
		).Scan(&session.ID)
	}
	err := whileBound(ctx, s.db, session, insert)
	if stderrors.Is(err, porte.ErrNotFound) {
		return porte.Session{}, err
	}
	if err != nil {
		return porte.Session{}, fmt.Errorf("porte/pg: create session: %w", err)
	}
	return session, nil
}

// Find returns the session for a token hash. It deliberately does not filter
// on expiry: an expired session and a missing one are different answers, and
// only the caller knows which error to return.
func (s *SessionStore) Find(ctx context.Context, tokenHash string) (porte.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, token_hash, user_id, label, created_at, last_used_at, expires_at
		  FROM porte_sessions WHERE token_hash = $1`, tokenHash)
	session, err := scanSession(row)
	if stderrors.Is(err, sql.ErrNoRows) {
		return porte.Session{}, porte.ErrNotFound
	}
	if err != nil {
		return porte.Session{}, fmt.Errorf("porte/pg: find session: %w", err)
	}
	return session, nil
}

// Touch records use, and skips the write when the column is already recent
// enough. The caller coalesces too; doing it here as well means a second
// caller cannot turn the column into a per-request UPDATE.
func (s *SessionStore) Touch(ctx context.Context, tokenHash string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE porte_sessions SET last_used_at = $2
		 WHERE token_hash = $1
		   AND last_used_at < $2::timestamptz - interval '1 minute'`, tokenHash, at)
	if err != nil {
		return fmt.Errorf("porte/pg: touch session: %w", err)
	}
	return nil
}

// Delete revokes one session by its token hash.
func (s *SessionStore) Delete(ctx context.Context, tokenHash string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM porte_sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("porte/pg: delete session: %w", err)
	}
	return requireOne(result, "porte/pg: delete session")
}

// DeleteByUser drops every session a user holds. This is what back-channel
// logout calls.
func (s *SessionStore) DeleteByUser(ctx context.Context, userID int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM porte_sessions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: delete sessions: %w", err)
	}
	return result.RowsAffected()
}

// DeleteLogins drops a user's logins and spares their named API tokens.
//
// The label is the whole discriminator, and it is the same one the sweeper
// uses: an unlabelled row was minted by signing in, a labelled row was created
// on purpose from an already authenticated session. except spares one id, so
// the session performing a password change can be rotated rather than dropped
// out from under the request making it.
func (s *SessionStore) DeleteLogins(ctx context.Context, userID, except int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM porte_sessions
		 WHERE user_id = $1 AND label = '' AND ($2 = 0 OR id <> $2)`, userID, except)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: delete logins: %w", err)
	}
	return result.RowsAffected()
}

// ListByUser backs an active-sessions screen.
func (s *SessionStore) ListByUser(ctx context.Context, userID int64) ([]porte.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, token_hash, user_id, label, created_at, last_used_at, expires_at
		  FROM porte_sessions WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	sessions := []porte.Session{}
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: scan session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("porte/pg: list sessions: %w", err)
	}
	return sessions, nil
}

// DeleteByID revokes one session. The user id is in the WHERE clause, not
// checked afterwards, so a handler cannot revoke someone else's session by
// guessing an integer.
func (s *SessionStore) DeleteByID(ctx context.Context, userID, sessionID int64) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM porte_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("porte/pg: delete session: %w", err)
	}
	return requireOne(result, "porte/pg: delete session")
}

// DeleteExpired is the sweeper. Rows with no expiry are API tokens and are
// never swept.
func (s *SessionStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM porte_sessions WHERE expires_at IS NOT NULL AND expires_at < $1`, now)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: delete expired sessions: %w", err)
	}
	return result.RowsAffected()
}

// Create stores a pending code. An unbound code uses the column list every
// adopter's own DDL already has; a bound one also stores provider and
// subject, which a table without those columns refuses, and is inserted only
// while its identity exists, like a bound session.
func (s *LoginCodeStore) Create(ctx context.Context, code porte.LoginCode) error {
	insert := func(q querier) error {
		if code.Provider == "" {
			_, err := q.ExecContext(ctx,
				`INSERT INTO porte_login_codes (code_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
				code.CodeHash, code.UserID, code.ExpiresAt)
			return err
		}
		_, err := q.ExecContext(ctx, `
			INSERT INTO porte_login_codes (code_hash, user_id, expires_at, provider, subject)
			VALUES ($1, $2, $3, $4, $5)`,
			code.CodeHash, code.UserID, code.ExpiresAt, code.Provider, code.Subject)
		return err
	}
	err := whileBound(ctx, s.db, porte.Session{UserID: code.UserID, Provider: code.Provider, Subject: code.Subject}, insert)
	if err != nil && !stderrors.Is(err, porte.ErrNotFound) {
		return fmt.Errorf("porte/pg: create login code: %w", err)
	}
	return err
}

// Consume claims the code in one statement, so two exchanges racing each other
// cannot both win.
//
// The row is stamped rather than deleted, and the stamp is what tells a replay
// from a typo: a code that was valid a moment ago and is being presented a
// second time means either the CLI retried or somebody else is holding it, and
// that is worth a distinct error and a log line. Nothing usable survives — what
// is kept is the SHA-256 of a credential that is already spent, and the sweeper
// removes it on the same schedule as an unused one.
func (s *LoginCodeStore) Consume(ctx context.Context, codeHash string) (porte.LoginCode, error) {
	var code porte.LoginCode
	err := s.db.QueryRowContext(ctx, `
		UPDATE porte_login_codes SET consumed_at = $2
		 WHERE code_hash = $1 AND consumed_at IS NULL
		RETURNING code_hash, user_id, expires_at,
		          COALESCE(to_jsonb(porte_login_codes) ->> 'provider', ''),
		          COALESCE(to_jsonb(porte_login_codes) ->> 'subject', '')`, codeHash, time.Now(),
	).Scan(&code.CodeHash, &code.UserID, &code.ExpiresAt, &code.Provider, &code.Subject)
	if stderrors.Is(err, sql.ErrNoRows) {
		return porte.LoginCode{}, s.whyItFailed(ctx, codeHash)
	}
	if err != nil {
		return porte.LoginCode{}, fmt.Errorf("porte/pg: consume login code: %w", err)
	}
	return code, nil
}

// whyItFailed distinguishes a code that was already spent from one that never
// existed. Both refuse the exchange; only the first is worth alarming about.
func (s *LoginCodeStore) whyItFailed(ctx context.Context, codeHash string) error {
	var consumed sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT consumed_at FROM porte_login_codes WHERE code_hash = $1`, codeHash).Scan(&consumed)
	switch {
	case stderrors.Is(err, sql.ErrNoRows):
		return porte.ErrNotFound
	case err != nil:
		return fmt.Errorf("porte/pg: consume login code: %w", err)
	case consumed.Valid:
		return porte.ErrCodeConsumed
	default:
		return porte.ErrNotFound
	}
}

// DeleteExpired sweeps codes nobody exchanged and the spent rows kept to
// recognise a replay. Both are gone within LoginCodeTTL of being issued.
func (s *LoginCodeStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM porte_login_codes WHERE expires_at < $1`, now)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: delete expired login codes: %w", err)
	}
	return result.RowsAffected()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIdentity(row scanner) (porte.StoredIdentity, error) {
	var (
		identity      porte.StoredIdentity
		expiry        sql.NullTime
		rolesSyncedAt sql.NullTime
		syncedAt      sql.NullTime
		roles         []byte
	)
	if err := row.Scan(&identity.UserID, &identity.Provider, &identity.Subject,
		&identity.PasswordHash, &identity.Tokens.AccessToken, &identity.Tokens.RefreshToken,
		&expiry, &roles, &rolesSyncedAt, &syncedAt); err != nil {
		return porte.StoredIdentity{}, err
	}
	identity.Tokens.Expiry = expiry.Time
	identity.RolesSyncedAt = rolesSyncedAt.Time
	identity.SyncedAt = syncedAt.Time
	if len(roles) > 0 {
		if err := json.Unmarshal(roles, &identity.Roles); err != nil {
			return porte.StoredIdentity{}, fmt.Errorf("porte/pg: unreadable roles: %w", err)
		}
	}
	return identity, nil
}

func scanSession(row scanner) (porte.Session, error) {
	var (
		session   porte.Session
		expiresAt sql.NullTime
	)
	if err := row.Scan(&session.ID, &session.TokenHash, &session.UserID, &session.Label,
		&session.CreatedAt, &session.LastUsedAt, &expiresAt); err != nil {
		return porte.Session{}, err
	}
	session.ExpiresAt = expiresAt.Time
	return session, nil
}

func marshalRoles(roles []string) ([]byte, error) {
	if roles == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(roles)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: encode roles: %w", err)
	}
	return encoded, nil
}

// nullTime maps porte's zero time — which means "never" on an expiry and "not
// yet" on a stamp — onto SQL NULL, so the two are not confused with the year 1.
func nullTime(value time.Time) sql.NullTime {
	return sql.NullTime{Time: value, Valid: !value.IsZero()}
}

func requireOne(result sql.Result, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if affected == 0 {
		return porte.ErrNotFound
	}
	return nil
}

// keptLocale is the locale an update writes: the claim normalised, or empty
// to keep the stored one when the caller built Claims without a locale.
func keptLocale(claimed string) string {
	if claimed == "" {
		return ""
	}
	return porte.NormalizeLocale(claimed)
}
