# Agenda — Configuration

Every environment variable the API actually reads, taken from `apps/api/internal/env` and
`tronc/env`.

**Agenda does not load a `.env` file.** Unlike most Go apps in the suite, there is no
`godotenv` call in `env.Load` — configuration comes from the process environment only. In
Docker that is what `env_file: .env` in `docker-compose.yml` provides; running `go run .`
by hand means exporting the variables yourself.

## Database

Agenda is the one Go app in the suite that **cannot use `troncenv.LoadCore`**, because
`LoadCore` requires `DATABASE_URL` and the deployment only sets `DB_USER` and
`DB_PASSWORD`. `internal/env.loadCore` fills `troncenv.Core` by hand instead, and
`resolveDatabaseURL` assembles the DSN.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `DATABASE_URL` | see below | — | Full Postgres DSN. When set, every `DB_*` variable is ignored |
| `DB_USER` | unless `DATABASE_URL` | — | DSN username |
| `DB_PASSWORD` | unless `DATABASE_URL` | — | DSN password |
| `DB_HOST` | no | `db` | DSN host |
| `DB_PORT` | no | `5432` | DSN port |
| `DB_NAME` | no | `agenda` | DSN database name |
| `DB_SSLMODE` | no | `disable` | Appended as `?sslmode=` |

**The trap:** with neither `DATABASE_URL` nor both of `DB_USER` and `DB_PASSWORD`, startup
fails with `set DATABASE_URL, or DB_USER and DB_PASSWORD, to connect to PostgreSQL` and
the process exits 1. Setting only one of the pair is the same as setting neither.

## Core

| Variable | Required | Default | What it does |
|---|---|---|---|
| `PORT` | no | `4000` | HTTP listen port. Must be a valid TCP port or startup fails |
| `APP_ENV` | no | `development` | `development`, `staging`, or `production`. Never gates security behavior |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `CORS_ALLOWED_ORIGINS` | no | — | Comma-separated browser origins, scheme included |
| `JOURNAL_URL` | no | — | Journal ingest URL for log shipping |
| `JOURNAL_TOKEN` | no | — | Journal per-app key. Logs ship only when both this and `JOURNAL_URL` are set |

The `PORT` default is `4000`, not `tronc`'s `8080` — `loadCore` passes `4000` as the
fallback to `troncenv.Int`. Compose pins `PORT: "4000"` and the Traefik service port
matches; changing one means changing all three.

CORS is only needed when the client is served from a different origin than the API. In the
single-container deployment the SPA is served by the same binary, so the list can be empty.

### CORS name fallbacks

`troncenv.CORSOrigins` reads the first of these that is set, in order:
`CORS_ALLOWED_ORIGINS`, `ALLOWED_ORIGINS`, `DOMAINS`, `DOMAIN`, `CORS_ORIGINS`,
`TRUSTED_ORIGINS`, `CLIENT_ORIGIN`. Because `DOMAIN` is often a bare hostname,
`OIDC_SUCCESS_URL`'s default skips any entry that does not start with `http://` or
`https://`.

## Storage and client

| Variable | Required | Default | What it does |
|---|---|---|---|
| `STORAGE_DIR` | no | `./data` | Root for uploaded files. `STORAGE_DIR/avatars` is created at startup and served under `/files/` |
| `CLIENT_DIR` | no | `./client` | Directory holding the built SPA. The image sets `/client` explicitly |

`CLIENT_DIR` is read by `tronc/spa`. If the directory has no `index.html`, the SPA
catch-all is not mounted and the binary serves the API and CalDAV alone.

## Encryption

| Variable | Required | Default | What it does |
|---|---|---|---|
| `ENCRYPTION_KEY` | no | — | Passphrase for OIDC token encryption at rest |

Any string works — `crypto.DeriveKey` takes its SHA-256 to produce the 32-byte AES key,
so the "32 characters" hint in `.env.example` is a suggestion, not a requirement. With the
key unset, OIDC tokens are stored in plaintext and `MigrateOIDCTokens` is skipped
entirely. Setting it later encrypts existing rows on the next boot.

Changing the key after tokens have been encrypted makes them undecryptable. There is no
re-key path.

## Auth and OIDC

| Variable | Required | Default | What it does |
|---|---|---|---|
| `SSO_ONLY` | no | `false` | When true, the register and login routes are not registered at all |
| `OIDC_ISSUER` | no | — | Discovery URL, for example `https://porte.facile.studio/application/o/agenda/` |
| `OIDC_CLIENT_ID` | with issuer | — | Client id from the provider |
| `OIDC_CLIENT_SECRET` | with issuer | — | Client secret from the provider |
| `OIDC_REDIRECT_URL` | with issuer | — | Registered redirect URI |
| `OIDC_SUCCESS_URL` | no | first absolute CORS origin | Where the callback redirects after a successful login |

Setting `OIDC_ISSUER` without all three of the client id, secret, and redirect URL fails
startup with an explicit error.

The callback is served at `/api/auth/oidc/callback`. A registration pointing at the
legacy `/auth/oidc/callback` keeps working: that root path is registered as a 302 to its
`/api` twin whenever OIDC is configured.

## Compose-only

`DB_NAME`, `DB_USER`, and `DB_PASSWORD` are also consumed by the `db` service in
`docker-compose.yml` as `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD`. They are
the same variables, deliberately, so one `.env` configures both sides.

## Client

`apps/client/src/lib/backend.ts` reads `VITE_API_BASE_URL` at build time, strips any
trailing slash, and falls back to the empty string — meaning same-origin. The production
image never sets it, because the Go binary serves the SPA. In development,
`apps/client/.env.example` sets it to `http://localhost:4000`, and the API then needs
`http://localhost:5173` in its CORS origin list.
