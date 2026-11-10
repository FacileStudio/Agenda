# MCP server

Agenda serves a remote MCP server at `POST /mcp` for any MCP client: Claude (web, Desktop,
Code), ChatGPT, Cursor, VS Code. It is free for every account. It runs on
`github.com/FacileStudio/tronc/mcp` v0.19.0 (go-sdk v1.8.0, stateless, spec 2026-07-28), in
`apps/api/modules/mcp`.

## Connecting

Add `https://agenda.facile.studio/mcp` as a remote MCP server. The client finds Registre
through `/.well-known/oauth-protected-resource/mcp`, signs the person in and asks consent for
`agenda.read` and `agenda.write`. Tokens are bound to that URL and verified by
`porte/mcpauth`. A token with only `agenda.read` gets a 403 `insufficient_scope` challenge on a
write tool, so the client can ask for more.

## Tools

| Tool | What it does | Hints |
|---|---|---|
| `agenda_list_calendars` | Calendars with id, name, color and role | read-only |
| `agenda_list_events` | Events between `from` and `to`, recurring ones per occurrence; `query` searches title, description and location | read-only, open world |
| `agenda_read_event` | One event in full: description, attendees, reminders, recurrence | read-only, open world |
| `agenda_find_free_slots` | Free windows of at least `duration_minutes` inside working hours, around every calendar's events | read-only |
| `agenda_create_event` | Add an event, to the personal calendar unless `calendar_id` is set | write |
| `agenda_update_event` | Change only the fields given; a new start keeps the length | write, idempotent |
| `agenda_delete_event` | Delete an event for good; always confirmed by the person | destructive |

Times are RFC 3339 or dates (all-day events and range bounds). Pass the person's IANA
`timezone`; the default is UTC. `agenda_list_events` pages with `cursor` and `next_cursor` and
takes `response_format` (`concise` or `detailed`). A result stays under about 25,000 tokens.
Errors are `isError` results naming the tool or argument to fix. Event text written by other
people is data for the model, never instructions, so event reads are marked open world.

## Confirmations

`agenda_delete_event` never runs on the model's word alone. It refuses at once an event from
Opus or one in a calendar the person can only read. Otherwise:

1. It stores a pending action in `mcp_actions` with a digest of the event (its ETag, update
   time, title, times, rule and calendar).
2. A 2026-07-28 client that declares form elicitation gets an `input_required` question. The
   retry carries the answer and a `requestState` signed with a key derived from
   `ENCRYPTION_KEY`, naming the action, the person and a 15-minute expiry.
3. Any other client gets `status: pending_confirmation` and a `confirm_url`,
   `https://agenda.facile.studio/pending?action=<id>`, open for 7 days. The person confirms or
   dismisses it on the Pending actions page. A signed-out person signs in and lands back on it.

Either way the action runs once. An event edited after the question is not deleted and the
action closes. A delegated (Bulle) token cannot use the confirm page. Every write call and
every confirmation event is logged (no arguments) with the client id.

## Testing

`go test ./modules/mcp/` with `TEST_DATABASE_URL` runs tronc's `mcptest` kit
(`TestConformance`) and the tool tests. `MCP_SERVE=127.0.0.1:<port>` with
`-run TestServe` serves the endpoint over the scratch database for MCP Inspector or the
official suite (`npx @modelcontextprotocol/conformance server --url ...`, Node 22).
