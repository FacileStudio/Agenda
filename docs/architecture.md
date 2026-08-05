# Agenda — Architecture

How a request reaches the database, what the tables look like, how the CalDAV server is
wired, and why five URL prefixes live outside `/api`.

## Runtime topology

```
Internet ──▶ Traefik ──▶ Go binary (:4000) ──┬──▶ /health /ready       liveness, readiness
                                              ├──▶ /api/*              six modules
                                              ├──▶ /dav, /dav/*        CalDAV, Basic Auth
                                              ├──▶ /.well-known/caldav 302 to /dav/
                                              ├──▶ /files/*            STORAGE_DIR
                                              ├──▶ /auth/oidc/callback 302 to /api twin
                                              └──▶ /*                  SPA catch-all, gzipped
                                                              │
                                                        Postgres 16
                                                              │
                        Journal (HTTP) ◀── slog handler

  Apple Calendar / iOS / Thunderbird / DAVx⁵ ──▶ /dav (PROPFIND, REPORT, MKCALENDAR, …)
```

One process, one container, one Traefik router.

## The URL space, and why it is shaped that way

`mountRoutes` in `apps/api/main.go` claims every URL the app answers. The whole JSON API
lives under `/api`, so an unknown API path 404s instead of falling through to the SPA
catch-all and returning an HTML shell with a 200. Five things deliberately stay at the
root because their URLs are held outside this repo:

| Prefix | Held by |
|---|---|
| `/dav`, `/.well-known/caldav` | External CalDAV clients, and RFC 6764 for the well-known path |
| `/files/*` | `users.avatar_url` rows already stored as `/files/avatars/...` |
| `/health`, `/ready` | `tronc/health.Mount`, which registers them at both the root and `/api` |
| `/auth/oidc/callback` | Authentik's registered redirect URI, pinned by `OIDC_REDIRECT_URL` |

The legacy OIDC callback is a 302 to `/api/auth/oidc/callback`, keeping an existing
Authentik registration working without moving the rest of the API.

`main_test.go` asserts this: `TestEveryCalledURLIsRouted` walks a list of the URLs the
client calls and fails if one no longer matches a route,
`TestUnknownAPIPathsDoNotReachTheSPA` guards the 404 behavior,
`TestHealthSurvivesTheAPIMount` guards the probes, and
`TestLegacyOIDCCallbackRedirectsUnderAPI` guards the redirect. Add to `clientCalls` when
you add an endpoint.

## Components

| Piece | Where | What it does |
|---|---|---|
| Entrypoint | `apps/api/main.go` | Config, DB, migrations, OIDC token migration, router, graceful shutdown |
| Router | `tronc/httpx.NewRouter` | Chi router with the suite's logging, recovery, and CORS middleware |
| Modules | `apps/api/modules/*` | `auth`, `calendars`, `events`, `spaces`, `users`, `settings`, `caldav` |
| Schemas | `apps/api/schemas` | Ten GORM models plus `Migrate`, which is a plain `AutoMigrate` |
| Internal | `apps/api/internal/*` | `database`, `middleware`, `env`, `crypto`, `authcrypto`, `authcontext`, `oidcavatar`, `documentation` |
| Client | `apps/client` | SvelteKit 5 with `adapter-static` and `fallback: index.html` |

Six modules register under `/api`; `caldav` registers at the root.

## Middleware

Applied globally in `main.go`, in order: the `tronc/httpx` stack (request logger, panic
recovery, CORS with `AllowCredentials: true`), then `middleware.SecurityHeaders`, then
`middleware.MaxBodySize(4 << 20)` — a 4 MB cap on every request body.

`SecurityHeaders` sets `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
`X-XSS-Protection: 0`, `Referrer-Policy: strict-origin-when-cross-origin`, and a
`Permissions-Policy` denying camera, microphone, and geolocation. It adds HSTS only when
the request arrived over TLS or carries `X-Forwarded-Proto: https`.

Gzip is applied to the SPA handler alone, not to the API.

Rate limits are per-route: 3 requests per minute on `POST /api/auth/register`, 10 on
`POST /api/auth/login`, and 100 on the CalDAV endpoint.

## Authentication

`middleware.RequireAuth` resolves a token from three places, in order:

1. The `session` cookie — `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` whenever the
   request arrived over TLS or through a proxy setting `X-Forwarded-Proto: https`.
2. The `Authorization` header.
3. A `?token=` query parameter, which logs a deprecation warning.

Register and login both set the session cookie and return the token in the body. Sessions
live 30 days (`auth.SessionTTL`). `POST /api/auth/logout` deletes the row and clears the
cookie. Tokens are stored hashed in `sessions`; `api_tokens` holds the long-lived
per-user alternative.

### OIDC

Additive. `env.Load` builds an `OIDCConfig` only when `OIDC_ISSUER` is set, and then
requires `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, and `OIDC_REDIRECT_URL`. When configured,
`GET /api/auth/oidc`, `GET /api/auth/oidc/callback`, and `POST /api/auth/sync-profile`
appear, plus the root-level redirect described above.

