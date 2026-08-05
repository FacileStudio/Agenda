# Agenda — Deployment

How the image is built, what Compose declares, and how Agenda is routed on la ruche.

## The image

`Dockerfile` is a three-stage build producing one binary plus the built client:

1. `oven/bun:1` installs `apps/client` dependencies with `--frozen-lockfile` and runs
   `bun run build`, producing static files in `/app/build`.
2. `golang:1.24-alpine` builds the API with
   `go build -mod=vendor -trimpath -ldflags="-s -w" -o /agenda .`. Dependencies come from
   the committed `apps/api/vendor` directory, so the build never reaches the network — and
   so the local go-webdav patches described in [development.md](development.md) are what
   actually ship.
3. `gcr.io/distroless/static-debian12` receives the binary at `/agenda` and the client at
   `/client`.

The runtime stage sets `ENV CLIENT_DIR=/client` explicitly. A distroless base can carry its
own `WorkingDir`, which would make the relative default `./client` resolve somewhere the
SPA is not.

`EXPOSE 4000`, `ENTRYPOINT ["/agenda"]`.

## Compose topology

Two services, and no published host ports — this file is written for Dokploy and Traefik.

| Service | Image | Notes |
|---|---|---|
| `db` | `postgres:16-alpine` | `pg_isready` healthcheck every 5s, volume `postgres_data` |
| `agenda` | built from `Dockerfile` | `env_file: .env`, `expose: 4000`, joins `default` and the external `dokploy-network`, volume `agenda_data` at `/app/data` |

`agenda` waits on `db` being healthy through `depends_on: condition: service_healthy`.

The `db` service reads `DB_NAME`, `DB_USER`, and `DB_PASSWORD` from your `.env` as
`POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD` — the same variables the API uses
to assemble its DSN, so one file configures both sides.

`env_file: .env` is how the API gets its configuration: the Go code loads no `.env` file
of its own.

### The volume and `STORAGE_DIR`

The `agenda_data` volume mounts at `/app/data`, and `STORAGE_DIR` defaults to `./data`.
Those line up only because the image's `WORKDIR` is `/app`. If you override `STORAGE_DIR`,
move the volume mount with it or avatars will be written outside the volume and vanish on
the next deploy.

### Healthcheck

```yaml
test: ["CMD", "/agenda", "healthcheck"]
```

The image is distroless: no shell, no `curl`, no `wget`. `tronc/healthcheck` handles this
by making the binary probe itself — `main` checks `os.Args` first and, when the argument is
`healthcheck`, requests `http://127.0.0.1:$PORT/health` with a three-second timeout and
exits 0 or 1. It targets `127.0.0.1` rather than `localhost` on purpose: in these
containers `localhost` resolves to `::1` first while the server binds `0.0.0.0`, so a
`localhost` probe fails against a perfectly healthy process.

### Traefik labels

One hostname, one service, two routers — plain HTTP redirecting to HTTPS:

```yaml
traefik.enable: "true"
traefik.docker.network: dokploy-network
traefik.http.routers.agenda-web.rule: "Host(`agenda.facile.studio`)"
traefik.http.routers.agenda-web.entrypoints: web
traefik.http.routers.agenda-web.middlewares: redirect-to-https@file
traefik.http.routers.agenda-web.service: agenda-svc
traefik.http.routers.agenda-secure.rule: "Host(`agenda.facile.studio`)"
traefik.http.routers.agenda-secure.entrypoints: websecure
traefik.http.routers.agenda-secure.tls.certresolver: letsencrypt
traefik.http.routers.agenda-secure.service: agenda-svc
traefik.http.services.agenda-svc.loadbalancer.server.port: "4000"
```

This is the suite's one-container / one-router / one-hostname rule. Do not add a
path-prefix router for `/api` — the same binary answers `/api`, `/dav`, `/files`, and the
SPA, and splitting them at the edge would break CalDAV discovery, which requires
`/.well-known/caldav` and `/dav` to sit on the same hostname as everything else.

The load balancer port must match `PORT`. Both are `4000`; change one and you must change
the other.

### CalDAV behind the proxy

Traefik must forward the WebDAV methods `PROPFIND`, `PROPPATCH`, `MKCALENDAR`, and
`REPORT` untouched — they are ordinary HTTP methods to a proxy, so the default
configuration is fine, but a middleware that filters methods would silently break sync.

`X-Forwarded-Proto: https` matters twice: it is what makes the session cookie `Secure` and
what triggers the HSTS header. A proxy that drops it downgrades both.

## Deploying to la ruche

Agenda runs on la ruche behind the Dokploy panel at `gare.facile.studio`. Prefer the
`dokploy` CLI over SSH plus `docker`:

```sh
dokploy compose --help
```

Environment values are set in the Dokploy project, not committed. `.env.example` is the
template — at minimum `DB_USER` and `DB_PASSWORD`, plus `ENCRYPTION_KEY` if you want OIDC
tokens encrypted at rest.

## Migrations

There is no migration step to run. `schemas.Migrate` executes on every boot as a single
GORM `AutoMigrate` over the ten models. A failure aborts startup and the container exits 1,
which Dokploy surfaces as a failed deploy.

`crypto.MigrateOIDCTokens` runs right after, but only when `ENCRYPTION_KEY` is set. It is
best-effort: a failure logs a warning and startup continues.

## Verifying a deploy

`/health` answers as soon as the process is serving and touches nothing, so a green
`/health` proves the binary started and nothing more. `/ready` pings Postgres and is the
one that tells you the database is reachable. Neither says anything about the SPA or about
CalDAV. For the SPA, request `/` and confirm you get the client's `index.html` rather than
a 404 — that is what a missing or misplaced `CLIENT_DIR` looks like. For CalDAV, request
`/.well-known/caldav` and confirm a 302 to `/dav/`.

## Persistent state

| Volume | Mounted at | Holds |
|---|---|---|
| `postgres_data` | `/var/lib/postgresql/data` | The database |
| `agenda_data` | `/app/data` | Uploaded and OIDC-fetched avatars under `/app/data/avatars` |

Avatars are files on disk, not rows. Losing `agenda_data` loses every uploaded avatar;
OIDC-sourced ones come back on the next profile sync, uploaded ones do not.
