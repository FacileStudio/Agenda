# tronc/testdb

Connects a Go test to a real PostgreSQL database, one schema per test binary.

A **separate module**, like [`migrate`](../migrate/). It needs `gorm.io/gorm` and
`gorm.io/driver/postgres`, which `tronc` itself must never depend on. Every Go repo in the suite
already has both, so adopting this adds nothing to a consumer's dependency graph.

## Why

Postgres is the only database in this suite, and the schema now ships as hand-written Postgres
DDL under `migrations/`. Several repos were testing against in-memory SQLite, which builds a
*different* schema from the GORM struct tags and then passes — a schema-drift detector with the
batteries removed. Capsule proved the point: `modules/pastes/service.go` does
`SELECT ... FOR UPDATE`, which SQLite has no concept of, so the burn-after-read concurrency guard
was the one thing its tests could not cover.

Casier wrote the original; this is that harness, parameterised and with three bugs fixed.

## Use

```go
var testDB *gorm.DB

func TestMain(m *testing.M) {
	url, ok := testdb.URL()
	if !ok {
		testdb.Announce("sh scripts/postgres-up.sh")
	} else {
		db, err := testdb.Open(url, cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		testDB = db
	}
	os.Exit(m.Run())
}

var cfg = testdb.Config{
	Prefix:  "capsule_test",
	Migrate: func(db *gorm.DB) error { /* migrate.Run over the embedded migrations */ },
}
```

Then reset between cases with `testdb.Truncate(testDB, cfg)`, and skip the database-backed tests
with `t.Skip(testdb.SkipReason("sh scripts/postgres-up.sh"))` when `testDB` is nil.

| Symbol | Purpose |
|---|---|
| `URL() (string, bool)` | reads `TEST_DATABASE_URL`; false means skip |
| `Open(url, Config)` | creates this binary's schema, connects scoped to it, migrates |
| `Truncate(db, Config)` | empties every table **except the migration ledger** |
| `SchemaName(prefix)` | the schema this binary owns — derive a goose `LockID` from it |
| `Announce(bootstrap)` | the loud skip banner, on stderr |
| `SkipReason(bootstrap)` | the message for `t.Skip` |

## The parts that are load-bearing

- **`search_path` travels in the DSN, never a `SET`.** GORM hands out pooled connections, so a
  `SET` binds one of them and every other query in the package quietly runs against `public`.
- **`Truncate` skips `goose_db_version`.** Emptying it is worse than useless: `TRUNCATE` does not
  drop the tables, so the next migration run replays from version 0 and dies on
  `relation "..." already exists`. Order-dependent, and a genuinely confusing afternoon.
- **A missing `TEST_DATABASE_URL` skips loudly, on stderr.** `t.Skip` only shows under `-v`, and
  a gate nobody notices is off is worse than no gate.
- **`connect_timeout=5` is added when the DSN lacks one.** pgx has no default, so a DSN pointing
  at a blackholed host blocks in TCP connect until the OS gives up — around 75 seconds, per
  attempt, inside a pre-push hook.
- **One schema per test binary**, named from `os.Args[0]`, so `go test ./...` keeps its
  package-level concurrency instead of needing `-p 1`.

## Known limits

- Two concurrent runs of the *same* package share a schema and will truncate each other. The
  schema name is deterministic on purpose — it makes schemas reusable — but it means an IDE test
  run racing a pre-push hook will misbehave.
- `t.Parallel()` within a package is not supported: `Truncate` is global to the schema.
- goose's advisory lock is scoped to the **database**, not the schema. Pass a `LockID` derived
  from `SchemaName` or every test package queues on the same default.

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
