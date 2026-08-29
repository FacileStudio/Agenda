# Changelog

Decisions are recorded with their reasoning. The reasoning is the part that stops a future
session from undoing a deliberate choice.

## v0.4.0 — 2026-08-25

**One CLI login for the suite: `POST /auth/oidc/device/exchange`.** A CLI trades the access
token it holds from the provider's device grant (RFC 8628) for this app's own session token.
`{"access_token": …}` goes in, `{"user_id", "token"}` comes out. It is the last missing piece of
`facile login`, which until now ran the device grant against Registre, held a valid token, and
had nowhere to spend it: every tool answered 404, so the CLI fell back to the loopback browser
flow, the one flow that cannot work when the browser is on another machine. Writing the
provider's token into the slot where a CLI keeps its own session was never an option; that is a
login that 401s when the token expires an hour later.

The handler adds no verification of its own. It composes the bearer verifier the Authorization
header path already uses, the same `(issuer, sub)` lookup the login callback matches on, and the
same `Manager.Issue` that mints every other session. All that is new is that the token arrives in
a request body rather than a header, so `Cache-Control: no-store` applies (OAuth 2.1 §7.1) and no
CSRF header is required, because there is no ambient credential to abuse when the caller has to
put a token it holds into the body.

**Two audiences, because they are two token populations.** The exchange has its own setting,
`Config.CLIAudience` (env `OIDC_CLI_AUDIENCE`), which an app sets to the CLI's client id,
`facile-cli` for the suite. `OIDC_MACHINE_AUDIENCE` keeps its old meaning and its old value:
this app's own client id, which is what a service account's token is addressed to. Registre's
`suite-ci` account declares `audiences: [courrier]`, so courrier checks `courrier` there and the
CLI's `aud: ["facile-cli"]` never matches it. Folding the two into one variable would have made
an app choose between service accounts and CLI login, and an app that chose CLI login would have
started rejecting every service-account token it had been accepting, silently, with nothing in
its own configuration having changed meaning.

**The second audience builds a second verifier, and it never reaches the session manager.**
`OIDC_CLI_AUDIENCE` arms this route alone; it does not put a `facile-cli` token on the
`Authorization` header path. So the exchange is a real boundary rather than a formality: the
token traded in cannot already open every `RequireAuth` route on its own, and what comes back is
a session row the app can list, expire and revoke, where a verified bearer JWT leaves no row at
all. The cost is one extra discovery and JWKS fetch at boot, paid once per process, because
`porte/oidc/jwt` bakes the audience in at construction.

**An empty `OIDC_CLI_AUDIENCE` does not mount the route, and its 404 is load-bearing.** An app
with no CLI audience cannot tell a Registre token from a forgery and must not pretend to serve
the exchange; 404 is exactly what `facile login` reads as "not shipped" before falling back. The
corollary binds an app that does mount it: the CLI probes with a POST carrying an empty body, so
a mounted route refuses on the merits (400 or 401) and never with a 404. `OIDC_MACHINE_AUDIENCE`
does not mount it, so an app can run service accounts with no exchange, the exchange with no
service accounts, or both, and the two never interfere.

Refusals are one status and one message. Bad signature, wrong issuer, wrong audience, expired,
not yet valid, unknown kid, no subject, and a subject with no row in `porte_identities` are
indistinguishable to the caller, and a token addressed to the machine audience is refused here
exactly like any other wrong audience. The identity store is consulted only after every
cryptographic and claim check has passed, so refusal latency does not answer "does this subject
have an account here" either. No account is created here: provisioning from a bearer would let
anyone the provider will mint a token for materialise an account in every app at once, and the
callback owns account creation because it is the path holding a verified email.

**What the split does not close, stated plainly:** a stolen or phished `facile-cli` token can
still be exchanged for a durable session at every app that trusts it. `facile-cli` is a public
client, so anyone can start its device grant; the human approving the code and the requirement
of an existing local account are what stand in the way. Bounding the rest is back-channel
logout's job, and `backchannel_logout_uri` appears zero times in Registre's `seed.yaml` today,
so that bound is not in place yet.

**A verified bearer whose identity row names account zero is now refused** on both paths. No
account has that id, so the row is a store bug, and both callers spend the `UserID` directly,
one to authorize a request and one to mint a session. Zero would be the identity of everybody at
once.

**Offline verification of bearer JWTs.** `Config.MachineAudience` (env
`OIDC_MACHINE_AUDIENCE`) gives the session manager a second bearer verifier: a bearer that parses
as three dot-separated segments is verified against the provider's JWKS — signature, `iss`, `aud`,
`exp`, and `nbf`/`iat` when present — and never reaches the hashed-session lookup. Keys cache for
an hour and refetch once on an unknown kid, so a rotation does not wait out the TTL. The engine is
the new stdlib-only `porte/oidc/jwt`; the manager learns of it through the `session.JWTVerifier`
interface so the credential package still compiles without any OIDC dependency. Introspection
stays out: offline verification is the point.

**A verified bearer resolves to a local account.** This is what turns one login per app into one
login for the suite: a user who has signed in here through the browser can afterwards present a
Registre-issued token to the same app, and porte authenticates them as themselves. `sub` is
matched against `porte_identities` on `(issuer, subject)` — the same key the callback and
back-channel logout already use, and never the email address, because matching on a mutable
address is the account-takeover primitive v0.3.0 removed from the callback. A subject with no row
is refused rather than provisioned: account creation stays in the callback, which is the path
that holds a verified email.

That lookup is also the whole of the deactivation story on this path, and the reason it is
mandatory. A JWT carries no session row, so revoking sessions — all back-channel logout can do —
does not reach one, and `SessionID` is zero. What does reach one is `IdentityStore.Find`: an app
that deactivates an account by making `Find` answer `ErrNotFound` locks it out on the next
request. An app that leaves the row readable keeps admitting the token until it expires, so the
issuer's access-token lifetime is the real bound. Registre SPEC §10 question 6 chose offline
verification knowing this and named short lifetimes as the mitigation.

The behaviour change lands on an unreleased, unadopted surface: `MachineAudience` was empty in
every app, and a service account that previously authenticated as an all-but-empty identity now
needs a row like anybody else. Registre's SPEC asks for exactly that — "the app-side identity is
`(issuer, sub)` exactly as it is for a human, a service account is a principal, not a special
case". Roles are taken from the verified token rather than from the cached row, so a provider
that emits no roles claim leaves a bearer caller with none rather than with yesterday's.

