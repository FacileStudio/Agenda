# Changelog — tronc/migrate

`migrate` is a separate module with its own tags, so it has its own history. Tags are prefixed
(`migrate/v0.1.0`); consumers still request plain semver (`@v0.1.0`).

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
semver — while on `v0`, a breaking change bumps the minor.

## [Unreleased]

## [0.1.0] — 2026-08-06

### Added

- `Run` applies pending migrations at boot, holding a Postgres session advisory lock so two
  instances starting together cannot both migrate.

- `Command` exposes the same runner as a `migrate` subcommand — `status`, `version`, `up`,
  `baseline`. It reaches a distroless container because the image's `ENTRYPOINT` is the binary,
  which is the only way in: five of the suite's six Go images have no shell, no `psql` and no
  migration CLI, and Dokploy has no pre-deploy hook.

  Unlike `healthcheck.Handle`, `Command` returns its error instead of calling `os.Exit`. The
  exit paths of `healthcheck` are the one part of it no test can reach; this package has no
  database in CI, so being testable at the boundary mattered more than symmetry.

- `baseline <n>` records migrations as applied without running them, for a database whose schema
  already exists. goose has no equivalent — [#431](https://github.com/pressly/goose/issues/431)
  and [#938](https://github.com/pressly/goose/issues/938) are open and
  [PR #954](https://github.com/pressly/goose/pull/954) has been unmerged since 2025 — and the
  documented workaround is a raw `INSERT` through a `psql` these containers do not ship.

  It asks goose to create the ledger table rather than emitting the DDL itself: goose inserts a
  `version_id = 0` sentinel row on creation and every later command fails with
  `missing zero version migration` without it. Hand-copied DDL would also drift on any goose
  release. It refuses to stamp a database that already has migration history.

### Notes

- Requires **Go 1.25.7**, which is goose's own floor and higher than `tronc`'s 1.24. The three
  suite repos that build offline (`GOFLAGS=-mod=vendor GOPROXY=off`) cannot rely on Go's
  automatic toolchain download, so adopting this means bumping their `go.mod` and their
  `golang:1.24-alpine` builder image together.

- The only dependency is `github.com/pressly/goose/v3`. Importing it as a library pulls **no
  database driver**: every driver in goose sits behind a build tag in `cmd/goose`. Measured on
  v3.27.3 — four pure-Go indirect dependencies, and `go list -deps` matches zero packages under
  pgx, sqlite, mysql or clickhouse. The consumer owns its driver, exactly as `tronc/health`
  takes a `*sql.DB` it did not open.

- This module depends on nothing from `tronc`, which keeps their release cadences independent.
  A nested module *may* require its parent — `golang.org/x/tools/gopls` does — but then the
  parent has to be tagged first, and there is no reason to buy that ordering here.

[Unreleased]: https://github.com/FacileStudio/tronc/compare/migrate/v0.1.0...HEAD
[0.1.0]: https://github.com/FacileStudio/tronc/releases/tag/migrate%2Fv0.1.0
