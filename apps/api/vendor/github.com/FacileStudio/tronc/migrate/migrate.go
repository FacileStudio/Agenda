// Package migrate applies ordered database migrations from inside the app binary.
//
// Five of the suite's Go APIs run gcr.io/distroless/static-debian12: no shell,
// no psql, no migration CLI, and Dokploy offers no pre-deploy hook. A migration
// can therefore only be applied by the application itself. Run does that at
// boot; Command exposes the same runner as a subcommand, which reaches even a
// distroless container because the image's ENTRYPOINT is the binary:
//
//	docker run <image> migrate status
//
// Wire both in, after the database is open:
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	migrations, err := fs.Sub(migrationsFS, "migrations")
//	cfg := migrate.Config{DB: sqlDB, FS: migrations, Logger: appLogger}
//
//	if handled, err := migrate.Command(ctx, os.Args, cfg); handled {
//		if err != nil {
//			appLogger.Error("migrate", slog.Any("error", err))
//			return 1
//		}
//		return 0
//	}
//	if err := migrate.Run(ctx, cfg); err != nil {
//		return 1
//	}
//
// This is a separate Go module so that goose stays out of the dependency graph
// of every other tronc consumer, including the ones with no database at all.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"
)

// TableName is the ledger table recording which migrations have been applied.
const TableName = goose.DefaultTablename

// Config describes one application's migrations.
type Config struct {
	// DB is the pool to migrate. GORM applications get it from gormDB.DB().
	// It must permit more than one open connection: the advisory lock pins one
	// for the whole run, so a pool capped at one deadlocks against itself.
	DB *sql.DB

	// FS holds the migration files at its root. An embed.FS preserves the
	// directory it embedded, so pass it through fs.Sub to strip the prefix.
	FS fs.FS

	// Logger receives per-migration progress and the run summary. Defaults to
	// slog.Default().
	Logger *slog.Logger

	// LockID overrides the Postgres advisory lock identifier. Advisory locks
	// are scoped per database, so the default is already isolated while each
	// application owns its own; set this only when two services share one.
	LockID int64
}

func (c Config) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return slog.Default()
}

// newProvider builds the goose provider. It performs no database round-trip,
// which is what lets the tests below run without a Postgres.
func newProvider(cfg Config) (*goose.Provider, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("migrate: DB is required")
	}
	if cfg.FS == nil {
		return nil, fmt.Errorf("migrate: FS is required")
	}

	lockID := cfg.LockID
	if lockID == 0 {
		lockID = lock.DefaultLockID
	}
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(lockID))
	if err != nil {
		return nil, fmt.Errorf("migrate: session locker: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, cfg.DB, cfg.FS,
		goose.WithSessionLocker(locker),
		goose.WithSlog(cfg.logger()),
		// goose logs nothing unless it is verbose, and a migration that takes
		// minutes should say so while it runs rather than only at the end.
		goose.WithVerbose(true),
		// Nothing in this suite registers Go migrations through init(), and a
		// package that started to would otherwise be picked up silently.
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return provider, nil
}

// Run applies every pending migration in order, holding a Postgres advisory
// lock for the duration. It is a no-op when the schema is already current.
//
// The provider is deliberately not closed: goose's Close closes the underlying
// *sql.DB, which belongs to the caller.
func Run(ctx context.Context, cfg Config) error {
	provider, err := newProvider(cfg)
	if err != nil {
		return err
	}
	return up(ctx, provider, cfg.logger())
}

// Command runs the migrate subcommand and reports whether it recognised one, so
// a caller can fall through to a normal start. It returns errors rather than
// exiting, which is what makes it testable.
func Command(ctx context.Context, args []string, cfg Config) (bool, error) {
	if len(args) < 2 || args[1] != "migrate" {
		return false, nil
	}

	subcommand := "status"
	if len(args) > 2 {
		subcommand = args[2]
	}

	// Validate arguments before touching the database, so a typo fails fast.
	var baselineVersion int64
	if subcommand == "baseline" {
		if len(args) < 4 {
			return true, fmt.Errorf("migrate: baseline needs a version, as in 'migrate baseline 1'")
		}
		parsed, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil || parsed < 1 {
			return true, fmt.Errorf("migrate: baseline needs a positive version, got %q", args[3])
		}
		baselineVersion = parsed
	}

	provider, err := newProvider(cfg)
	if err != nil {
		return true, err
	}

	switch subcommand {
	case "up":
		return true, up(ctx, provider, cfg.logger())
	case "status":
		return true, status(ctx, provider)
	case "version":
		return true, version(ctx, provider)
	case "baseline":
		return true, baseline(ctx, provider, cfg.DB, baselineVersion, cfg.logger())
	default:
		return true, fmt.Errorf("migrate: unknown subcommand %q, want up, status, version or baseline", subcommand)
	}
}

func up(ctx context.Context, provider *goose.Provider, log *slog.Logger) error {
	results, err := provider.Up(ctx)
	if err != nil {
		// A partial failure names the migration that broke and how far the run
		// got. Without this the operator sees only the driver's error and has
		// to read the ledger to find out what actually landed.
		var partial *goose.PartialError
		if errors.As(err, &partial) {
			return fmt.Errorf("migrate: version %d failed after %d applied: %w",
				partial.Failed.Source.Version, len(partial.Applied), partial.Err)
		}
		return fmt.Errorf("migrate: %w", err)
	}

	if len(results) == 0 {
		log.Debug("database schema is current")
		return nil
	}

	highest := results[len(results)-1].Source.Version
	log.Info("database migrations applied",
		slog.Int("count", len(results)),
		slog.Int64("version", highest),
	)
	return nil
}

func status(ctx context.Context, provider *goose.Provider) error {
	items, err := provider.Status(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, item := range items {
		applied := "-"
		if item.State == goose.StateApplied {
			applied = item.AppliedAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(os.Stdout, "%-6d %-8s %-22s %s\n",
			item.Source.Version, item.State, applied, item.Source.Path)
	}
	return nil
}

func version(ctx context.Context, provider *goose.Provider) error {
	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, current)
	return nil
}

// baseline records migrations up to target as applied without running them, for
// a database whose schema already exists — every Facile database, all of them
// built by GORM AutoMigrate with no migration history.
//
// goose has no stamp command of its own (pressly/goose#431, #938, PR #954 is
// unmerged), and the raw-SQL workaround cannot reach a distroless container.
func baseline(ctx context.Context, provider *goose.Provider, db *sql.DB, target int64, log *slog.Logger) error {
	// HasPending creates the ledger table and the version 0 sentinel row that
	// goose refuses to operate without, and is documented to skip the lock.
	// Hand-writing that DDL instead would drift on any goose release.
	if _, err := provider.HasPending(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if current > 0 {
		return fmt.Errorf("migrate: database is already at version %d; baseline only stamps a database with no migration history", current)
	}

	var stamp []int64
	for _, source := range provider.ListSources() {
		if source.Version <= target {
			stamp = append(stamp, source.Version)
		}
	}
	if len(stamp) == 0 || stamp[len(stamp)-1] != target {
		return fmt.Errorf("migrate: no migration has version %d", target)
	}

	store, err := database.NewStore(goose.DialectPostgres, TableName)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, recorded := range stamp {
		if err := store.Insert(ctx, db, database.InsertRequest{Version: recorded}); err != nil {
			return fmt.Errorf("migrate: recording version %d: %w", recorded, err)
		}
	}

	log.Info("database baselined",
		slog.Int64("version", target),
		slog.Int("recorded", len(stamp)),
	)
	return nil
}
