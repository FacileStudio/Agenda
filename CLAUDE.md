# Agenda

Self-hosted calendar for creative studios. Part of the Facile Suite.

## Tech Stack

| Layer    | Stack                                                        |
| -------- | ------------------------------------------------------------ |
| API      | Go 1.24, Chi router, GORM, PostgreSQL 16                    |
| Client   | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, muse        |
| Build    | `go build -mod=vendor`, Bun                                 |
| Deploy   | Docker Compose (single container: distroless Go binary)      |
| Auth     | Sign-in through Registre (OIDC) only; API tokens over HTTP Basic Auth for CalDAV clients |
| CalDAV   | go-webdav v0.7.0, go-ical — endpoint `/dav/`                |

## Project Structure

```
Dockerfile                Unified multi-stage build (bun build + go build -> distroless)
docker-compose.yml        Two services: db + agenda
.env.example              Root-level env template (production)
apps/
  api/                    Go backend
    main.go               Entrypoint: env, DB, migrations, router, static file serving, graceful shutdown
    modules/              Domain modules
      auth/               porte/kit store, session middleware, erasure (EraseData)
      users/              User profile management (copied from Courrier)
      settings/           App settings (copied from Courrier)
      calendars/          Calendar CRUD + sharing + member management
      spaces/             Read-only spaces + the Registre directory sync (import, Apply, nudge)
      events/             Event CRUD, attendees, reminders, occurrence exceptions
      caldav/             CalDAV server (go-webdav backend + Basic Auth middleware)
    internal/             Shared infra (database, middleware, logger, env, errors, etc.)
    schemas/              GORM models (User, Session, ApiToken, Calendar, CalendarMember, Event, EventAttendee)
    vendor/               Vendored Go dependencies
  client/                 SvelteKit frontend
    src/
      routes/             SvelteKit file-based routing
        (app)/            Authenticated layout group (calendar, settings, profile)
        login/            Login page
      lib/
        backend.ts        API client on @facile/client (relative /api, CSRF, 401 to /login)
        theme.svelte.ts   @facile/client theme store under the agenda.theme key
        components/       App components (UI primitives come from @facile/muse)
```

## Commands

```sh
mise run db                 # Postgres on :5442, credentials injected from Casier
mise run dev                # API on :4000, secrets injected from Casier
mise run dev-offline        # the same, from casier's last-known-good cache
mise run client             # client dev server on :5173
mise run secrets            # what Casier holds for agenda/dev
mise run check-secrets      # casier check .env.example
mise run check              # the quality gate (scripts/check.sh)
```

### API (`apps/api/`)

```sh
go build -mod=vendor -o bin/api .
# no plain go mod vendor: it drops the go-webdav patch (see Conventions)
```

`go run .` by hand needs the environment exported by hand — the Go code never calls
`godotenv.Load()` and there is no `apps/api/.env.example`. Prefer `mise run dev`.

### Client (`apps/client/`)

```sh
bun install
bun run dev                 # Dev server on :5173, proxying /api, /files and /dav to :4000
bun run build
bun run check
```

### Full Stack (Docker)

```sh
cp .env.example .env
docker compose up --build
```

## CalDAV

CalDAV endpoint: `/dav/`
- Principal: `/dav/{userEmail}`
- Calendar home: `/dav/{userEmail}/calendars`
- Calendar: `/dav/{userEmail}/calendars/{calendarID}`
- Event: `/dav/{userEmail}/calendars/{calendarID}/{uid}.ics`

Supported clients: Apple Calendar, iOS Calendar, Thunderbird, DAVx⁵ (Android)

Auth: HTTP Basic Auth (email + API token) or session cookie

## Routing

Everything the API serves lives under `/api` (`mountRoutes` in `apps/api/main.go`), so an
unknown API path 404s instead of falling through to the SPA catch-all. Four things stay at
the root because their URLs are held outside this repo:

- `/dav/`, `/.well-known/caldav` — external CalDAV clients
- `/files/*` — avatar URLs are stored in `users.avatar_url` as `/files/avatars/...`
- `/health`, `/ready` — tronc mounts them at both `/` and `/api`
- `/auth/oidc/callback` — 302s to `/api/auth/oidc/callback`; the old URL is registered in
  Authentik and pinned by `OIDC_REDIRECT_URL`