**The unknown-kid refetch is rate limited.** The previous entry claimed forged kids could not
cause a fetch storm and the code did not deliver it: `kid` is read off the header before any
signature is checked, and a fresh one on every request bought one outbound JWKS fetch each,
serialized under the mutex every real verification waits on and aimed at the one provider all
eleven apps share. It is now one refetch per `minRefetchInterval` (30s). A rotation the provider
published before signing with the new key is already cached and unaffected; the floor costs at
most half a minute of refusals on a rotation that skipped that step.

## v0.5.0 — 2026-08-28

**`porte/spaces`: the space-membership guard seven apps wrote separately, and three wrote wrong.**
Sablier, Courrier, Agenda, Plume, Nuage, Vision and Antenne each own a spaces module, each with its
own answer to "is this caller allowed to act in this space". `modules/spaces/types.go` is
54/49/40/39/45 lines across five of them with five different hashes. The module is additive: an app
that has no spaces never imports it, the way `tronc/migrate` and `caisse/pg` are optional.

It numbers **v0.5.0 and not v0.4.0** — SPEC §4 planned it as v0.4 and the OIDC device exchange took
that tag first.

**Four rules, and each one is a bug somebody shipped.**

- **Membership is the only key.** No instance-admin bypass, no superuser flag, and no hook to add
  one. Nuage carries `users.is_admin` one package away from its space guard and, to its credit, does
  not consult it there; the point of putting the rule in a library is that the day somebody wants to,
  the answer is a membership row, which is listed in the member screen and revocable. A bypass inside
  `porte` would be invisible to every app that imported it.
- **A space id is checked before it is usable.** An empty id is personal scope in `Resolve` and
  never touches the store; every non-empty id goes through `Store.Membership`. A `Scope` carrying a
  space id is therefore proof of membership in it, and there is no constructor that makes one
  otherwise: the struct holds an unexported marker that only `Guard` sets, so
  `Scope{UserID: "mallory", SpaceID: "victim", Role: RoleOwner}` still compiles — every adopter
  builds these in middleware — and reports `Resolved() == false`. A refusal returns the zero
  `Scope`, and the zero `Scope` reports neither resolved nor personal, so a caller that ignores the
  error gets nothing usable. That last clause used to be prose rather than behaviour:
  `Scope{}.Personal()` was `true`, so ignoring an error and branching on it ran the personal-data
  path for user `""`.
- **A space always has a reachable owner.** Sablier, Nuage and Agenda count the owners before letting
  one leave. Courrier and Plume refuse *every* owner outright, which is not the same rule: it makes
  ownership transfer the only exit from a space two people own equally. `CanLeave` returns
  `ErrSoleOwner` on the count, not on the rank.
- **No privilege escalation.** Agenda's `AddMember` checks that the actor is owner or admin, then
  passes the requested role through `normalizeRole`, which accepts `owner`
  (`modules/spaces/service.go:155`). An admin can mint an owner and be promoted by it.
  `AssignableBy(actor Scope, target Role)` is false whenever target outranks actor; granting one's
  own rank stays allowed, because appointing a peer admin is what every member screen already does.
  It takes the resolved `Scope` and not a second `Role` string, because two plain roles invite
  passing both straight off the wire, which checks the request against itself. The caller already
  holds the `Scope` that `Require` returned, so it costs nothing; an unresolved or personal one
  grants nothing.

**`Require` refuses an empty space id.** Passing every minimum on personal scope is fail-open on
empty input. The realistic exploit is a gate and a use reading the id from different places:
`Require(ctx, uid, r.Header.Get("X-Space"), RoleAdmin)` returns nil when the header is absent, and
the handler then acts on a space id taken from the body. Personal scope comes from `Resolve`, which
is explicit about it; a handler serving both shapes calls `Resolve` and branches on
`Scope.Personal`.

**`Resolve` requires the row to carry both ids, and both to match.** Skipping an id the store left
empty reads as caution and is the opposite: `SELECT role FROM ... WHERE space_id=$1 AND user_id=$2`
is the most natural `Store` an app writes, it returns `Membership{Role: "owner"}` with two blank
ids, and treating blank as agreement would disarm the whole defence for exactly that app. A row
missing an id is `ErrNotMember`. `Spaces` applies the same identity rule per row rather than
filtering on ladder validity alone, because a store's bad join would otherwise reach the caller with
a nil error, and a space switcher renders precisely that list.

**An explicitly empty `Ladder` is not `Default()`.** `NewLadder` marks what it builds, so `Guard`
substitutes the suite's three roles only for the unset ladder. An app assembling its vocabulary from
configuration and getting it wrong refuses every role rather than silently inheriting
owner/admin/member. `Ladder.Configured()` reports which one it is holding.

**`CanLeave` is time-of-check to time-of-use, and says so.** It counts the owners, the caller
deletes, and two owners leaving at the same instant both count two and both pass. The package cannot
close that without the database dependency it exists to refuse, so the contract sits on the caller
and in the godoc: run `CanLeave` and the `DELETE` in one transaction with the space's membership
rows locked — `SELECT ... FOR UPDATE` over the rows `CountRole` counts, or a serializable
transaction with a retry. Sablier, Agenda and Plume ship the unlocked count-then-delete today, so
importing the package without the lock reproduces their bug with a certificate attached.

**The role ladder is configurable, against the plan's fixed three.** Vision gates every write on
`owner|admin|editor` with `viewer` below (`internal/siteaccess/siteaccess.go:29`). A package
hard-coding owner/admin/member would have left Vision holding its own copy of the guard, which is one
more copy from the package built to end them. So `Ladder` is a list ordered by privilege, `Default()`
is the suite's three, and every check is a comparison inside a ladder rather than a switch over
names.

**A role the ladder does not rank is unknown, not weak.** Plume's `normalizeRole` returns
`RoleMember` for anything it does not recognise, so a corrupt or renamed value in the column becomes
a valid low role instead of a refusal. Here `Ladder.AtLeast` is false when *either* side is unranked,
`Resolve` returns `ErrUnknownRole` rather than a `Scope` when the store hands back a role it cannot
rank, and `Require` refuses a minimum the caller invented before it ever reads the table. A typo
closes a door.

