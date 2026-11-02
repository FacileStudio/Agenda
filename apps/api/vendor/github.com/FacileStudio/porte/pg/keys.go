package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/keys"
)

// KeySchema defines the SQL schema for porte API keys and usage tracking.
const KeySchema = `
CREATE TABLE IF NOT EXISTS porte_api_keys (
	id              bigserial PRIMARY KEY,
	app             text NOT NULL,
	kind            text NOT NULL DEFAULT 'secret',
	prefix          text NOT NULL,
	token_hash      text NOT NULL UNIQUE,
	allowed_origins jsonb NOT NULL DEFAULT '[]'::jsonb,
	daily_quota     integer NOT NULL DEFAULT 0,
	created_at      timestamptz NOT NULL DEFAULT now(),
	revoked_at      timestamptz
);
CREATE INDEX IF NOT EXISTS porte_api_keys_app_idx ON porte_api_keys (app);
CREATE INDEX IF NOT EXISTS porte_api_keys_token_hash_idx ON porte_api_keys (token_hash) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS porte_api_key_usage (
	api_key_id bigint NOT NULL REFERENCES porte_api_keys(id) ON DELETE CASCADE,
	day        date NOT NULL DEFAULT CURRENT_DATE,
	count      bigint NOT NULL DEFAULT 0,
	PRIMARY KEY (api_key_id, day)
);
`

// KeyStore implements [keys.Store] over PostgreSQL tables.
type KeyStore struct {
	db *sql.DB
}

// Keys returns a key store over the database.
func (s *Store) Keys() *KeyStore {
	return &KeyStore{db: s.db}
}

var _ keys.Store = (*KeyStore)(nil)

// Create inserts a new API key record.
func (s *KeyStore) Create(ctx context.Context, key keys.Key) (keys.Key, error) {
	origins, err := json.Marshal(key.AllowedOrigins)
	if err != nil {
		return keys.Key{}, fmt.Errorf("porte/pg: marshal origins: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
		INSERT INTO porte_api_keys (app, kind, prefix, token_hash, allowed_origins, daily_quota, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		key.App, string(key.Kind), key.Prefix, key.TokenHash, origins, key.DailyQuota, key.CreatedAt,
	).Scan(&key.ID)
	if err != nil {
		return keys.Key{}, fmt.Errorf("porte/pg: create api key: %w", err)
	}

	return key, nil
}

// FindByHash retrieves an API key by its SHA-256 token hash.
func (s *KeyStore) FindByHash(ctx context.Context, tokenHash string) (keys.Key, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app, kind, prefix, token_hash, allowed_origins, daily_quota, created_at, revoked_at
		  FROM porte_api_keys
		 WHERE token_hash = $1`, tokenHash)

	key, err := scanKey(row)
	if stderrors.Is(err, sql.ErrNoRows) {
		return keys.Key{}, porte.ErrNotFound
	}
	if err != nil {
		return keys.Key{}, fmt.Errorf("porte/pg: find api key: %w", err)
	}

	return key, nil
}

// List returns registered API keys optionally filtered by app name.
func (s *KeyStore) List(ctx context.Context, app string) ([]keys.Key, error) {
	query := `
		SELECT id, app, kind, prefix, token_hash, allowed_origins, daily_quota, created_at, revoked_at
		  FROM porte_api_keys`
	args := []any{}
	if app != "" {
		query += ` WHERE app = $1`
		args = append(args, app)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: list api keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []keys.Key
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: scan api key: %w", err)
		}
		out = append(out, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("porte/pg: list api keys: %w", err)
	}

	return out, nil
}

// Revoke stamps revoked_at on an API key by id.
func (s *KeyStore) Revoke(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE porte_api_keys
		   SET revoked_at = now()
		 WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("porte/pg: revoke api key: %w", err)
	}

	return requireOne(res, "porte/pg: revoke api key")
}

// RecordUsage increments today's usage counter for an API key.
func (s *KeyStore) RecordUsage(ctx context.Context, keyID int64, count int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO porte_api_key_usage (api_key_id, day, count)
		VALUES ($1, CURRENT_DATE, $2)
		ON CONFLICT (api_key_id, day) DO UPDATE
		   SET count = porte_api_key_usage.count + EXCLUDED.count`,
		keyID, count)
	if err != nil {
		return fmt.Errorf("porte/pg: record api key usage: %w", err)
	}

	return nil
}

// UsageToday returns a map of key ID to today's consumption count.
func (s *KeyStore) UsageToday(ctx context.Context) (map[int64]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT api_key_id, count
		  FROM porte_api_key_usage
		 WHERE day = CURRENT_DATE`)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: query api key usage today: %w", err)
	}
	defer func() { _ = rows.Close() }()

	usage := make(map[int64]int64)
	for rows.Next() {
		var keyID, count int64
		if err := rows.Scan(&keyID, &count); err != nil {
			return nil, fmt.Errorf("porte/pg: scan api key usage: %w", err)
		}
		usage[keyID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("porte/pg: query api key usage today: %w", err)
	}

	return usage, nil
}

func scanKey(row scanner) (keys.Key, error) {
	var (
		k         keys.Key
		kindStr   string
		origins   []byte
		revokedAt sql.NullTime
	)

	err := row.Scan(
		&k.ID,
		&k.App,
		&kindStr,
		&k.Prefix,
		&k.TokenHash,
		&origins,
		&k.DailyQuota,
		&k.CreatedAt,
		&revokedAt,
	)
	if err != nil {
		return keys.Key{}, err
	}

	k.Kind = keys.Kind(kindStr)
	if revokedAt.Valid {
		k.RevokedAt = &revokedAt.Time
	}

	if len(origins) > 0 {
		_ = json.Unmarshal(origins, &k.AllowedOrigins)
	}
	if k.AllowedOrigins == nil {
		k.AllowedOrigins = []string{}
	}

	return k, nil
}