`main_test.go` asserts every URL the client calls still matches a route. Add to
`clientCalls` when you add an endpoint.

## Environment Variables

`internal/env` embeds `troncenv.Core`, so `PORT`, `LOG_LEVEL`, `DATABASE_URL`, `APP_ENV`,
`JOURNAL_URL`, `JOURNAL_TOKEN` and the CORS origins are read by tronc.

- `DB_USER`, `DB_PASSWORD` — required unless `DATABASE_URL` is set; the DSN is assembled
  from them plus `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SSLMODE`. This is why `Load` fills
  `Core` by hand instead of calling `troncenv.LoadCore`, which requires `DATABASE_URL`.
- `APP_ENV` — `development` (default), `staging`, `production`
- `PORT` — API port (default `4000`)
- `LOG_LEVEL` — `debug`, `info`, `warn`, `error`
- `GET /version` answers `v` plus the version in `apps/client/package.json`, baked at build time by `scripts/version.sh`; no env is involved.
- `STORAGE_DIR` — File storage for avatars (default `./data`)
- `CLIENT_DIR` — Static SvelteKit build served by the binary
- `ENCRYPTION_KEY` — Optional, encrypts OIDC tokens at rest
- `CORS_ALLOWED_ORIGINS` — Comma-separated origins, scheme included (only needed for a
  separate client deploy). `ALLOWED_ORIGINS`, `DOMAINS`, `DOMAIN`, `CORS_ORIGINS`,
  `TRUSTED_ORIGINS` and `CLIENT_ORIGIN` are read as fallbacks.
- `ANTENNE_URL`, `ANTENNE_SECRET`, `ANTENNE_INSTANCE_ID` — Optional Antenne Pool, read by antenne-client v1's `ConfigFromEnv` (`ANTENNE_INSTANCE` is the older spelling of the URL); a URL without a secret fails boot. Agenda only listens: it mirrors Opus tasks with a due date into a read-only "Opus tasks" calendar (`internal/pool`) and publishes nothing, so it has no outbox
- `OIDC_*` — OpenID Connect config; `OIDC_ISSUER` is required in production

## Conventions

- **Single binary architecture** — Go binary serves SvelteKit static build from `CLIENT_DIR`
- **Go modules are vendored, with a patched go-webdav** — never run plain `go mod vendor`: add the require and go.sum lines by hand and copy the new package into `vendor/` plus `vendor/modules.txt`
- **Migrations run on startup** via `schemas.Migrate(db)`
- **Svelte 5 runes** enforced (`$state`, `$props`, `$derived`, `$effect`)
- **@facile/muse** for UI primitives
- **CalDAV sync token** — stored per calendar, bumped on every event create/update/delete
- **Spaces belong to Registre** — `modules/spaces` imports once, then applies Registre snapshots; Agenda has no space write endpoints and the UI links to `https://sso.facile.studio/spaces`
- **One calendar per space** — `schemas.SyncSpaceCalendars` (run by `Migrate` and after every Registre snapshot) gives each space exactly one calendar, named after it and pinned by the partial unique index `idx_calendars_one_per_space`; extra or orphaned space calendars are handed back to their owner as personal ones. Access to a space's calendar is space membership alone (owner/admin → admin, member → writer), resolved by `schemas.CalendarRoleExpr`, which calendars, events and CalDAV all share. Only personal calendars are created, deleted or shared one by one
- **Mirrored events** — `events.source` / `events.external_id` mark events owned by another app (Opus tasks); user edits and CalDAV PUT/DELETE are rejected
- **Event raw ICS** — stored in `events.raw_ics` for round-trip fidelity with CalDAV clients
- **Event ICS is the source of truth** — attendees, reminders, EXDATE and RECURRENCE-ID overrides live only in `events.raw_ics` (rendered by `modules/events/ics_write.go`, read by `details.go`), so CalDAV clients and the API agree; `color` and `timezone` are columns
