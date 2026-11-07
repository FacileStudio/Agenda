# Changelog

All notable changes to Agenda are documented here.

## [Unreleased]

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
