# Agenda

Self-hosted calendar for creative studios and freelancers. Personal and shared calendars,
with real CalDAV sync to the apps people already use.

One Go binary serves the JSON API, the CalDAV endpoint, and the built SvelteKit client, so
a deployment is a single container behind a single Traefik router.

Live at [agenda.facile.studio](https://agenda.facile.studio).

## What it does

- Personal and shared calendars, with per-calendar `reader`, `writer`, and `admin` roles
- Spaces that group members and scope the calendars belonging to them
- Events with locations, all-day flags, recurrence rules, attendees, and conference links
- CalDAV server at `/dav` with `.well-known` discovery, so Apple Calendar, iOS,
  Thunderbird, and DAVx⁵ sync natively
- Email/password auth with optional OIDC SSO and an `SSO_ONLY` mode
- HTTP Basic Auth for CalDAV clients, accepting either the account password or an API token
- OIDC access and refresh tokens encrypted at rest with AES-GCM

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.24, Chi v5, GORM, PostgreSQL 16, [tronc](https://github.com/FacileStudio/tronc) v0.6.0 |
| CalDAV | go-webdav v0.7.0 (vendored and patched), go-ical |
| Client | SvelteKit 5 (runes), Tailwind CSS 4, shadcn-svelte, bits-ui |
| Deploy | Docker Compose, one distroless container behind Traefik |

## Quick start

`docker-compose.yml` publishes no host port and expects the external `dokploy-network`
with Traefik in front of it. It is how Agenda deploys, not how you browse it locally.

```sh
cp .env.example .env
docker compose up -d --build
```

### Local development

Start Postgres, then the API and the client in separate terminals.

```sh
mise run install
docker compose up db -d
```

```sh
cd apps/api
DB_USER=agenda DB_PASSWORD=change-me DB_HOST=localhost go run .
```

```sh
cd apps/client
cp .env.example .env
bun run dev
```

The API does not read a `.env` file — it takes everything from the process environment.
The client runs on <http://localhost:5173> and calls the API at `VITE_API_BASE_URL`, so
the API needs `CORS_ALLOWED_ORIGINS=http://localhost:5173` for the browser to reach it.

## Configuration

| Variable | What it does |
|---|---|
| `DB_USER`, `DB_PASSWORD` | Required unless `DATABASE_URL` is set; the DSN is assembled from them |
| `DATABASE_URL` | Full Postgres DSN, which short-circuits the `DB_*` assembly |
| `ENCRYPTION_KEY` | Passphrase for encrypting OIDC tokens at rest. Optional but recommended |
| `PORT` | HTTP listen port, `4000` by default in this repo |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins, only needed for a split deploy |
| `STORAGE_DIR` | Root for uploaded avatars, served back under `/files/` |

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
apps/
  api/       Go backend — modules/ (auth, calendars, events, spaces, users,
             settings, caldav), schemas/ (GORM models), vendor/ (patched go-webdav)
  client/    SvelteKit 5 SPA, built into the API image and served by it
scripts/     check.sh, the quality gate the pre-push hook runs
docs/        Architecture, configuration, development, deployment, API
```

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Request flow, data model, CalDAV, how the pieces fit |
| [Configuration](docs/configuration.md) | Every environment variable and default |
| [Development](docs/development.md) | Local setup, tests, the quality gate, the vendored patch |
| [Deployment](docs/deployment.md) | Docker Compose, Dokploy, Traefik routing |
| [API](docs/api.md) | HTTP endpoints, payloads, and the CalDAV URL space |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