**No models, and that went further than the plan.** SPEC §4 promised `Space`/`SpaceMember` structs
alongside the guard. Those structs are where the copies diverge hardest — `ID` is a string in
Courrier and an `int64` in Nuage — so shipping them would have made adoption a migration in six
databases before the guard could run in any of them. The app keeps its table and implements a
three-method `Store` over it, converting at the boundary; ids cross as strings. Standard library
only: no GORM, no chi, no models. The package every app's authorization depends on brings nothing
into the binary.

**`spaces/spacestest.Conformance(t, newStore)` is how an adopter inherits the proof and not only the
code.** It seeds a fixture — one space with a sole owner, one with two — through a `Seeder` the store
implements, then runs the invariants against the app's *own* table. Copying the guard was never the
hard part; knowing that the query underneath it still answers honestly after somebody edits a
`WHERE` clause is. The guard defends itself against that too: `Resolve` builds the `Scope` from its
arguments rather than from the returned row, and refuses a row whose ids disagree with what was
asked, so a wrong `WHERE` is a failed lookup and not a membership in somebody else's space.

**The suite asserts what a store returns, not how much of it.** Counting rows certifies the two
stores that matter most: one that blanks the ids on every row, which is the shape that disarms
`Resolve`'s cross-check, and one whose `Memberships` promotes every row to owner. Both were written
as probes, both passed the counting suite, and both fail the current one. So `Conformance` now
checks that ids come back populated from the row and matching the arguments, that roles survive the
round trip, and that `Memberships` lists the caller's own rows and no others.

**`ConformanceWithLadder(t, newStore, ladder)` runs the suite on an app's own vocabulary.** The
suite always built `Default()`, so Vision — the app that forced `Ladder` to be configurable — could
not put its own `owner|admin|editor|viewer` through it, and a conformance run that proves the guard
on a vocabulary the app does not use proves nothing about the guard the app ships. The fixture takes
its three ranks from the ladder's top, second and bottom, so the ladder must rank at least three
roles; the suite needs a top, a middle and a bottom to tell a refusal from an escalation.

**One argument order was reversed against the spec.** `CanLeave(ctx, userID, spaceID)` matches
`Resolve` and `Require` instead of taking the space first. Both parameters are strings, nothing would
catch a swap, and the two calls sit in the same handler.

## v0.3.1 — 2026-08-22

**`SPEC.md` calls the event bus Antenne.** Two forward-looking passages still named Nook: §4's
`porte/espace` scope, where `FacileID` is the key a space syncs on across apps, and §5c's third
freshness mechanism, the group-change event that invalidates cached claims. Same bus, renamed.
Neither plan changed, and neither is built yet.

It is recorded here rather than pushed silently because `SPEC.md` is the contract an adopter
reads before building against porte, and Agenda, Courrier and Sablier vendor it — Go vendoring
carries a module's markdown too. Their copies keep the old name until each re-vendors, which is
the place to fix them; do not hand-edit a file under `vendor/`.

No Go code changed in this range. The filet configuration and its CI workflow landed here too and
are deliberately unlisted: they gate this repository and ship nothing an adopter consumes.

## v0.3.0 — 2026-08-10

**A password identity is keyed on the account id now, not on the email address.** This is the
version's whole point; everything else in it follows from the same mistake.

`porte` already knew the rule. SPEC §3 has said since 2026-08-07 that OIDC account matching keys
on `(provider, subject)` and **never on email**, because the address is mutable — an email change
silently orphans the account. That rule was applied to federated identities and broken for
`porte`'s own local ones, where `subject` was the normalised address. OpenID Connect Core §5.7 is
explicit: *"other Claims such as `email`, `phone_number`, `preferred_username`, and `name` MUST NOT
be used as unique identifiers for the End-User."* Every mature implementation agrees — Keycloak's
`credential` table keys on `user_id`, Supabase sets `identities.provider_id` to the user's uuid for
the email provider, better-auth sets `account.accountId` equal to `userId` for credential accounts,
Auth0 documents `user_id` as "unique and immutable".

What the mutable key cost, measured across the eight adopters rather than imagined:

- **Five apps wrote the same `UPDATE porte_identities SET subject = ?` by hand**, because the
  contract offered no way to re-key — there is no delete and no update on `IdentityStore`. Copying
  a raw statement against another package's table into five repos is the symptom; the missing
  operation was the disease.
- **Sablier did not write it.** Changing an address there moved `users.email` and left the
  credential behind, so the old address kept signing in and the new one never did. Worse together
  with a password change: `Save` upserts on `(provider, subject)`, so setting a password on the new
  address **inserted a second identity** and left the first intact with its old hash — two working
  passwords on one account, one of them on an address the human no longer owned.

Keying on the id deletes the failure class instead of defending against it. Changing an address is
now one `UPDATE` on the app's own user row, and `subject` being half the primary key means "one
password per account" is enforced by the table rather than by a check somebody has to remember.

**Migration.** `pg.Schema` carries `UPDATE porte_identities SET subject = user_id::text WHERE
provider = 'local' AND subject <> user_id::text`. It is idempotent, it leaves federated subjects
alone, and it is allowed to fail: the only way it can is an account holding two password
identities, which the old key made reachable and which nothing should paper over by picking one.
Refusing to migrate is the right answer to ambiguous credentials. Across the fleet on 2026-08-10
the audit found **four local identities in total** — Journal 1, Boutique 3, and zero in the other
six — which is why this landed now rather than later. It will never be cheaper.

### `ChangePassword`, and why `SetPassword` refuses

**Four of eight adopters shipped a settings screen that set a new password without asking for the
old one.** Sablier, Courrier, Agenda and Boutique took `PATCH /users/me {"password": …}` and passed
it straight to `SetPassword`. OWASP ASVS puts the confirmation at L1 (v4 §2.1.6, v5 §6.2.3): *"password
change functionality requires the user's current and new password."* One method served both "add a
first password" and "replace an existing one", so the check was the app's to remember, and half of
them did not.

So the two operations are two methods now. `SetPassword(ctx, userID, password)` gives a first
password to an account that has none and returns `ErrPasswordSet` otherwise — it takes no address
any more, because there is no longer an address in the key. `ChangePassword(ctx, w, r, userID,
current, next)` is the replace path, and it cannot be called without the current password.

`ChangePassword` also ends the account's **other** logins, rotates the caller's own session, and
leaves named API tokens alone. Each third of that has a reason, and they are not the same reason:

