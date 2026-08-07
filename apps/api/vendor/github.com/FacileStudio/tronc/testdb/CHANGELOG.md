# Changelog — tronc/testdb

`testdb` is a separate module with its own tags, so it has its own history. Tags are prefixed
(`testdb/v0.1.0`); consumers still request plain semver (`@v0.1.0`).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
semver — while on `v0`, a breaking change bumps the minor.

## [Unreleased]

## [0.1.1] — 2026-08-06

### Fixed

- A test compared `SchemaName("x")` with itself, which staticcheck reads as SA4000 — an
  identical expression on both sides of `!=`. The property is worth pinning (a random suffix
  would leave `Open` connected to a schema it never created), so it is now asserted through
  variables, alongside a new check that two prefixes cannot collide. No change to the package.

## [0.1.0] — 2026-08-06

### Added

- `Open`, `Truncate`, `URL`, `SchemaName`, `Announce`, `SkipReason` — a real PostgreSQL for
  tests, one schema per test binary, so `go test ./...` keeps its package-level concurrency.

  Lifted from Casier's `internal/testdb`, which was the only database harness in the suite not
  running on SQLite. Four repos were testing against in-memory SQLite, building a schema from
  GORM struct tags while production runs hand-written Postgres DDL from `migrations/` — a
  schema-drift detector with the batteries removed. Capsule is the proof: its
  `SELECT ... FOR UPDATE` burn-after-read guard is the one thing SQLite could never cover.

### Fixed

Three defects in the harness this was lifted from, all of which would have been copied:

- `Truncate` emptied goose's `goose_db_version` ledger along with everything else, because the
  ledger is unqualified and therefore lands in the per-test schema. `TRUNCATE` does not drop
  tables, so the next migration run replayed from version 0 and failed on
  `relation "..." already exists` — and only on the second `Open` in a binary, making it
  order-dependent.
- No `connect_timeout` in the DSN. pgx has no default, so a DSN pointing at a blackholed host
  blocked in TCP connect for around 75 seconds per attempt, inside a pre-push hook.
- `withSearchPath` appended `?search_path=` unconditionally, which silently corrupts the libpq
  key/value DSN form (`host=... dbname=...`).

### Notes

- Requires Go 1.24, the same floor as the root module.
- Depends on `gorm.io/gorm` and `gorm.io/driver/postgres`, which the root module must never
  have. Every Go repo in the suite already has both, so a consumer gains nothing new.
- Deliberately not part of `migrate`: that module takes a `*sql.DB` and never a driver, and
  mixing a test harness into it would drag GORM in behind it.

[Unreleased]: https://github.com/FacileStudio/tronc/compare/testdb/v0.1.1...HEAD
[0.1.1]: https://github.com/FacileStudio/tronc/releases/tag/testdb%2Fv0.1.1
[0.1.0]: https://github.com/FacileStudio/tronc/releases/tag/testdb%2Fv0.1.0
