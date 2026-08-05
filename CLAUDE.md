# Agenda

Self-hosted calendar for creative studios. Part of the Facile Suite.

## Tech Stack

| Layer    | Stack                                                        |
| -------- | ------------------------------------------------------------ |
| API      | Go 1.24, Chi router, GORM, PostgreSQL 16                    |
| Client   | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, shadcn-svelte |
| Build    | `go build -mod=vendor`, Bun                                 |
| Deploy   | Docker Compose (single container: distroless Go binary)      |
| Auth     | Session tokens + optional OIDC SSO + HTTP Basic Auth (CalDAV clients) |
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
      auth/               Session auth + OIDC SSO (copied from Courrier)
      users/              User profile management (copied from Courrier)
      settings/           App settings (copied from Courrier)
      calendars/          Calendar CRUD + sharing + member management
      events/             Event CRUD with date range filtering
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
        backend.ts        API client (fetch wrapper)
        components/       App components + shadcn-svelte ui/ primitives
```

## Commands

### API (`apps/api/`)

```sh
cp .env.example .env
go run .                    # Dev server on :4000
go build -mod=vendor -o bin/api .
go mod vendor               # After changing dependencies
```

### Client (`apps/client/`)

```sh
bun install
bun run dev                 # Dev server on :5173 (needs VITE_API_BASE_URL in .env)
bun run build
bun run check
```

### Full Stack (Docker)

```sh
cp .env.example .env
docker compose up --build
docker compose up db -d     # Just PostgreSQL for local dev
```

## CalDAV

CalDAV endpoint: `/dav/`
- Principal: `/dav/{userEmail}`
- Calendar home: `/dav/{userEmail}/calendars`
- Calendar: `/dav/{userEmail}/calendars/{calendarID}`
- Event: `/dav/{userEmail}/calendars/{calendarID}/{uid}.ics`

Supported clients: Apple Calendar, iOS Calendar, Thunderbird, DAVx⁵ (Android)

Auth: HTTP Basic Auth (email + password) or session cookie

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
- `STORAGE_DIR` — File storage for avatars (default `./data`)
- `CLIENT_DIR` — Static SvelteKit build served by the binary
- `ENCRYPTION_KEY` — Optional, encrypts OIDC tokens at rest
- `CORS_ALLOWED_ORIGINS` — Comma-separated origins, scheme included (only needed for a
  separate client deploy). `ALLOWED_ORIGINS`, `DOMAINS`, `DOMAIN`, `CORS_ORIGINS`,
  `TRUSTED_ORIGINS` and `CLIENT_ORIGIN` are read as fallbacks.
- `OIDC_*` — OpenID Connect config (optional)
- `SSO_ONLY` — Hide password auth when `true`

## Conventions

- **Single binary architecture** — Go binary serves SvelteKit static build from `CLIENT_DIR`
- **Go modules are vendored** — run `go mod vendor` after changing dependencies
- **Migrations run on startup** via `schemas.Migrate(db)`
- **Svelte 5 runes** enforced (`$state`, `$props`, `$derived`, `$effect`)
- **shadcn-svelte** for UI primitives in `src/lib/components/ui/`
- **CalDAV sync token** — stored per calendar, bumped on every event create/update/delete
- **Event raw ICS** — stored in `events.raw_ics` for round-trip fidelity with CalDAV clients
