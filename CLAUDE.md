# Agenda

Self-hosted calendar for creative studios. Part of the Facile Suite.

## Tech Stack

| Layer    | Stack                                                        |
| -------- | ------------------------------------------------------------ |
| API      | Go 1.26, Chi router, GORM, PostgreSQL 16                    |
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

## Environment Variables

- `DATABASE_URL` — PostgreSQL connection string (default: local postgres, db `agenda`)
- `PORT` — API port (default `4000`)
- `LOG_LEVEL` — `debug`, `info`, `warn`, `error`
- `STORAGE_DIR` — File storage for avatars (default `./data`)
- `ENCRYPTION_KEY` — Optional, encrypts OIDC tokens at rest
- `DOMAINS` — Comma-separated CORS origins (only needed for separate client deploy)
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