- **Other logins go.** ASVS asks an application to *offer* termination of all other sessions after
  a password change (v4 §3.3.3, v5 §7.4.3, L2). Doing it rather than offering it is stronger than
  the control and matches Google and Entra.
- **The caller is rotated, not dropped.** No ASVS control covers this; the OWASP Session Management
  Cheat Sheet does, naming password changes specifically and requiring the previous id to be
  destroyed. The old token is dead before the call returns and the new one is already in the
  cookie, so the screen that made the change keeps working. No vendor documents logging you out of
  the browser you just used, and all eight say "all **other** sessions".
- **Named API tokens survive**, and this one is porte's decision rather than a standard's — ASVS,
  NIST SP 800-63B-4 and the cheat sheets are all silent on long-lived credentials here. The
  industry is not: GitHub's exhaustive list of revocation triggers omits password change, AWS
  states access keys keep working through an expired password, Entra exempts app passwords,
  Stripe's keys are account-scoped. Revoking them would put `porte` outside all eight products
  surveyed, and would mean a routine rotation silently stops the CalDAV client on somebody's phone.
  `RevokeUser` remains the call for a leak, where the answer really is everything.

### Breaking

- `porte.IdentityStore` gains nothing, but **the meaning of `Subject` changed** for
  `provider = 'local'`: it is `porte.LocalSubject(userID)`, never an address.
- `porte.SessionStore` gains `DeleteLogins(ctx, userID, except)`. Any custom implementation must
  add it; no adopter has one — all eight use `porte/pg`.
- `local.Kit.SetPassword` loses its `email` parameter and now refuses an account that already has a
  password.
- New sentinels `porte.ErrNoPassword` and `porte.ErrPasswordSet`.

**Adopting apps must delete their own re-keying SQL.** Left in place it is harmless — it updates
zero rows, since no `subject` matches an address any more — but it is a statement against a table
`porte` owns, describing a design that no longer exists.

## v0.2.10 — 2026-08-10

Two things the Plume adoption found, both in the same corner: what happens to a session over time.

**A named API token is issued without an expiry.** The rest of porte already assumed this — the
store's sweeper documents *"rows with no expiry are API tokens and are never swept"*, the idle
window does not apply to the bearer transport, and every adoption migrated its old `api_tokens`
table across with a null `expires_at`. `Issue` was the one place that disagreed, stamping the
30-day session lifetime onto labelled rows too. The effect was invisible and dated: a token minted
through an app's UI died a month later, while a token migrated from the same app's old table lived
forever, and nobody finds out until a nightly job stops. An app that wants named tokens to expire
owns that policy; it holds the label.

**`Manager.Sweep` deletes what has expired.** `SessionStore.DeleteExpired` has existed since v0.1
but only on the store, so an app running an hourly cleanup had to keep a second reference to
`porte/pg` purely to expire rows. It is on the manager now for the same reason `List` and `Revoke`
are: an app that holds a Manager should not also have to hold the store, because the whole point
of the manager is that one thing owns the credential.

**Upgrade note for Courrier and Agenda:** both mint named tokens through `Issue`, so any token
created there between their adoption and this release carries a 30-day expiry. There are none in
production yet — both were adopted today — but a fork that has been running longer should clear
the expiry on its labelled rows:
`UPDATE porte_sessions SET expires_at = NULL WHERE label <> ''`.

## v0.2.9 — 2026-08-10

**The CLI loopback redirect can now carry the caller's nonce.** `/auth/oidc?flow=cli&port=N`
accepts an optional `cli_state`, keeps it in the flow cookie, and echoes it back as `state`
alongside `code` on the `127.0.0.1` redirect.

Without it a CLI's listener has no way to tell its own callback from one somebody else sent. Any
local process — or any web page that can reach `127.0.0.1:<port>` — can race an attacker-chosen
code into the listener before the real one lands, and the CLI will exchange it and store the
resulting token. `sablier-cli` accepts any callback bearing a `code` today; `casier-cli`, which
does not use porte, has validated a nonce since it was written. This closes the gap for every
porte adopter at once.

**The parameter is optional, and that is deliberate, not laziness.** A CLI that predates this
release sends no nonce and gets a redirect identical to today's. Requiring it server-side would
hard-abort every installed binary the moment the server deployed — the lockout is a worse outcome
than the race it prevents, and one the client can close on its own schedule. A CLI that *sends*
the nonce must *require* it back; that half is the client's, and it may only ship after this
release is live.

`cli_state` is bounded at 128 characters and restricted to `[A-Za-z0-9-_]`. The value is opaque to
the server and reflected into a redirect, so the only question worth asking is whether it can stop
being a nonce and start being a second query parameter or a header.

## v0.2.8 — 2026-08-10

**Logging out works when the session is already dead.** `POST /auth/logout` moves from
`RequireAuth` to `Optional`: it clears the cookie and answers `{"logged_out":true}` whether or not
the credential was still valid.

Behind `RequireAuth` an expired or revoked cookie got a `401` and was **not cleared**, so the one
state that makes somebody press the button — a session the server has stopped honouring, in a
browser that still holds it — was the state where the button did nothing. The stale cookie kept
being sent and the user had no way out short of clearing site data. Found adopting Courrier and
Agenda, whose own logout had always answered `ok` regardless; this restores that.

Nothing was protected by demanding auth: ending a session you already hold is not privileged, and
for a caller without one this only expires a cookie in their own browser.

**The CSRF header is still required from anything presenting a cookie**, and the check is now the
handler's own, because `Optional` swallows that refusal along with every other error. Without it a
cross-site `POST` would log the victim out — a nuisance rather than an escalation, but a
protection porte already had and must not drop while widening the route. A bearer-only caller
sends no cookie and needs no header: there is no ambient credential for a forged request to ride.

## v0.2.7 — 2026-08-10

`local.Kit.Verify` — a password check that issues nothing. `Login` is now this plus a session.

Found adopting Agenda, which serves CalDAV over Basic auth: a client re-sends its credentials on
every PROPFIND, so a handler reaching for `Login` would mint a session row per request and attach
a `Set-Cookie` to a response no browser will ever read. The alternative was the app verifying
against `porte_identities.password_hash` itself, which is the argon2 parameters copy-pasted back
out of the library they were extracted into.

It keeps `Login`'s enumeration guarantees, because they are the part that gets dropped on
reimplementation: an unknown address still costs a real hash and still returns the error a wrong
password returns.

## v0.2.6 — 2026-08-10

