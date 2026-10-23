# Changelog

All notable changes to Agenda are documented here.

## [Unreleased]

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
