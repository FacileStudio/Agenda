# Agenda — Development

Getting a local Agenda running, the tasks that exist, the quality gate, and the one thing
in this repo you must not regenerate blindly.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| Go | 1.24 | `apps/api/go.mod` declares `go 1.24.0`; `mise.toml` pins `go = "1.24"` |
| Bun | any recent | Client package manager and dev server |
| Docker | any recent | Postgres 16 for local development |
| mise | optional | Runs the tasks below; each is a one-line shell command you can run by hand |

## Setup

```sh
mise run hooks
mise run install
docker compose up db -d
```

`mise run hooks` points `core.hooksPath` at `.githooks`, which wires up the pre-push
quality gate. `mise run install` runs `bun install --frozen-lockfile` in `apps/client`.

The `db` service takes `DB_USER` and `DB_PASSWORD` from your `.env`, so copy
`.env.example` first even for a database-only start.

## Running

The API takes configuration from the process environment — **there is no `.env` loading in
the Go code** and there is no `apps/api/.env.example`. Export what you need:

```sh
cd apps/api
DB_USER=agenda DB_PASSWORD=change-me DB_HOST=localhost \
  CORS_ALLOWED_ORIGINS=http://localhost:5173 go run .
```

The client, in another terminal:

```sh
cd apps/client
cp .env.example .env
bun run dev
```

The API listens on `:4000` and the client on `:5173`. There is no Vite proxy here: the
client calls the API cross-origin at `VITE_API_BASE_URL`, which is why the API needs the
dev origin in its CORS list. CORS is configured with `AllowCredentials: true` so the
session cookie survives the hop.

Migrations run automatically at startup via `schemas.Migrate`. There is no separate
migration tool and no migration files.

## Tasks

| Task | Command | What it does |
|---|---|---|
| `mise run install` | `bun install --frozen-lockfile` in `apps/client` | Client dependencies |
| `mise run check` | `sh ./scripts/check.sh` | The full quality gate: Go, then the client |
| `mise run check-go` | `sh ./scripts/check.sh --go-only` | Go half only |
| `mise run format` | `sh ./scripts/check.sh --format` | `go fmt ./...`, rewriting files in place |
| `mise run hooks` | `git config core.hooksPath .githooks` | Enables the tracked hooks in this clone |

Client scripts, run from `apps/client`: `bun run dev`, `bun run build`, `bun run preview`,
`bun run check` (`svelte-check` against `tsconfig.json`).

## The quality gate

`scripts/check.sh` is the gate, and `.githooks/pre-push` does nothing but exec it. It
reports and never rewrites, except under `--format`:

1. `gofmt -l .` over `apps/api`, ignoring `vendor/`. Any listed file fails the gate.
2. `go vet ./...`
3. `go test ./...`
4. `bun run check` in `apps/client`, skipped with a warning if `bun` is not on `PATH`.

Two deliberate details worth knowing before you "fix" the script:

- **It is not invoked through mise.** `mise run` resolves every tool in the merged config
  before running any task body, so one broken tool in your global mise config would take
  the gate down with it. The hook calls `sh` directly.
- **It resolves the toolchain from `GOROOT`.** mise exports `GOROOT` for the pinned version
  but can leave an unrelated `go` earlier on `PATH`; mixing them produces
  `compile: version "X" does not match go tool version "Y"`.

Bypass once with `git push --no-verify`.

## Tests

```sh
cd apps/api
go test ./...
```

| File | What it covers |
|---|---|
| `main_test.go` | Routing: every URL the client calls is routed, unknown `/api` paths do not reach the SPA, health survives the `/api` mount, the legacy OIDC callback redirects |
| `modules/caldav/backend_test.go` | `MKCALENDAR` body parsing, tasks-only calendar detection, calendar-object fallback |
| `modules/users/service_test.go` | Avatar files land in `STORAGE_DIR` and only managed avatars are deleted |
| `internal/crypto/crypto_test.go` | AES-GCM round trip, empty string, nonce uniqueness, wrong key, deterministic key derivation |

`main_test.go` carries a `clientCalls` list. **Add to it when you add an endpoint** — that
is the mechanism keeping the client and the router from drifting apart.

There is no client-side test setup.

## The vendored go-webdav patch

`apps/api/vendor/github.com/emersion/go-webdav` is **not a clean copy of v0.7.0.** It
carries four local fixes, all of them for Apple client compatibility. Running
`go mod vendor` will silently throw them away and break iOS sync, with no compile error to
warn you.

| Fix | File | Why |
|---|---|---|
| `propFindRoot` returns the requested path as the response href, not the principal path | `caldav/server.go` | iOS ignores a PROPFIND response whose href does not match the requested URL, so it never discovered `current-user-principal` and left the account inactive |
| Trailing-slash normalization on principal and home-set path comparisons | `caldav/server.go` | Clients that append `/` still match |
| `cs:getctag` exposed in calendar PROPFIND | `caldav/caldav.go`, `caldav/elements.go`, `caldav/server.go` | Apple's CalendarServer extension; lets iOS and macOS detect changes without fetching every event. Adds a `SyncToken` field to `caldav.Calendar` |
| `PropPatch` answers 207 Multi-Status with 403 per property instead of a bare 501 | `caldav/server.go` | iOS treats a 501 as a fatal server error and may mark the account inactive; a 403 propstat reads as "read-only property" and is handled gracefully |

If you must re-vendor, do it and then re-apply these by hand — the commits are
`ece0543`, `73082f0`, and `65dfa13`.

## Dependencies

Go dependencies are vendored in `apps/api/vendor`, and the Docker build runs
`go build -mod=vendor`. After changing anything in `go.mod`:

```sh
cd apps/api
go mod tidy
go mod vendor
```

Then check the go-webdav patch survived, per the section above.

## Conventions

- Svelte 5 runes only. UI primitives come from shadcn-svelte in `src/lib/components/ui/`.
- Each API module keeps the same layout: `router.go` wires paths, `controller.go` decodes
  and shapes, `service.go` holds the logic and GORM calls, `types.go` holds the request and
  response structs.
- Every new JSON endpoint goes under `/api`. Root-level paths are reserved for URLs held
  outside this repo — see [architecture.md](architecture.md#the-url-space-and-why-it-is-shaped-that-way).
- `calendars.sync_token` must be bumped on every event mutation. Both `modules/events` and
  `modules/caldav` have their own `bumpSyncToken`; a write path that skips it leaves Apple
  clients showing stale data.
