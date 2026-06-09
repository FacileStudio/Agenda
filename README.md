# Agenda

Self-hosted calendar for creative studios. Part of the [Facile Suite](https://github.com/FacileStudio).

Personal and shared calendars, CalDAV sync with Apple Calendar, iOS, Thunderbird and Android (DAVx⁵).

A single Go binary serves both the API and the static SvelteKit frontend.

## Quick Start

```bash
cp .env.example .env
# Edit .env — set ENCRYPTION_KEY at minimum
docker compose up --build
```

Open http://localhost:4000 — the API and frontend are both served from there.

## Development

```bash
# Start PostgreSQL
docker compose up db -d

# Terminal 1 — API (Go)
cd apps/api
cp .env.example .env
go run .

# Terminal 2 — Client (SvelteKit)
cd apps/client
cp .env.example .env    # Sets VITE_API_BASE_URL=http://localhost:4000
bun install
bun run dev
```

The client dev server runs on http://localhost:5173.

## Stack

- **API + Server**: Go 1.26, Chi, GORM, PostgreSQL 16 (single binary serves API + frontend)
- **Client**: SvelteKit 5, Svelte 5 runes, Tailwind CSS 4, shadcn-svelte
- **CalDAV**: go-webdav v0.7.0, go-ical — `/dav/` endpoint
- **Auth**: Session tokens + optional OIDC SSO + HTTP Basic Auth (CalDAV clients)
- **Deploy**: Docker Compose (single distroless container + PostgreSQL)

## CalDAV Sync

Configure your CalDAV client with:

- **Server URL**: `https://your-domain.com/dav/`
- **Username**: your email
- **Password**: your Agenda password

Supported: Apple Calendar, iOS Calendar, Thunderbird, DAVx⁵ (Android)

## License

MIT
