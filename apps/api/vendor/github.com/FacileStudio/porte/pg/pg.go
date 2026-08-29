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
ALTER TABLE porte_identities ADD COLUMN IF NOT EXISTS created_at timestamptz;
ALTER TABLE porte_identities ALTER COLUMN created_at SET DEFAULT now();

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
ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS consumed_at timestamptz;
`

// Store is the entry point: one value over a *sql.DB that hands out the four
// stores porte asks for.
//
// They are four types rather than one because two of the interfaces spell the
// same method differently — SessionStore.Find takes a token hash and
// IdentityStore.Find takes a provider and a subject — and Go has no way to
// carry both on one receiver. Renaming a method to dodge that would leak a
// storage accident into the contract.
type Store struct {
	db *sql.DB
}

// New returns a store over db.
func New(db *sql.DB) *Store { return &Store{db: db} }

// Users resolves OIDC callbacks to user ids.
func (s *Store) Users() *UserStore { return &UserStore{db: s.db} }

// Identities persists the authentication rows and the cached claim.
func (s *Store) Identities() *IdentityStore { return &IdentityStore{db: s.db} }

// Sessions persists issued credentials.
func (s *Store) Sessions() *SessionStore { return &SessionStore{db: s.db} }

// LoginCodes persists pending CLI login codes.
func (s *Store) LoginCodes() *LoginCodeStore { return &LoginCodeStore{db: s.db} }

// UserStore implements [porte.UserStore] over porte_users.
type UserStore struct{ db *sql.DB }

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

// UpsertFromOIDC resolves the callback's claims to a user id, creating the
// user and the identity link when they are new.
//
// Matching is on (provider, subject) first. The email fallback runs only when
// the provider verified the address: matching a mutable, unproven email is how
// an IdP that lets a user set any address becomes an account takeover
// primitive. It exists at all so accounts created before the subject was
// recorded still link on their next login instead of forking in two.
//
// The whole thing is one transaction. A login that created a user but failed
// to link the identity would create a second user on the next attempt.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	err = tx.QueryRowContext(ctx,
		`SELECT user_id FROM porte_identities WHERE provider = $1 AND subject = $2`,
		claims.Provider, claims.Subject).Scan(&userID)

	if stderrors.Is(err, sql.ErrNoRows) && claims.EmailVerified && claims.Email != "" {
		err = tx.QueryRowContext(ctx,
			`SELECT id FROM porte_users WHERE email = $1`, claims.Email).Scan(&userID)
	}
	switch {
	case err == nil:
		if _, err := tx.ExecContext(ctx, `
			UPDATE porte_users
			   SET email = $2,
			       email_verified = $3,
			       name = COALESCE(NULLIF($4, ''), name),
			       avatar_url = COALESCE(NULLIF($5, ''), avatar_url),
			       avatar_source = CASE WHEN $5 <> '' THEN 'oidc' ELSE avatar_source END,
			       updated_at = now()
			 WHERE id = $1`,
			userID, claims.Email, claims.EmailVerified, claims.DisplayName(), claims.AvatarURL,
		); err != nil {
			return 0, fmt.Errorf("porte/pg: update user: %w", err)
		}
	case stderrors.Is(err, sql.ErrNoRows):
		// A new subject whose email already belongs to somebody. Either
		// the provider re-created the account — in which case linking
		// on an unverified address is the account takeover this refuses
		// — or two humans share an address, which they cannot. Saying
		// so beats a raw unique-violation 500 on the login path.
		var taken int64
		switch err := tx.QueryRowContext(ctx,
			`SELECT id FROM porte_users WHERE email = $1`, claims.Email).Scan(&taken); {
		case err == nil:
			return 0, fmt.Errorf(
				"porte/pg: %s is already registered under a different identity, and the provider did not verify the address, so porte will not link them",
				claims.Email)
		case !stderrors.Is(err, sql.ErrNoRows):
			return 0, fmt.Errorf("porte/pg: check email: %w", err)
		}

		avatarSource := ""
		if claims.AvatarURL != "" {
			avatarSource = "oidc"
		}
		// ON CONFLICT rather than a bare INSERT, because the check above
		// is a read in a READ COMMITTED transaction and two first logins
		// for the same new user — a double click, two tabs, a retried
		// callback — both pass it and both arrive here. Letting the
		// loser adopt the winner's row turns a unique-violation 500 on
		// the login path into a second successful login.
		err := tx.QueryRowContext(ctx, `
			INSERT INTO porte_users (email, email_verified, name, avatar_url, avatar_source)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (email) DO NOTHING
			RETURNING id`,
			claims.Email, claims.EmailVerified, claims.DisplayName(), claims.AvatarURL, avatarSource,
		).Scan(&userID)
		if stderrors.Is(err, sql.ErrNoRows) {
			// Somebody else inserted this email between the check above
			// and here. Adopting their row is right when the provider
			// verified the address — that is the same rule the fallback
			// at the top of this method applies — and is the account
			// takeover it refuses when the provider did not, so the
			// guard is applied again rather than assumed.
			if !claims.EmailVerified {
				return 0, fmt.Errorf(
					"porte/pg: %s is already registered under a different identity, and the provider did not verify the address, so porte will not link them",
					claims.Email)
			}
			err = tx.QueryRowContext(ctx,
				`SELECT id FROM porte_users WHERE email = $1`, claims.Email).Scan(&userID)
		}
		if err != nil {
			return 0, fmt.Errorf("porte/pg: insert user: %w", err)
		}
	default:
		return 0, fmt.Errorf("porte/pg: resolve user: %w", err)
	}

	// The link only. Tokens and roles arrive through Save, so this method
	// stays the same shape an app implementing UserStore has to satisfy.
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

// Save inserts or updates by (provider, subject).
func (s *IdentityStore) Save(ctx context.Context, identity porte.StoredIdentity) error {
	roles, err := marshalRoles(identity.Roles)
	if err != nil {
		return err
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
func (s *SessionStore) Create(ctx context.Context, session porte.Session) (porte.Session, error) {
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO porte_sessions (token_hash, user_id, label, created_at, last_used_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		session.TokenHash, session.UserID, session.Label,
		session.CreatedAt, session.LastUsedAt, nullTime(session.ExpiresAt),
	).Scan(&session.ID)
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

// Create stores a pending code.
func (s *LoginCodeStore) Create(ctx context.Context, code porte.LoginCode) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO porte_login_codes (code_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		code.CodeHash, code.UserID, code.ExpiresAt)
	if err != nil {
		return fmt.Errorf("porte/pg: create login code: %w", err)
	}
	return nil
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
		RETURNING code_hash, user_id, expires_at`, codeHash, time.Now(),
	).Scan(&code.CodeHash, &code.UserID, &code.ExpiresAt)
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
