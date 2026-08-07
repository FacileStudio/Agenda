// Package testdb connects a test to a real PostgreSQL database.
//
// It is deliberately Postgres and nothing else. The suite's SQL is Postgres SQL
// — SELECT ... FOR UPDATE, string_to_array, LATERAL, ILIKE, pg_trgm, catalog
// queries — and the schema now ships as hand-written Postgres DDL under
// migrations/. A test against in-memory SQLite builds a different schema from
// the struct tags and then passes, proving nothing about what runs in
// production.
//
// Each test binary gets its own Postgres schema, so `go test ./...` keeps its
// package-level concurrency instead of being forced through -p 1:
//
//	func TestMain(m *testing.M) {
//		url, ok := testdb.URL()
//		if !ok {
//			testdb.Announce("sh scripts/postgres-up.sh")
//		} else {
//			db, err := testdb.Open(url, testdb.Config{
//				Prefix:  "capsule_test",
//				Migrate: runMigrations,
//			})
//			...
//		}
//		os.Exit(m.Run())
//	}
//
// A missing TEST_DATABASE_URL skips rather than fails, so a developer without a
// Postgres can still run the gate — but see Announce for why the skip is loud.
package testdb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EnvVar names the connection string the database-backed tests read.
const EnvVar = "TEST_DATABASE_URL"

// LedgerTable is goose's migration ledger, which Truncate must never empty.
// It matches migrate.TableName; it is repeated rather than imported so that a
// test helper does not drag a migration engine into every consumer.
const LedgerTable = "goose_db_version"

// Config describes one application's test database.
type Config struct {
	// Prefix names the per-binary schemas, as in "capsule_test". Required.
	Prefix string

	// Migrate brings a freshly created schema up to date. Required. Apps on
	// goose pass a closure over migrate.Run; anything still on AutoMigrate can
	// pass schemas.Migrate until it converts.
	Migrate func(*gorm.DB) error

	// Open overrides how the connection is made, for an app whose GORM config
	// matters to its tests — a naming strategy, say. Defaults to the same
	// silent-logger, TranslateError setup every Facile app uses.
	Open func(string) (*gorm.DB, error)

	// Keep lists tables Truncate must not empty, on top of LedgerTable.
	Keep []string
}

// URL returns the configured connection string, or false when tests should skip.
func URL() (string, bool) {
	value := strings.TrimSpace(os.Getenv(EnvVar))
	return value, value != ""
}

// Announce writes the skip banner to stderr, naming the command that turns the
// tests on. t.Skip only shows under -v, and a gate nobody notices is off is
// worse than no gate at all.
func Announce(bootstrap string) {
	fmt.Fprintf(os.Stderr, "\n=== SKIPPED: %s is not set, so the database-backed tests did not run ===\n\n%s\n\n",
		EnvVar, SkipReason(bootstrap))
}

// SkipReason is the message to hand t.Skip. It spells out the whole recipe,
// because whoever reads it has to be able to act without opening another file.
func SkipReason(bootstrap string) string {
	if bootstrap == "" {
		bootstrap = "# start a PostgreSQL and create a scratch database, then:"
	}
	return EnvVar + ` is not set, so the database-backed tests did not run.

Point them at a scratch database:

    ` + bootstrap + `
    export ` + EnvVar + `='postgres://localhost:5433/scratch?sslmode=disable'`
}

// SchemaName is the schema this test binary owns. It is derived from the
// compiled binary's name, whose base is the package under test, so it is stable
// across runs and distinct across packages.
//
// Apps that migrate with goose should derive their advisory lock id from this:
// the lock is scoped to the database, not the schema, so every test package
// sharing one test database otherwise queues on the same default id.
func SchemaName(prefix string) string {
	base := strings.TrimSuffix(filepath.Base(os.Args[0]), ".test")
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		default:
			return '_'
		}
	}, base)
	if safe == "" {
		safe = "anonymous"
	}
	return prefix + "_" + safe
}

// Open creates this binary's schema, connects scoped to it, and migrates.
func Open(url string, cfg Config) (*gorm.DB, error) {
	if cfg.Prefix == "" {
		return nil, fmt.Errorf("testdb: Prefix is required")
	}
	if cfg.Migrate == nil {
		return nil, fmt.Errorf("testdb: Migrate is required")
	}
	open := cfg.Open
	if open == nil {
		open = defaultOpen
	}
	schema := SchemaName(cfg.Prefix)

	admin, err := open(withTimeout(url))
	if err != nil {
		return nil, fmt.Errorf("testdb: open %s: %w", EnvVar, err)
	}
	if err := admin.Exec(`CREATE SCHEMA IF NOT EXISTS "` + schema + `"`).Error; err != nil {
		return nil, fmt.Errorf("testdb: create schema %s: %w", schema, err)
	}
	if handle, err := admin.DB(); err == nil {
		_ = handle.Close()
	}

	// The search_path has to travel in the connection string, not in a SET.
	// GORM hands out pooled connections, so a SET would bind one of them and
	// every other query in the package would quietly run against public.
	scoped, err := open(withSearchPath(withTimeout(url), schema))
	if err != nil {
		return nil, fmt.Errorf("testdb: open %s scoped to %s: %w", EnvVar, schema, err)
	}
	if err := cfg.Migrate(scoped); err != nil {
		return nil, fmt.Errorf("testdb: migrate %s: %w", schema, err)
	}
	return scoped, nil
}

// Truncate empties every table in the current schema and restarts the identity
// sequences, so a re-seeded fixture always lands on the same primary keys. The
// migration ledger is left alone.
//
// Truncate-and-reseed is used rather than a transaction per test because
// service layers open transactions of their own on the shared handle, and their
// fire-and-forget goroutines use that same connection concurrently.
func Truncate(db *gorm.DB, cfg Config) error {
	skip := map[string]bool{LedgerTable: true}
	for _, name := range cfg.Keep {
		skip[name] = true
	}

	var tables []string
	if err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = current_schema()`).Scan(&tables).Error; err != nil {
		return fmt.Errorf("testdb: list tables: %w", err)
	}

	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		// Emptying goose's ledger would be worse than useless: TRUNCATE does not
		// drop the tables, so the next migration run would replay from version 0
		// and fail on "relation already exists".
		if skip[table] {
			continue
		}
		quoted = append(quoted, `"`+table+`"`)
	}
	if len(quoted) == 0 {
		return nil
	}

	statement := "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	if err := db.Exec(statement).Error; err != nil {
		return fmt.Errorf("testdb: truncate: %w", err)
	}
	return nil
}

func defaultOpen(url string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(url), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
}

// withSearchPath appends the scoping parameter. pgx forwards unknown URL
// parameters to the server as runtime settings, which is what makes this work.
func withSearchPath(url, schema string) string {
	if !isURL(url) {
		return url + " search_path=" + schema
	}
	return url + separator(url) + "search_path=" + schema
}

// withTimeout bounds the connect attempt. pgx has no default, so a DSN pointing
// at a blackholed host blocks in TCP connect until the OS gives up — around 75
// seconds, per attempt, inside a pre-push hook.
func withTimeout(url string) string {
	if strings.Contains(url, "connect_timeout") {
		return url
	}
	if !isURL(url) {
		return url + " connect_timeout=5"
	}
	return url + separator(url) + "connect_timeout=5"
}

// isURL distinguishes postgres://host/db from the libpq key/value form
// (host=... dbname=...), which takes space-separated pairs rather than a query.
func isURL(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://")
}

func separator(url string) string {
	if strings.Contains(url, "?") {
		return "&"
	}
	return "?"
}