`OIDC_SUCCESS_URL` defaults to the first entry of the CORS origin list **that starts with
`http://` or `https://`**. That filter exists because the CORS list is also fed by
`DOMAIN`, which deployments set to a bare hostname; redirecting to a bare hostname would
resolve relative to the callback path instead of leaving the site.

`SSO_ONLY=true` removes register and login from the router entirely — they 404, not 403.

### Encryption at rest

When `ENCRYPTION_KEY` is set, `crypto.DeriveKey` takes its SHA-256 and uses the result as
an AES-256-GCM key for the OIDC access and refresh tokens stored on the user row. On every
boot, `crypto.MigrateOIDCTokens` scans for plaintext tokens and encrypts them in place, so
turning the key on later back-fills existing rows. The migration is best-effort: a failure
logs a warning and startup continues.

## Data model

| Table | Key columns | Notes |
|---|---|---|
| `users` | `id`, `email` unique, `name`, `password_hash` | Plus `avatar_url`, `avatar_source`, `oidc_picture_url`, the encrypted OIDC tokens, and `profile_synced_at` |
| `sessions` | `token` PK (hashed), `user_id`, `expires_at` | 30-day sessions |
| `api_tokens` | `token` PK (hashed), `user_id`, `name` | One long-lived token per user; also accepted as a CalDAV password |
| `app_settings` | single row `id = 1`, `encryption_key` | Surfaced through `/api/settings` as a boolean only |
| `spaces` | `id`, `name`, `description` | |
| `space_members` | unique `(space_id, user_id)`, `role` | Cascades on delete of either side |
| `calendars` | `id`, `owner_id`, `space_id`, `cal_dav_path`, `slug`, `name`, `color`, `is_personal`, `sync_token` | `echo_url` holds a per-calendar conference URL |
| `calendar_members` | unique `(calendar_id, user_id)`, `role` | Roles are `reader`, `writer`, `admin` |
| `events` | `id`, unique `(uid, calendar_id)`, `etag`, `sequence`, `start_at`, `end_at`, `raw_ics` | Also `is_all_day`, `recurrence_rule`, `status`, `conference_url`, `conference_provider` |
| `event_attendees` | `event_id`, `user_id`, `email`, `response` | `response` is `needs-action`, `accepted`, `declined`, or `tentative` |

`schemas.Migrate` is a single `AutoMigrate` over those ten models, run on every boot.
There are no migration files and no backfills.

Two columns exist for CalDAV round-trip fidelity: `events.raw_ics` keeps the original
iCalendar payload a client sent, and `calendars.sync_token` is bumped on every event
create, update, and delete so clients can detect changes cheaply.

## CalDAV

`modules/caldav` wraps `github.com/emersion/go-webdav/caldav` with a GORM-backed
`Backend`. The URL space under the `/dav` prefix:

```
/dav/{email}                                 principal
/dav/{email}/calendars                       calendar home set
/dav/{email}/calendars/{calendarID}          calendar collection
/dav/{email}/calendars/{calendarID}/{uid}.ics  event
```

Three things the registration does that are easy to miss:

- **Chi only knows standard HTTP methods.** `chi.RegisterMethod` is called for `PROPFIND`,
  `PROPPATCH`, `MKCALENDAR`, `REPORT`, `COPY`, `MOVE`, `LOCK`, and `UNLOCK` so they land in
  chi's method bitmask and `HandleFunc` matches them at all.
- **go-webdav has no `MKCALENDAR` support**, only `MKCOL`, which makes Apple Calendar's
  "create calendar" 405. The handler intercepts `MKCALENDAR` and routes it to
  `backend.HandleMkcalendar`.
- **`/.well-known/caldav` is unauthenticated** and 302s to `/dav/`. RFC 6764 requires it to
  be reachable before a client has credentials, and the redirect is deliberately 302 rather
  than 308 because iOS has known problems with 308 on well-known redirects.

Authentication accepts a `session` cookie, or HTTP Basic with the user's email plus either
their account password or an API token — the API token path is what lets SSO-only users,
who have no password, connect a CalDAV client at all. iOS 18.4 and later percent-encode
`@` in the Basic Auth username, so the username is URL-decoded before lookup.

The vendored go-webdav copy carries local patches. They are documented in
[development.md](development.md#the-vendored-go-webdav-patch).

## Cross-app integration

Logs ship to Journal when both `JOURNAL_URL` and `JOURNAL_TOKEN` are set; the Journal SDK
wraps the `slog` handler. Agenda is **not** wired into the Nook Pool — there is no `pool`
or `enveloppe` dependency in `go.mod`. Identity federation, when enabled, goes through
Authentik over standard OIDC like the rest of the suite.