**A token that says nothing about the address no longer vouches for it.** New optional
`Config.TrustEmailWithoutVerifiedClaim` restores the old behaviour for a provider that omits the
claim on purpose.

`emailClaimTrusted` treated an absent `email_verified` as trusted, with the reasoning that the
provider "is asserting nothing either way". That is exactly right and it is the argument for the
opposite conclusion: porte cannot tell a provider that omits a claim it checks anyway from one
where any visitor can register any address, and `Claims.EmailVerified` decides whether a callback
may **adopt an existing account by email**. So the permissive default handed the strongest thing
in the flow — matching a stranger's row — to the one case porte knows least about. It is the
`v0.2.3` takeover with a different door: an attacker registers the victim's address at a provider
that does not verify addresses, signs in, and lands in the victim's account.

The claim now has three states rather than two. An explicit `true` verifies, an explicit `false`
refuses, and **anything else is silence** — absent, a string nobody can parse, a number. Silence
resolves to `TrustEmailWithoutVerifiedClaim`, which is off. `"maybe"` and `42` used to resolve to
trusted and refused respectively, which was two different answers to the same question.

**The flag cannot overrule an explicit `false`.** That one is the provider answering, and an
operator turning a no into a yes is the guard being disabled by checkbox. There is a test for it,
because the useful part of a security flag is what it refuses to do.

**What this changes for the suite: nothing, and that is checkable.** Authentik emits
`email_verified` on every token — as `false` for addresses it never verified, which already
refused the fallback — so no Facile app is on the absent path. Adopters migrating existing users
do it by backfilling `porte_identities` (`adoptOIDCSubjects` in Sablier, `adoptExistingPasswords`
in Journal), which is the supported route and does not touch this. The five Go apps that have not
adopted porte still carry their own `emailClaimTrusted` with the old absent-is-trusted rule; that
is now a divergence from the library and a thing to fix when each adopts.

`porte` keeps signing these users in — the claim never was a login gate, and gating login on it
locks out every Authentik account, which is a bug the suite already shipped once. It gates one
thing: whether an incoming subject may take over a row it has never authenticated as.

## v0.2.5 — 2026-08-10

`porte_identities` gains `created_at`. No API change, no migration to write by hand: it is in
`Schema`, so an app that already applies `pg.Schema` (or its own copy of it, which is what an app
keeping its own `users` table does) picks it up on the next boot.

The column exists because of the account takeover fixed in `v0.2.3`. The audit that fix asks for —
*which* accounts had a `local` identity grafted onto them while the hole was open, and when — could
not be run: the table recorded that an identity existed and nothing about when it appeared. It
records that now, and it is the first write that stamps it. Neither upsert carries `created_at` in
its `ON CONFLICT` list, so a login never moves the stamp; a column that recorded the last sign-in
would answer a different question while looking like an answer to this one.

**Rows that predate the column stay NULL, deliberately.** The obvious `ADD COLUMN … NOT NULL
DEFAULT now()` backfills every identity in production with the timestamp of the deploy, which is
not late, not approximate, and not conservative — it is wrong, and it is wrong in the exact rows an
audit is about to read. So the upgrade adds the column nullable and sets the default afterwards:
new rows are stamped, old rows say "predates the column", and NULL is the honest answer to a
question the database was never asked. A fresh install has the same nullable column rather than a
stricter one, because two shapes of the same table is how a query that works in one deployment
fails in another.

`Boutique` added this column itself in its `00005` migration as `NOT NULL`; the statements here are
`IF NOT EXISTS` and a `SET DEFAULT`, so they are no-ops against it and it keeps the stricter column.

## v0.2.4 — 2026-08-09

A failed login is a page now, not a JSON body. New optional `Config.FailureURL`; no breaking
change, and an app that sets nothing gets the better behaviour.

`/auth/oidc` and `/auth/oidc/callback` are browser navigations — the user *is* the HTTP client.
Every failure in them called `httpjson.WriteError`, so an expired flow cookie or a provider that
declined put `{"code":"invalid_argument","message":"..."}` in the address bar. Plume shipped the
identical bug to production this morning and the report that came back was, in full, "le sso est
cassé ça m'envoie ça en json wtf". That is the correct reaction.

Both handlers now redirect to `Config.LoginFailure(reason)` — `FailureURL` when set, otherwise
`/login` on `SuccessURL`'s **origin** — with the reason in an `error` query parameter for the
login page to render. The default replaces `SuccessURL`'s path rather than appending to it: an
app landing on `/dashboard` has its login page at `/login`, not `/dashboard/login`. The CLI's own
endpoints (`/auth/cli/exchange`, back-channel logout, the middleware) keep writing JSON, because
there the caller really is a program.

Only failures porte classified as the caller's problem say what they were —
`invalid_argument`, `already_exists`, `unauthenticated`, `permission_denied`. An internal error
redirects with "could not sign you in" and puts the detail in the log, where it was already
going: a redirect target is a URL a user can screenshot into a support channel, so it is no
place for a database error.

**The cost, stated plainly:** a refused callback and a successful one are now both a `302`, and
`porte` no longer distinguishes them by status code. Anything watching these endpoints must look
at `Location`, not the status. Four tests in `flow_test.go` asserted on status alone — including
the roles-claim guard, whose entire purpose is catching a silent sign-in — and all four would
have passed against a build that redirected failures to `SuccessURL`. They now assert the two
things that actually differ: the browser lands on the login page carrying a reason, and no
session cookie comes back. That second half is the one that would catch a regression that signs
the user in anyway.

## v0.2.3 — 2026-08-09

**Security. Upgrade if you serve `POST /auth/register` to the public — Journal and Sablier both
do, and both were exposed.**

`local.Kit.Register` treated an address that already had an account *without* a password as "the
same human adding a password". It hashed the caller's password onto that account, saved it as
that account's `local` identity, and issued a session for it. So anyone who knew the address of a
user who had only ever signed in through SSO — or who was migrated in from a legacy table whose
hash did not come across — could register that address and be logged in as them. Registration
being open is the whole precondition, and open registration is what the flag is for.

The comment defending it was not wrong about the intent, only about who was asking: porte has no
mailer and therefore cannot tell "the same human" from a stranger who typed a known address.
Registration now refuses any address that already has an account, with `ErrEmailTaken`, whether
or not that account has a password.

Adding a password to an account that already exists is `Kit.SetPassword`, which takes a user id
and so can only be called for a caller the app has already authenticated. That is the supported
path and it always was; it is now the only one.

