package schemas

import "gorm.io/gorm"

// AdoptPorte moves Agenda's credentials onto porte's tables.
//
// issuer is the configured OIDC_ISSUER, and it has to be passed in because it
// is half of the account matching key: porte finds an identity by (provider,
// subject), and the provider is the issuer. Backfilling with the wrong value —
// or with a placeholder — would leave every existing SSO user unmatched, which
// degrades silently to the email fallback rather than failing.
//
// An empty issuer skips only the identity backfill. The sessions, the API
// tokens and the passwords still move, because they are what keeps people
// signed in and able to sign in again.
func AdoptPorte(db *gorm.DB, issuer string) error {
	statements := []string{porteSchema, carryLegacySessionsOver, carryAPITokensOver, adoptExistingPasswords}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if issuer == "" {
		return nil
	}
	return db.Exec(adoptOIDCSubjects, issuer).Error
}

// porteSchema is porte/pg's Schema with its porte_users table left out and the
// foreign keys repointed at Agenda's own users.
//
// porte offers UserStore as the escape hatch for exactly this. This users row
// carries the avatar columns, an upload path and the mailbox ownership every
// account, folder and email hangs off; moving it would be a rewrite of
// everything except authentication. The other three stores come from porte/pg
// unchanged — they only ever touch the tables below.
//
// Kept verbatim from porte otherwise, column for column: pg's queries are
// written against these names, and a divergence here surfaces as a runtime
// error on the login path rather than at boot.
const porteSchema = `
CREATE TABLE IF NOT EXISTS porte_identities (
	user_id         bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

CREATE TABLE IF NOT EXISTS porte_sessions (
	id           bigserial PRIMARY KEY,
	token_hash   text NOT NULL UNIQUE,
	user_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
	user_id     bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at  timestamptz NOT NULL,
	consumed_at timestamptz
);
ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS consumed_at timestamptz;
`

// carryLegacySessionsOver moves the pre-porte sessions table across instead of
// dropping it, so adopting porte does not sign every existing user out.
//
// Both tables store the SHA-256 hex of a 32-byte token and never the token
// itself, so a row copied here keeps authenticating the credential already in
// somebody's browser — and porte's AcceptLegacyCookie keeps reading it from
// the `session` cookie this app has always set.
//
// last_used_at is stamped now rather than copied from created_at: porte
// retires a browser session idle for seven days and the old table recorded no
// use at all, so carrying created_at over would sign out everyone who last
// signed in more than a week ago, on the deploy meant to keep them.
const carryLegacySessionsOver = `
DO $$
BEGIN
	IF to_regclass('sessions') IS NOT NULL THEN
		INSERT INTO porte_sessions (token_hash, user_id, created_at, last_used_at, expires_at)
		SELECT s.token, s.user_id, s.created_at, now(), s.expires_at
		  FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.expires_at > now()
		ON CONFLICT (token_hash) DO NOTHING;
		DROP TABLE sessions;
	END IF;
END
$$;
`

// carryAPITokensOver folds the separate api_tokens table into porte's sessions
// as labelled rows.
//
// porte.Session.Label exists for precisely this: what Agenda called an API
// token is a session with a name on it and no expiry, which is one mechanism
// more than the problem needs and a second branch in the authentication path.
// A null expires_at never expires, and porte's idle window deliberately does
// not apply to a labelled session, so a token wired into a script keeps
// working.
const carryAPITokensOver = `
DO $$
BEGIN
	IF to_regclass('api_tokens') IS NOT NULL THEN
		INSERT INTO porte_sessions (token_hash, user_id, label, created_at, last_used_at, expires_at)
		SELECT t.token, t.user_id, COALESCE(NULLIF(t.name, ''), 'API token'), t.created_at, now(), NULL
		  FROM api_tokens t JOIN users u ON u.id = t.user_id
		ON CONFLICT (token_hash) DO NOTHING;
		DROP TABLE api_tokens;
	END IF;
END
$$;
`

// adoptExistingPasswords moves the argon2 hashes from users.password_hash into
// the identity rows porte/local reads.
//
// Without it the deploy silently ends password login for every existing
// account: the hash is still in the users table, nothing reads it there any
// more, and the login form answers "invalid credentials" to a correct
// password with no error anywhere. The hashes are byte-identical — porte/local
// uses the parameters this app already used — so the move is a copy and nobody
// resets anything.
//
// The subject is the lowercased address because that is what porte/local
// normalises to before looking one up; a row keyed on a mixed-case address
// would simply never be found.
//
// users.password_hash is deliberately left in place. Blanking it in the same
// deploy makes the change unrollbackable for the sake of tidiness, and a
// column nothing reads can be dropped on any later day.
const adoptExistingPasswords = `
INSERT INTO porte_identities (user_id, provider, subject, password_hash)
SELECT id, 'local', lower(btrim(email)), password_hash
  FROM users
 WHERE coalesce(password_hash, '') <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`

// adoptOIDCSubjects moves the federated identity off the user row and into the
// identity table porte reads.
//
// Without it nothing breaks loudly: porte finds no identity for (issuer,
// subject), falls back to matching the verified email, finds the same user and
// links it on the next login. But the email fallback is the weaker path by
// design — it is the one an identity provider that lets a user set any address
// can abuse — and relying on it for every existing account, on the one deploy
// where it would go unnoticed, is not a trade worth making.
//
// The OIDC tokens deliberately do **not** come across. Agenda stores them
// encrypted with ENCRYPTION_KEY and porte stores them as it will send them, so
// copying the ciphertext across would hand porte a refresh token that is not
// one: the first profile sync would fail, clear both, and look like the
// provider had revoked them. They are re-issued in full on the user's next SSO
// login, and the only thing missing until then is a background profile
// refresh.
const adoptOIDCSubjects = `
INSERT INTO porte_identities (user_id, provider, subject)
SELECT id, ?, oidc_subject
  FROM users
 WHERE oidc_subject IS NOT NULL AND oidc_subject <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`
