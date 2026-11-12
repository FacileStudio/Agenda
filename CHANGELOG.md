# Changelog

All notable changes to Agenda are documented here.

## [Unreleased]

### Security

- fix(auth): an account re-created for a person Registre erased is erased again on the next deletion push or poll, without a restart (porte v0.19.0, `directory.Deletions.Holds`). Before, a bearer token minted before the erasure, or an MCP first call, could bring the account back and it stayed until the next restart.
- fix(auth): sign-in, bearer and MCP provisioning, and profile sync refuse an identity Agenda already erased (`oidc.Deps.Erased`), and the Registre space sync no longer provisions an erased member from a stale snapshot.
- build: porte v0.19.0 and porte/directory v0.5.0, which also raise go-chi/chi to v5.5.0, go-jose to v4.1.9 and x/sync, x/sys and x/text. Re-vendored with the go-webdav caldav patch kept.

## [0.18.3] - 2026-11-11

### Fixed

- fix(brand): favicons, touch and manifest icons, the link preview image and `site.webmanifest` come from muse v0.19.2's brand generator, and `app.html` carries its head block: the catalog title "Agenda · The shared studio calendar", its description, theme colours and Open Graph and Twitter tags. The landing page no longer overrides the title and description, and the landing header, footer and sign-in screen show the catalog icon `solar:calendar-add-bold-duotone`.

## [0.18.2] - 2026-11-11

### Fixed

- build: clear the douane high and medium findings. Client: vite 7.3.7 to 7.3.8, @sveltejs/kit 2.70.4, svelte 5.57.3, and refreshed transitive deps (devalue, postcss, source-map-js, nanoid). `overrides` pin cookie ^0.7.2 and esbuild ^0.28.1. API: go-chi/chi v5.3.1 to v5.4.0, re-vendored by hand with the go-webdav patch kept (`TestVendoredGoWebdavPatches` passes).
- build: left open: @sveltejs/kit findings fixed only in 3.0.0 (low) and deepmerge (no fix released).

## [0.18.1] - 2026-11-11

### Changed

- MCP: the server and the confirm page run on tronc/mcp v0.22.0's shared `NewApp` and `PendingPage`. Agenda's own list, confirm and dismiss handlers and the `PendingList`, `PendingResult` and `EmptyRequest` types are gone. Tool names and schemas do not change, and the event digest keeps its field list, so actions opened before this release still confirm.
- MCP: a delegated token on `/api/mcp/pending` now gets "only the person's own session can confirm actions" (code still `permission_denied`). The API reference names the list schema `PendingList_PendingItem`, documents the `id` path param on confirm and dismiss, and an optional `url` in the result.
- build: tronc v0.21.0 to v0.22.0. Re-vendored with the go-webdav patch kept (`TestVendoredGoWebdavPatches` passes).

## [0.18.0] - 2026-11-10

### Added

- Bus: Agenda publishes its calendars and events on the Antenne pool, on `calendar.created`, `calendar.updated`, `calendar.deleted`, `calendar_event.created`, `calendar_event.updated` and `calendar_event.deleted` (antenne-client v1.1.0). Every write queues its change in `antenne_outbox` inside the write's own transaction, so a change is published if and only if it commits: the REST calendar and event handlers (including occurrence edits and the events a deleted calendar held), CalDAV PUT, DELETE, MKCOL and MKCALENDAR, the MCP tools, and account erasure. The pool client drains the outbox with `EmitNow`. Payloads carry only `envelope.Calendar` and `envelope.CalendarEvent` fields, never a description, a location or an attendee; an erased person's events and calendars go out as deletions carrying the ids only. `facile_id` is the row's integer id as a decimal string. Events mirrored from Opus and the calendars the Registre space sync creates or renames are not published. Nothing is queued or sent when `ANTENNE_URL` is unset.
- Database: `00003_antenne_outbox.sql`, antenne-client's `outbox.Migration` byte for byte.

### Changed

- Bus: the pool connection uses antenne-client's `WithPersistentConnect`, which retries the first connection and every later one by itself; Agenda's own dial loop and `WithMaxReconnect(math.MaxInt32)` are gone. Agenda declares the six calendar channels it emits when it registers.
- build: antenne-client/go v1.0.0 to v1.1.0. Re-vendored with the go-webdav patch kept (`TestVendoredGoWebdavPatches` passes).

## [0.17.0] - 2026-11-10

