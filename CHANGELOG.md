# Changelog

All notable changes to Agenda are documented here.

## [Unreleased]

## [0.1.0] - 2026-10-23

### Added

- Agenda subscribes to the Antenne Pool and mirrors Opus tasks that have a due date into a read-only "Opus tasks" calendar, as all-day events. Enabled when `ANTENNE_URL` and `ANTENNE_SECRET` are set.
- Events carry `source` and `external_id`. Mirrored events reject edits and deletes over the API and CalDAV.

### Fixed

- Switching to a space with no calendars no longer leaves the previous space's events on screen.
- The personal context lists only personal calendars, so switching spaces visibly changes the view.