The refusal also runs the timing equaliser. It previously returned before hashing, so the
response time alone said whether an address was registered — a slower oracle than the status
code, but one that survives every attempt to hide the status code. Equalisation is not perfect
(the create path also writes two rows) and cannot be without a queue; argon2 is the dominant term
and it is now paid on both paths.

Not fixed here, and worth knowing: `emailClaimTrusted` still treats an **absent** `email_verified`
claim as trusted, so an OIDC provider that omits the claim can link a callback to an existing
account by a mutable email. That trade-off is deliberate — refusing it strands every account
created before `oidc_subject` was recorded — but it is a trade-off, not a non-issue, and it wants
a config switch rather than a constant.

## v0.2.2 — 2026-08-09

`session.Manager` gained `List` and `Revoke`. The `SessionStore` contract has advertised
`ListByUser` and `DeleteByID` since v0.1 as what an "your active sessions" screen is built on,
and the manager — which exists so that one thing owns the credential — exposed neither, so the
second adopter had to keep holding the store alongside the manager to show a user their own API
token. Found by Sablier, whose named API tokens are labelled sessions.

## v0.2.1 — 2026-08-09

The error sentinels were decoration. `porte.go` says they exist "so a handler can map them to
status codes without matching on message text", and `porte/local` was returning
`errors.Unauthorized(porte.ErrWrongPassword.Error())` — the text, not the error — so `errors.Is`
never matched and the only way to tell a wrong password from a closed registration was to compare
strings. They are wrapped now, keeping tronc's code and therefore the HTTP status.
`ErrWeakPassword` was declared and never returned at all; it now carries the configured minimum.

## v0.2.0 — 2026-08-09

Local passwords, and the restructuring they forced.

### The session stops belonging to OIDC

v0.1 put session issuance, the cookie, the authenticator and the middleware inside `porte/oidc`,
because OIDC was all there was. Journal's adoption priced that mistake: an app with its own
password form could not mint a porte session or set porte's cookie, so half its logins carried an
HttpOnly cookie and the other half kept a token in `localStorage` — the exact split the cookie
was adopted to end. Five of the six remaining apps have a password form, so shipping them onto
v0.1 would have spread it.

`porte/session` is that code, extracted and unchanged in behaviour, plus the three methods v0.1
was missing: `Issue`, `IssueCookie` and `Clear`. `POST /auth/logout` moves here too, because
ending a session never was an OIDC concern. `porte/oidc` keeps `RequireAuth`, `Optional` and
`Mount`, now delegating, and gains `Sessions()`.

**Breaking:** `oidc.Deps.Sessions` is now a `*session.Manager` rather than a
`porte.SessionStore`, and the app builds the manager. Two managers over one table would each
keep their own idea of the clock and the cookie, so there is exactly one and both kits share it.
Mount the manager as well as the kit — the logout route lives on the former now.

### porte/local

Email and password, argon2id, as an identity row under the new `porte.ProviderLocal` keyed on the
normalised address. It depends on `porte/session` and not on `porte/oidc`: an app that wants only
passwords must not compile an OIDC client, which is the whole reason the manager was extracted.

The parameters are copied from the apps rather than chosen — 64 MiB, three passes, two lanes, PHC
encoding — so every hash already in a Facile database keeps verifying. Adopting this is a code
change, not a password reset.

What is shared is not the flow, which is easy, but its details, which are what drift across six
copies: the constant-time compare, the equalised timing on an unknown address, the length floor,
and the refusal to say which half of the pair was wrong. `Register` and `Login` are exported as a
service, not only as routes, because every Facile app answers `{token, user}` and porte has no
idea what a user looks like — the app keeps its response shape and porte keeps the credential.

**A human may hold a password identity and a federated one at once, and they are one account.**
Registering a password against an address that already signed in through the IdP adds a row; it
does not create a second user and does not disturb the OIDC subject. That is what identities
having their own table since v0.1 was for.

porte cannot make registration race-free by itself: counting accounts and inserting one must
happen under a lock on a database porte does not own. `Deps.Count` and `PasswordUserStore` are
the app's, and every Facile app already takes the advisory lock.

### porte/avatarfs

The filesystem `AvatarStore` five apps have each written, once. Atomic writes, so a concurrent
read never sees half a file; a key guard, because a store that joins a caller-supplied string
onto a path is a directory traversal waiting for its second caller; and a handler that serves
only names `Put` could have written.

## v0.1.1 — 2026-08-08

What the first adoption found. Journal — the one suite app with no OIDC at all, so the
integration adds rather than replaces — hit three things in the first hour of wiring, and all
three are the same shape: `porte` had decided something on the app's behalf that was not its to
decide.

- **`Mount` owned `/auth/config` outright, so an app could not keep its own key there.** Every
  Facile app serves a superset of `sso_only` and `oidc_enabled` at that path — Journal adds
  `allow_registration`, Mycelium a legacy `password_auth` — and registering the route a second
  time makes chi panic at boot. `Deps.ConfigExtra func() map[string]any` merges the app's fields
  in, and `porte` writes its own two keys over the result: the frontend decides whether to draw
  a password form on those two, so they answer to the configuration and nothing else. Nil is
  today's behaviour byte for byte.
- **`/auth/logout` was mounted only when OIDC was enabled.** It is session management: it needs
  the `SessionStore` that `New` already demands whether or not a provider is configured. An app
  with SSO switched off therefore had to keep its own logout handler and a second response
  shape, and inherited a route collision on the day it switched SSO on — which is precisely the
  day nobody is looking at logout. It is now always mounted. `/auth/sync-profile` stays
  OIDC-only; refreshing a profile against an identity provider means nothing without one.
- **`attachClaims` documented itself as filling `Identity.Email` and `Identity.Name`. It never
  did.** No store on the authenticated path holds either — `StoredIdentity` carries no email —
  so the fields were silently empty for every consumer that trusted the comment. The comment now
  says what the code does, and `Identity` records that hydrating a profile is the app's job:
  `porte` authenticates a session, which tells it a user id and nothing else, and going to the
  app's user table for a name would double the cost of every authenticated request to serve the
  handlers that do not need one. Journal reads its own row into its own context, which is the
  query it was already making.

## v0.1.0 — 2026-08-08