### Changed

- Database: the schema is ordered goose SQL under `apps/api/migrations/`, applied at boot by `tronc/migrate` (migrate v0.1.0) behind a Postgres advisory lock. GORM `AutoMigrate` and the hand-written `CREATE TABLE IF NOT EXISTS` are gone from the boot path; the GORM structs stay as the query layer, and `TestGORMModelsMatchTheMigratedSchema` fails when a struct names a column the migrations do not create or a type its field cannot hold. `scripts/check.sh` fails on an `.AutoMigrate(` call outside a test file.
- Database: `00001_baseline.sql` is production's schema as `pg_dump --schema-only` printed it on 2026-11-10. A database built before this release (Agenda's tables, no applied migration) is stamped at version 1 on its first boot with tronc/migrate's `baseline` command, so the baseline is recorded without running. A database built from the migrations dumps identical to production. `apps/api/schema.sql` is the normalized dump of the whole set.
- Boot: the data steps that ran at every boot still run at every boot, after the migrations and in the same order (`schemas.Reconcile`): the avatar column backfill, one calendar per space, the porte identity re-key, the pre-porte session and API token carry-over, the password and OIDC subject adoption, and the Registre picture backfill. `crypto.MigrateOIDCTokens` is unchanged.
- Gates: CI runs filet v0.19.1. `filet gate` skips its gofmt step, which scans `vendor/`; `scripts/check.sh`, the first pre-push job, runs gofmt with `vendor/` excluded.
- build: `github.com/FacileStudio/tronc/migrate` v0.1.0 with `pressly/goose/v3` v3.27.3; goose raises `segmentio/asm` from v1.1.3 to v1.2.1. Re-vendored with the go-webdav patch kept (`TestVendoredGoWebdavPatches` passes).

### Added

- `docker run <image> migrate status` (also `up`, `version`, `baseline <n>`) answers without a shell.
- Database: `00002_column_defaults.sql` gives `created_at`, `updated_at` and `joined_at` a `now()` default on every table that has them, and the users columns porte/kit does not write the values the kit's `OnCreate` writes (`''` and `0001-01-01 00:00:00+00`), so a row inserted outside GORM no longer gets NULL there. It has a Down. Rolling back to v0.16.x drops these defaults again, because AutoMigrate removes a default its struct tags do not declare, while the ledger still records version 2: before deploying this release again after such a rollback, run `DELETE FROM goose_db_version WHERE version_id = 2`.

## [0.16.2] - 2026-11-10

### Fixed

- CalDAV: the `/dav` rate limiter updated its per-address counters from concurrent requests without synchronisation, a data race that could admit more than 100 requests a minute or corrupt a counter. One mutex now guards the counters. The limits and the refusal are unchanged: 100 requests per minute per address in a fixed window, then `429` with the `rate_limited` JSON envelope and `Retry-After` counting down to the window's end. It stays a fixed window rather than tronc's token bucket, which would change both the admission pattern and `Retry-After` that CalDAV clients see.

## [0.16.1] - 2026-11-10

### Fixed

- Sign-up: an account created at first sign-in had a NULL `created_at` and NULL text and time columns, because porte/kit inserts only the columns it names and GORM no longer filled the rest. The kit's `OnCreate` hook now writes the sign-up time and the empty values GORM wrote before, in the insert's transaction. Accounts created by v0.16.0 keep their NULLs until repaired by hand.

### Changed

- build: porte v0.18.0 to v0.18.1, which treats `<issuer>` and `<issuer>/` as one provider in the kit's sign-in, erase, backfill and invitations lookups.
- Client: `@facile/client` is excluded from Vite's dependency prebundling, like muse, since both ship uncompiled Svelte source.

## [0.16.0] - 2026-11-10

### Added

- User menu: pending Registre space invitations, with a count pill and a dot on the avatar, from `GET /api/me/invitations` (`porte/kit` `MountInvitations`, mounted with `OIDC_ISSUER`). Any failure answers `{"count":0}` and shows no row.
- Tests: `TestVendoredGoWebdavPatches` pins the five hand-made go-webdav patches CalDAV clients rely on (`cs:getctag`, trailing-slash principal and home set, the root PROPFIND href, PROPPATCH answering 207 with 403). A `go mod vendor` that drops them now fails the build.

### Changed

