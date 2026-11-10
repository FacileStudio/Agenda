// Package migrations embeds the ordered SQL that owns Agenda's schema and
// applies it with tronc/migrate.
//
// Until v0.17.0 the schema came from GORM AutoMigrate plus hand-written
// CREATE TABLE IF NOT EXISTS at every boot. 00001_baseline.sql is the
// production schema those built, dumped with pg_dump; every later file is an
// ordinary migration.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/FacileStudio/tronc/migrate"
)

// FS holds the migration files at its root, which is the layout tronc/migrate
// expects.
//
//go:embed *.sql
var FS embed.FS

// BaselineVersion is the migration that reproduces the schema AutoMigrate built.
const BaselineVersion = 1

// Config is the tronc/migrate configuration over Agenda's migrations.
func Config(db *sql.DB, logger *slog.Logger) migrate.Config {
	return migrate.Config{DB: db, FS: FS, Logger: logger}
}

// Up applies every pending migration. A database built by AutoMigrate, with
// Agenda's tables and no applied migration, is first stamped at the baseline
// through tronc/migrate's own baseline command, so the baseline is recorded
// as applied without running and only the later migrations execute.
func Up(ctx context.Context, cfg migrate.Config) error {
	legacy, err := builtByAutoMigrate(ctx, cfg.DB)
	if err != nil {
		return err
	}
	if legacy {
		if cfg.Logger == nil {
			cfg.Logger = slog.Default()
		}
		cfg.Logger.Info("database predates migrations, recording the baseline", slog.Int("version", BaselineVersion))
		args := []string{"agenda", "migrate", "baseline", fmt.Sprint(BaselineVersion)}
		if _, err := migrate.Command(ctx, args, cfg); err != nil {
			return err
		}
	}
	return migrate.Run(ctx, cfg)
}

// builtByAutoMigrate reports whether the current schema holds Agenda's users
// table but no applied migration. A ledger can exist without one: goose
// creates it, with only its version 0 sentinel, the first time anything reads
// it, `migrate status` included. Every name resolves through search_path, so a
// test schema is judged on its own tables.
func builtByAutoMigrate(ctx context.Context, db *sql.DB) (bool, error) {
	var hasUsers, hasLedger bool
	err := db.QueryRowContext(ctx,
		`SELECT to_regclass('users') IS NOT NULL, to_regclass($1) IS NOT NULL`, migrate.TableName,
	).Scan(&hasUsers, &hasLedger)
	if err != nil {
		return false, fmt.Errorf("migrations: inspect schema: %w", err)
	}
	if !hasUsers || !hasLedger {
		return hasUsers, nil
	}
	var applied bool
	err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+migrate.TableName+` WHERE version_id > 0)`).Scan(&applied)
	if err != nil {
		return false, fmt.Errorf("migrations: read the ledger: %w", err)
	}
	return !applied, nil
}