OIDC only, as SPEC §4 scopes it: no local password, no `porte/espace`. The contract, the engine,
the PostgreSQL stores and the flow are complete and the whole thing is walked end to end against
a conformant issuer. **No application has adopted it in production yet** — that is the next
milestone, not this one, and it is why there is no `docs/architecture.md`.

### Proving the flow, and the hardening that came out of it

The engine had never spoken to an identity provider. SPEC §13 called PKCE, the nonce and the
back-channel logout token "the three paths a unit test cannot honestly cover, because they are
assertions about what the *provider* does" — which is true of a fake that only echoes what it is
handed. `oidc/flow_test.go` is a conformant in-process issuer instead: it signs RS256 tokens
behind a real JWKS, and its token endpoint **enforces** PKCE, the redirect URI and client
authentication rather than trusting the client to have sent them. The flow, the CLI exchange,
back-channel logout and the roles claim are now walked end to end, and a kit that dropped its
verifier or reused a nonce fails.

A parallel security review ran against the result. Seven findings survived adversarial
verification; four more were raised and refuted, and the refutations are worth as much:

- **The avatar SSRF guard let the IPv6 forms that embed an IPv4 address through.**
  `64:ff9b::a9fe:a9fe` (NAT64) and `2002:a9fe:a9fe::` (6to4) both reach the cloud metadata
  service, and every predicate in `net` says they are ordinary public IPv6 — `To4` only unwraps
  the IPv4-mapped form. They are now unwrapped to the address they actually reach and checked
  again, along with the deprecated `::a.b.c.d` form, Teredo, and the reserved IPv4 ranges `net`
  has no predicate for (CGNAT, the TEST-NETs, `240/4`, broadcast). NAT64 wrapping a *public*
  address still passes: on an IPv6-only host every IPv4 destination arrives in that form, so
  blocking the prefix outright would break every legitimate fetch.
- **The UserInfo response was never checked against the subject that asked for it.** OpenID
  Connect Core §5.3.2 makes this a MUST and `go-oidc` does not do it for you — it fetches and
  parses, nothing more. Without it a UserInfo response for somebody else rewrites this user's
  email, which is the key the rest of the suite joins on.
- **Cookies carry the `__Host-` prefix over https.** Every app in the suite sits on a subdomain
  of one parent, and a plain cookie named `session` scoped to the parent domain is
  indistinguishable at the server from the app's own host-only one — so one XSS, one rogue app or
  one subdomain takeover next door is enough to plant a look-alike and fix a victim into the
  attacker's session. The prefix is the one cookie property that cannot be forged: a browser
  accepts it only when the cookie is `Secure`, `Path=/` and carries no `Domain`. Over https the
  unprefixed name is **not read** unless `Config.AcceptLegacyCookie` is set: an unconditional
  fallback accepts precisely the cookie the attack plants, which would make the prefix
  decoration, and it is worst against a user who is not signed in and has no prefixed cookie for
  a preference order to prefer. The migration switch is meant to be on for one `SessionTTL` and
  then off. Over plain http the bare name is the only one a browser keeps, so it is the only one
  read there.
- **`Secure` is derived from the configuration as well as the request.** The per-request test is
  right behind Traefik and stays, but `Config.HTTPS()` now overrides it upward and never
  downward: a proxy that stops sending `X-Forwarded-Proto` must not be able to talk `porte` into
  shipping the session cookie in the clear.
- **Sessions gained an idle window.** `DefaultSessionIdleTTL` is seven days inside the thirty-day
  absolute lifetime, and it is the one default `porte` does not inherit from the apps — none of
  them can age out an unused session at all. Active users never meet it; a borrowed laptop stops
  being a month-long credential. It applies to the **cookie transport only**: everything arriving
  as a bearer is a CLI or an API token, which is idle by design and is the one class of
  credential with no human present to renew it. A negative `SessionIdleTTL` disables it.
- **`porte/pg` can finally tell a replayed login code from a typo.** The contract has always
  specified `ErrCodeConsumed` for the first case, and the shipped store returned `ErrNotFound`
  for both, so the replay branch in the engine was unreachable. `Consume` now stamps
  `consumed_at` under a conditional `UPDATE` rather than deleting: still exactly-once, still
  atomic, and what survives is the hash of a credential that is already spent. `DeleteExpired`
  sweeps those rows on the same schedule as the unused ones.
- **`IdentityStore.MarkRolesSynced` replaces a read-modify-write that could lose a token
  rotation.** When a role refresh fails, `porte` still records the attempt so a dead refresh
  token is not retried on every request — but it was doing that by saving back the whole identity
  it had read *before* the attempt, which would restore the old refresh token over one a
  concurrent request had just rotated in, and a lost rotation means every later refresh fails.
  One column, one statement, no read-modify-write. Apps implementing `IdentityStore` themselves
  gain a fourth method.
- **Concurrent first logins resolve to one user.** The pre-insert email check is a read in a
  `READ COMMITTED` transaction, so a double click, two tabs or a retried callback both passed it
  and one died on the unique index — the raw 500 on the login path that the check exists to
  prevent. The insert is now `ON CONFLICT (email) DO NOTHING RETURNING id` with a re-select, so
  the loser adopts the winner's row — and the conflict path re-applies the unverified-email
  guard rather than assuming it, because two *different* subjects can reach it with the same
  address, and adopting there would be the takeover the guard exists to refuse.
- `POST /auth/oidc/exchange` answers with `Cache-Control: no-store`. It is the token endpoint of
  the CLI flow in everything but name.
- An expired or idled-out session row is deleted when it is presented, rather than left to the
  sweeper to find.

Refuted, and recorded so they are not re-litigated: `SameSite=Strict` is not deployable here —
the `oidc_state` cookie has to survive the top-level cross-site redirect back from the provider,
and `Strict` would withhold it and break every login at the callback. The custom-header check the
browser-apps BCP offers as the sanctioned alternative is enforced on every mutating cookie
request, so `Lax` is compliance rather than a gap. Clearing the stored IdP tokens on logout was
also refused: they are user-scoped and shared across a user's other live sessions, and
Back-Channel Logout §2.7 explicitly exempts `offline_access` refresh tokens, which is exactly
what the CLI and the role refresh need.

### The engine and the stores

`porte/oidc` and `porte/pg`: the flow, the seven routes, the middleware, the SSRF-guarded avatar
fetch, and the four tables. Tested against a real PostgreSQL, not a fake — every interesting
behaviour in `pg` is PostgreSQL's own.