- Auth: the sign-in glue is `porte/kit` (porte v0.18.0). `internal/authcontext`, `internal/middleware/auth.go`, `internal/oidcavatar`, the unused `internal/authcrypto` and the hand-written user store are gone. Routes and their unauthenticated status codes are unchanged (172 routes walked before and after). The stored email is now rewritten at sign-in only when Registre marks it verified, by owner decision for every suite app.
- Bus: antenne-client v1.0.0 (`ConfigFromEnv`, `catalog`, `envelope`) for the Opus task mirror. `ANTENNE_INSTANCE` is read as the older spelling of `ANTENNE_URL`, and a URL without `ANTENNE_SECRET` now fails boot instead of silently turning the mirror off. Agenda publishes nothing, so it has no outbox.
- Client: `@facile/client` v0.1.0 for the fetch kernel and the theme store. The client always calls `/api` relative; `VITE_API_BASE_URL` and `apps/client/.env.example` are gone, and Vite proxies `/api`, `/files` and `/dav` to `:4000` in development. A 401 on a call other than the session probe now goes to `/login`. The theme renders exactly as before; the stored `agenda.theme` choice is kept.
- Gates: `filet.yml` is `preset: facile` plus Agenda's two exceptions (identical findings), CI runs filet v0.19.0 and douane 0.7.0 (`-fail critical -scope prod`), and `lefthook.yml` replaces `.githooks/` with a pre-push that runs `scripts/check.sh` and `filet gate .`.
- build: tronc v0.19.0 to v0.21.0, porte v0.16.0 to v0.18.0, porte/directory v0.3.0 to v0.4.0, antenne-client to v1.0.0, `@facile/muse` v0.17.1 to v0.18.0, Go 1.26.11 (`go.mod` and the build image, clearing douane's critical stdlib and toolchain findings), go-jose v4.1.6 and x/text v0.41.0.
- Licence: Agenda is now licensed under `AGPL-3.0-only` (see `LICENSE`); before this, the repository carried no licence. v0.16.0 is the first release under it.

### Fixed

- Erasure: a person Registre erases now also loses their pending MCP actions (`mcp_actions`), in the same transaction as the account. The erasure test asserts no row in any table references them afterwards.

## [0.15.1] - 2026-11-10

### Fixed

- MCP: a person's first call to `/mcp` now creates their Agenda account, exactly as a first web sign-in would, so a new Registre user can connect an MCP client without ever opening Agenda on the web. Before, a valid token for a person Agenda had never seen answered `401 invalid_token`. porte/mcpauth v0.2.0 (`Provision: true`) runs the OIDC kit's own login write path; it refuses a token without a verified email. Registre v0.10.3 carries the profile on resource-bound tokens.

### Changed

- build: porte v0.15.1 to v0.16.0 and porte/mcpauth v0.1.0 to v0.2.0.

## [0.15.0] - 2026-11-10

### Added

- Ideas: an "Ideas" entry first in the user menu and a lightbulb button next to it with a badge for new ideas, and an `/ideas` page where Bulle suggests what to schedule, move or prepare. An idea runs in place: Agenda polls the run, shows an approval card for each held call, then the result with its links and a link to continue in Bulle. Ideas are paid (`ai.access`): a free plan sees the upsell and Registre's plans link. The page follows the space of the calendar the views are narrowed to.
- API: `GET /api/ideas`, `POST /api/ideas/generate`, `POST /api/ideas/{id}/run`, `POST /api/ideas/{id}/dismiss`, `GET /api/ideas/runs/{run}` and `POST /api/ideas/runs/{run}/approvals/{call}`, mounted when `OIDC_ISSUER` and the new `BULLE_URL` are set. They call Bulle as the signed-in user through a Registre token exchange (audience `bulle`, scopes `bulle.read` and `bulle.write`). `space_id` must be a space the caller belongs to and that Registre knows. Agenda follows `ai.access` from the directory snapshots, keeps the last known plans in the new `forfait_state` table (created at boot, additive), and answers 402 with the upgrade link, or `ai_access: false` on the list, without calling Bulle when the plan lacks it.

### Changed

- build: porte v0.15.1 (`porte/exchange`) and tronc v0.19.0 (`tronc/bulle`). Client: `@facile/muse` v0.16.1 to v0.17.1, and `bun test` runs with the client tests.

## [0.14.0] - 2026-11-10

### Added

- A remote MCP server at `/mcp` for any MCP client, free for every account (docs/mcp.md). Registre is the authorization server: tokens are bound to `MCP_RESOURCE`, scoped `agenda.read` and `agenda.write`, and verified by `porte/mcpauth` v0.1.0. It runs on `tronc/mcp` v0.19.0 (go-sdk v1.8.0, stateless, spec 2026-07-28).
- Seven tools designed for agents: `agenda_list_calendars`, `agenda_list_events` (range and search, recurring events per occurrence), `agenda_read_event`, `agenda_find_free_slots`, `agenda_create_event`, `agenda_update_event`, `agenda_delete_event`, each annotated, with cursor pages and a 25,000-token cap.
- Deleting an event needs the person's confirmation: MRTR `input_required` when the client can ask, otherwise a pending action and a link to the new Pending actions page (`/pending`, fr/en/es/de). The link survives sign-in.
- The `mcp_actions` table (created at boot, additive).

### Changed

- build(api): porte v0.14.0, porte/mcpauth v0.1.0, tronc v0.19.0, go-sdk v1.8.0.

## [0.13.4] - 2026-11-07

### Changed

- build(client): bump muse to v0.16.1 and replace the deprecated `outline` button variant with `secondary`. Checkbox, Radio and Switch keep their 44px touch target.
- build: `GET /version` is baked at build time from `apps/client/package.json` by `scripts/version.sh`. The `APP_VERSION` build arg is gone from the Dockerfile and compose file, so a stale Dokploy env can no longer show the wrong version. The response is unchanged (`v` plus the version).

## [0.13.3] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.4. Switching the language in the user menu re-renders the whole UI at once, with no reload. The Subscription row no longer passes an empty `plan`.

## [0.13.2] - 2026-11-07

### Security

- Registre tokens are sealed at rest. porte to v0.13.0: `porte_identities.access_token` and `refresh_token`
  are encrypted with AES-256-GCM under `PORTE_TOKEN_KEY`. Existing rows are sealed once, in the background,
  after boot, and the count is logged. An invalid `PORTE_TOKEN_KEY` stops the boot. With the key unset nothing changes.
- Losing `PORTE_TOKEN_KEY` makes the stored tokens unreadable: people sign in again, nothing else is lost.
  Do not roll back to a build older than porte v0.13.0 after the first reseal.

## [0.13.1] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.2. The user menu stays inside the viewport at the right edge of the header.
- The API token drawer no longer asks for an expiry and the list drops the last-used column: Agenda keeps one non-expiring token and does not track its last use.
- The user menu gains a Subscription row that opens the plans on Facile (`https://sso.facile.studio/spaces`).

## [0.13.0] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.1. The avatar in the top bar opens muse's `UserMenu` (Account, Calendar, Spaces, Calendars and members, Settings, Language, Theme, Help, Sign out) in place of the bits-ui dropdown. The language and theme picks are persisted (cookie and `agenda.theme`).
- Settings use muse's `SettingsLayout` in the canonical order: Profile, Preferences, Notifications, CalDAV, Members, API tokens, Advanced. Profile is read-only with "Manage on Facile"; Preferences replaces Appearance; the danger zone ends Advanced; the sign-out button left the settings (it lives in the user menu).
- API tokens use muse's `TokensSection`. The CalDAV connection details and client instructions moved to a CalDAV section. The token API calls are unchanged: Agenda still keeps one non-expiring token per account, so the expiry choice in the create drawer is not enforced.
- The calendar views drop their grid borders for muse fills (month cells, week columns, alternating hour bands).
- The language picked in the menu (or on the login page) now wins over the Registre profile locale on the next load.
- `/settings/appearance` redirects to `/settings/preferences` and `/settings/api` to `/settings/caldav`.

### Removed

- The shadcn-svelte leftovers: `components/ui/`, `components.json`, `UserMenu.svelte`, and the `bits-ui`, `shadcn-svelte`, `vaul-svelte`, `svelte-sonner`, `lucide`, `tailwind-variants`, `tw-animate-css`, `clsx`, `tailwind-merge` and `class-variance-authority` dependencies. Toasts, the calendar filter menu and the spaces buttons use muse.

## [0.12.1] - 2026-11-07

### Changed

- build(api): bump porte to v0.12.1. The GDPR erasure no longer fails on a schema without `porte_magic_links`.
- build(client): bump muse to v0.13.0. Form controls are borderless fills, `Select` is muse's own combobox and badges stay on one line. No Agenda code needed to change.

## [0.12.0] - 2026-11-06

### Changed

- build(api): bump porte to v0.12.0, porte/directory to v0.3.0 and tronc to v0.17.1. tronc's security headers replace Agenda's own middleware: HSTS, nosniff, a report-only CSP carrying the built client's inline script hashes, and enforced anti-framing. `connect-src` also allows Ardoise's plan catalog and the icon API.
- build(client): bump muse to v0.12.1. The public landing page uses `LandingHeader` and `LandingFooter`; every sign-in button is `SignInWithFacile` with its canonical label.
- Sign-in goes through Registre only. The login page is muse's `AuthScreen` and the landing page uses `SignInWithFacile`. A person who already has a local account links to it by verified email on their first Facile sign-in.
- The API token is the CalDAV password for everyone. Account passwords no longer work over Basic Auth (porte refuses them under OIDC; `AllowProtocolPasswords` stays off).
- The whole UI is in fr, en, es and de, in the locale of the person's Registre profile (`GET /api/users/me` returns `locale`). Dates, times, weekday names and recurrence summaries use `Intl`. es and de need native review.
- `OIDC_ISSUER` is required when `APP_ENV=production`.

### Added

- GDPR deletion receiver: Agenda polls Registre's deletion list with its OIDC client credentials and accepts the signed `user.deleted` push on `POST /api/registre/deletions`. A deleted person loses their account, personal calendars with their events, shares, memberships and attendances; events they wrote in space or other people's calendars stay with no author. The receiver's cursor lives in `directory_deletion_state`, created idempotently at boot.
- `users.locale` (`fr`, `en`, `es` or `de`, default `en`), filled from the `locale` claim at each sign-in.
- `porte_login_codes.provider` and `porte_login_codes.subject`, added idempotently at boot by `schemas.AdoptPorte`, as porte v0.10.0 requires for CLI and mobile logins.
- `GET /version`, answering the `APP_VERSION` build arg (`dev` by default).
- `.github/workflows/ci.yml`: gofmt, vet, build and race tests against a Postgres service, client check, tests and build, and filet. It replaces `filet.yml`.
- A test that a first Facile sign-in links an existing account by verified email, that an unverified address never links, and that a recycled address is refused.

### Removed

- `POST /api/auth/register` and `POST /api/auth/login`, the local sign-up and password login page, and the `password` and `current_password` fields of `PATCH /api/users/me`.
- The `SSO_ONLY` setting. It is no longer read.
- Agenda's own `SecurityHeaders` middleware, now provided by tronc.

## [0.11.0] - 2026-11-06

### Added

- feat(client): show the live plans on the landing page with muse's `PricingPlans` (a `LandingOffers` component below the existing content, hidden while the catalog is empty)

### Changed

- build(client): bump muse to v0.11.0; muse's built-in strings are now English by default (v0.10.0)

## [0.10.0] - 2026-11-03

### Changed

- The calendar filter is now one compact dropdown next to the profile picture instead of a row of chips. It shows the selected calendar's color and name, lists "Tous les calendriers" then every calendar with a check on the active one, and stays on the header row on phones.

## [0.9.1] - 2026-11-03

### Fixed

- Phones (checked at 360 and 390px): the event form's time inputs no longer spill past the dialog, month-view events show their title instead of a clipped time, the Espaces header stacks its button under the text, a space's page wraps its buttons under its name, and the members table drops the Type column and shows icon-only actions.

## [0.9.0] - 2026-11-03

### Added

- A filter row next to the profile picture: "Tous" shows every calendar, and each chip narrows the views to one space's calendar or one personal calendar. The choice is remembered per browser, and a space's page links straight to its filtered calendar.
- The event form lists every calendar you can write to, grouped into "Espaces" and "Personnels", and starts on the calendar the view is filtered to.

### Changed

- **Breaking:** `GET /api/calendars` without `space_id` returns every calendar you can open, space calendars included. `space_id=0` returns only calendars outside any space.

### Removed

- The space switcher and the per-browser space context. Spaces now appear only as calendars.

## [0.8.2] - 2026-11-03

### Fixed

- The profile picture sat higher than the space switcher: its menu trigger was an inline box on the text baseline. It is now a 44px flex box centered on the switcher.

## [0.8.1] - 2026-11-03

### Changed

- The profile picture and the space switcher sit at the far left of the calendar header instead of on a bar of their own, and the avatar matches the switcher's height. Other pages show them above their content.
- The month title no longer truncates on laptop widths; the view switcher shows icons only below 1536px, with each label as a tooltip.

## [0.8.0] - 2026-11-03

### Added

- Every space has exactly one calendar, named after the space and shared with all its members: owners and admins administer it, members write to it. Agenda creates it at boot and on each Registre snapshot, and renames it when the space is renamed.
- A "Revenir à aujourd'hui" button appears in the calendar header once you navigate away from the current period.

### Changed

- The sidebar is gone. A top bar carries the profile picture, whose menu opens the calendar, settings, spaces, members and sign-out, with the space switcher beside it.
- Calendars, events and CalDAV resolve access through one shared rule, so a space member reaches the space's calendar from the web app and from CalDAV clients alike.
- The calendar header wraps onto two rows on small screens.

### Removed

- The always-visible "Aujourd'hui" button.
- **Breaking:** `POST /api/calendars` refuses a non-zero `space_id`. A space's calendar cannot be shared, renamed or deleted directly; its members come from Registre. Existing extra calendars in a space become personal calendars of their owner, with their events and shares intact.

## [0.7.0] - 2026-11-03

### Added

- `OIDC_CLI_AUDIENCE` mounts porte's device exchange at `POST /api/auth/oidc/device/exchange`, so suite CLIs sign in with the browser approval link from Registre's device grant, over SSH too. Set it to `facile-cli`; unset, the route stays absent and answers 404. It is independent of `OIDC_MACHINE_AUDIENCE`, which stays this app's own client id.

## [0.6.1] - 2026-11-03

### Changed

- Spaces use the share-circle icon: the Espaces nav item, the space list rows and empty state, and the "Tous les espaces" button. The web app moves to muse v0.8.1.

## [0.6.0] - 2026-11-03

### Changed

- The web app moves to muse v0.8.0 and drops decorative borders: landing and space cards, page headers, the recurrence panel, the API client list and the calendar toolbar are separated by tone and spacing. Calendar grid lines, inputs and buttons keep theirs.

## [0.5.0] - 2026-11-03

### Added

- Spaces follow Registre. With OIDC configured, Agenda imports its existing spaces and members into Registre once at boot, retrying with backoff while Registre is not ready, then pulls Registre's snapshot at boot, every five minutes and on Registre's nudge at `POST /api/registre/sync`. Spaces link by Registre id or by name, members are matched by SSO identity or email, and people Agenda has never seen are pre-provisioned so their first sign-in lands in their spaces. Local password members are never removed.
- `spaces.registre_space_id` (unique, nullable), added by the boot migration, and returned as `registre_space_id`.

### Changed

- The spaces pages are read-only and link to `https://sso.facile.studio/spaces` to create, rename, invite, change roles, remove or leave. Calendar shares are unchanged.

### Removed

- `POST /api/spaces`, `PUT` and `DELETE /api/spaces/{id}`, `POST /api/spaces/{id}/leave`, `POST /api/spaces/{id}/members`, `DELETE /api/spaces/{id}/members/{userID}` and `PUT /api/spaces/{id}/members/{userID}/role`, with the client pages that called them. Registre owns these changes now.

## [0.4.0] - 2026-11-03

### Added

- Every SSO account carries its Registre picture (`<issuer>/avatars/<subject>`) without signing in again: the boot migration fills it from the porte identity rows when the column is empty.
- Event attendees whose address is an Agenda account carry `avatar_url` in responses. Requests may send it back; it is ignored.

### Changed

- muse is bumped to v0.7.3. Space members render with muse `Avatar`, so a missing picture shows the initial instead of a broken image.
- Settings links to the Registre account to change the profile picture.

### Removed

- `POST` and `DELETE /api/users/me/avatar`. The client never called them and they refused every SSO account. Existing uploads stay on disk and still render for password accounts.

## [0.3.0] - 2026-11-02

### Added

- Events carry attendees (email, name, required or optional, response), reminders (minutes before, method), an optional `#RRGGBB` color and a `timezone`, on create, update and read. Attendees and reminders live in the event's ICS, so CalDAV clients see them.
- Single occurrences of a recurring event can be cancelled, overridden and restored: `DELETE`, `PUT` and `POST .../restore` on `/api/events/{id}/occurrences/{when}`. List expansion honours them and reports `occurrence_start`; ICS export carries `EXDATE` and `RECURRENCE-ID`.
- Bearer tokens minted by Registre for Agenda are accepted when `OIDC_MACHINE_AUDIENCE` is set. With `OIDC_SCOPE_PREFIX=agenda`, a token needs `agenda.read` for `GET`, `HEAD` and `OPTIONS` and `agenda.write` for anything else; a token with no scope claim is refused with 403. Cookie sessions are not gated. Both variables default to empty, which keeps today's behaviour.

### Changed

- porte is bumped to v0.7.0.
- An unknown `status` or `timezone` on create or update is now a 400 `invalid_argument` instead of becoming confirmed or UTC.
- An update with an empty `timezone` keeps the stored zone instead of re-anchoring a recurring series to UTC, and an omitted `attendees`, `reminders` or `color` keeps its value.
- An API edit keeps a recurring event's `EXDATE` and `RECURRENCE-ID` data unless the schedule itself changes.
- All-day recurring events expand in UTC, so `EXDATE` matches the right day for every viewer zone.

## [0.2.1] - 2026-10-24

### Fixed

- In the month view, all-day bars and Opus tasks now sit at the same 4px inset as timed events instead of 2px.
- The creator initial is centered in its avatar on event chips.
- Select chevrons no longer touch the right border (muse v0.7.2).

### Changed

- muse bumped from v0.5.0 to v0.7.2: the calendar colour picker is `SwatchPicker`, space role badges use the `fc-owner` and `fc-admin` tokens, and the UI uses the platform sans font now that muse no longer bundles Goga.

## [0.2.0] - 2026-10-24

### Added

- Recurring events: a repeat picker in the event form (daily, weekly on chosen days, monthly, yearly, every N, ending never, on a date or after N times).
- The calendar views show each occurrence of a series in the requested range. A series keeps its wall-clock time across DST changes, and EXDATE and single-occurrence edits made in CalDAV clients are respected.
- Every view shows the creator's avatar on the event, and recurring events carry a repeat icon.

### Changed

- Editing or deleting an occurrence applies to the whole series, and the form says so.
- Recurrence rules are validated on create and update; an invalid rule is rejected with `invalid_argument`.

## [0.1.4] - 2026-10-23

### Changed

- Editing or deleting a mirrored Opus event now says the event comes from an Opus task and must be changed in Opus, over the HTTP API and CalDAV.

### Added

- A contract test feeds Agenda the exact envelope Opus publishes and checks create, replay, stale update, cleared due date and delete.

## [0.1.3] - 2026-10-23

### Fixed

- An Opus due date set on the 27th no longer shows on the 26th. Opus stores local midnight as a UTC instant, so Agenda now reads the date in `DUE_DATE_TIMEZONE` (default `Europe/Paris`).

## [0.1.2] - 2026-10-23

### Fixed

- A task with no assignee is mirrored into the calendar of the user who set its due date (`owner_email`), instead of being skipped silently. Skipped tasks are now logged at info level.

## [0.1.1] - 2026-10-23

### Fixed

- The Opus task mirror only accepts events whose envelope app is Opus. Sablier task events no longer create, change or delete mirrored events.
- Replayed and out-of-order task events are dropped using the event version, so an old update cannot overwrite newer state and a deleted task stays deleted. Events with a numeric legacy id are ignored. `task.status_changed` and `task.actor_changed` without a due date leave the mirrored event alone.
- Mirrored all-day events no longer spill onto the next day in Europe/Paris, and CalDAV `DTEND` is exclusive. A due date with a UTC offset keeps the date as written.
- The personal calendar list only hides calendars of spaces the user belongs to, so calendars of deleted spaces or created with `space_id` 0 stay visible. `space_id` 0 is stored as NULL.
- Deleting a calendar deletes its events, attendees and members instead of orphaning them.
- A panic in a pool event handler is recovered and logged. Stopping the pool client also ends the dial retry loop.

## [0.1.0] - 2026-10-23

### Added

- Agenda subscribes to the Antenne Pool and mirrors Opus tasks that have a due date into a read-only "Opus tasks" calendar, as all-day events. Enabled when `ANTENNE_URL` and `ANTENNE_SECRET` are set.
- Events carry `source` and `external_id`. Mirrored events reject edits and deletes over the API and CalDAV.

### Fixed

- Switching to a space with no calendars no longer leaves the previous space's events on screen.
- The personal context lists only personal calendars, so switching spaces visibly changes the view.