- **`porte/oidc` is a separate package, not the root.** SPEC §8 put the implementation in the
  root package, which contradicts the zero-dependency decision taken with the contract: an app's
  `UserStore` must not compile against `go-oidc`. The literal promise is unreachable inside one
  module anyway — `go.mod` requirements are module-scoped — but the layering is worth keeping,
  and it costs one import line in `main.go`. Only that file sees the engine.
- **Four store types in `porte/pg`, not one.** `SessionStore.Find` takes a token hash and
  `IdentityStore.Find` takes a provider and a subject; Go cannot carry both on one receiver.
  Renaming a method to dodge that would leak a storage accident into the contract.
- **The Go floor is 1.25, not 1.24.** SPEC §8 left this open and leaned 1.24 on the principle
  that a library floors low. `go-oidc` v3.20 requires 1.25, so the dependency decided. It costs
  nothing: the apps are on 1.25 and `tronc/migrate` already declares 1.25.7.
- **The avatar SSRF guard now runs inside the dialer.** All six existing copies resolve the
  hostname, check the addresses, then call `http.Get` — which resolves again. A DNS record that
  answers publicly on the first lookup and privately on the second walks straight through.
  Checking at connect time closes that window and covers redirects for free.
- **`UpsertFromOIDC` refuses an unverified email that already belongs to someone**, with a
  message, instead of hitting the unique index and returning a 500 from the login path.
- **The startup guard is split in two**, because only half of it is checkable at boot: `New`
  verifies the roles scope against the discovery document, and the callback fails loudly when
  the scope was granted but no claim arrived. Between them there is no path where a
  half-configured provider denies everyone in silence.

### Contract corrections, found by implementing it

SPEC §11 froze the contract and asked for a review before anything was built on it. Writing the
implementation *was* the review, and it found three things wrong:

- **`LoginCode.SessionID` is gone.** The session is created at exchange time, not at callback
  time. The old shape could not work: the session row stores only a hash, so handing the CLI a
  usable token later would have meant keeping the plaintext at rest. The stated reason for
  creating it up front — avoiding a second write path that could half-succeed — does not survive
  contact with the fact that creating a session is one insert. `Consume` is atomic, so a code
  still yields at most one session.
- **`AvatarStore.Put` takes an opaque key, not a user id**, and `Claims` gained `AvatarURL` and
  `AvatarKey()`. The old signature had an ordering problem with no solution: the avatar URL must
  reach the app's user row, the app writes that row in `UpsertFromOIDC`, and `Put` needed a user
  id that does not exist until that call returns. Keying on the identity instead lets the fetch
  run first and the URL ride into the upsert. A stable key also means a re-sync overwrites rather
  than accumulating one file per login.
- **`Config.Validate` requires an absolute http(s) issuer.** The old check called `url.Parse` and
  tested the error, which catches almost nothing: `sso.facile.studio` parses fine as a relative
  path, and the failure surfaces later as an opaque discovery error naming neither the variable
  nor the problem.

### The frozen contract, as Go types

`porte.go`, `identity.go` and `session.go`: types, interfaces and wire shapes only, no
behaviour. This is SPEC.md §11's first step, taken after diffing the six `modules/auth/`
implementations rather than before.

- **Zero dependencies, standard library only.** An app implementing `UserStore` must not
  inherit `go-oidc`, `golang.org/x/oauth2` or a database driver from `porte`. That is why
  `Claims` carries plain fields and `TokenSet` exists instead of an `*oauth2.Token`. The
  implementation may depend on whatever it needs; the boundary an app sees may not.
- **`Identity.UserID` is an `int64`.** Every app passes a decimal string around today. The
  conversion moves to the edge, and the type matches `porte_users.id` and the `int64` foreign
  keys already in place. `ExchangeResponse.UserID` stays a string on the wire: to a CLI it is
  an opaque identifier, so breaking it buys nothing.
- **The middleware yields a typed `Identity`.** It replaces
  `Authenticate(ctx, string) (string, any, error)` followed by an
  `interface{ GetEmail() string }` assertion, which is a runtime failure mode sitting in the
  auth path of six apps.
- **Stores take hashes, never tokens.** Keeping the plaintext out of the store interface is
  what stops it reaching a log line or a query parameter.
- **`SessionStore.Find` does not filter on expiry.** An expired session and a missing one are
  different answers, and only the caller knows which error to return.
- **`ErrCodeConsumed` is separate from `ErrNotFound`.** A replayed login code is an attack;
  an unknown one is a typo. One error value buys that distinction in the logs.
- **`CSRFHeaderName` is `X-Facile-CSRF`, any non-empty value** (SPEC §7 Q5). No app sends a
  CSRF header today, so the name was free. Presence is the whole signal — a browser will not
  attach a custom header cross-site without a preflight — so there is no token to mint,
  distribute or rotate.
- **`DefaultClaimsTTL` is five minutes** (SPEC §7 Q6), deliberately the same number as the
  existing `profile_synced_at` rate limit. One refresh cadence, not two.
- **`SessionCookieName` is `session`,** taken from Courrier and Agenda, which already ship the
  cookie transport. Adopting their name means those two do not have to log everyone out twice.

### SPEC.md corrected against source

The diff found three claims that had gone stale, all dated in place rather than deleted:

- **Matching on `sub` is already fixed in all six apps** — it is the HEAD commit of every
  `modules/auth/`, and every `schemas/user.go` now carries `OIDCSubject *string uniqueIndex`.
  It was listed as "the most severe item"; `porte` now preserves it instead of introducing it.
- **The cookie transport is already live in Courrier and Agenda**, including a `Secure` flag
  derived from `X-Forwarded-Proto`, which is the correct test behind Traefik.
- **Constant-time state comparison already exists in Plume.** PKCE and the nonce are still
  missing everywhere, as specified.

New findings recorded: Courrier and Agenda encrypt OIDC tokens at rest while four apps do not;
Courrier's middleware still accepts `?token=` and Vision's `?api_key=`, which `porte` will not
carry forward; `authcrypto` has three distinct hashes across seven apps, not one.

### Repo

- `.github/workflows/ci.yml`, copied from `caisse` and pinned to Go 1.24 — the floor `go.mod`
  documents. Held back until the first package existed, because `go vet ./...` on an empty
  module exits non-zero. The PostgreSQL service is present from the start: a database test that
  skips itself when the URL is unset passes silently, and a green CI that ran nothing is worse
  than a red one.
