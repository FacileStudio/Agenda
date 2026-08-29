# porte — Specification

The authentication kit for the Facile Suite. This document is the brief for building it: what
it is, what was decided and why, the contract it must freeze, and what it must not become.

Written 2026-08-07, before any code. Everything below that describes existing apps was read
from source on that date, not recalled.

---

## 1. What porte is

Three layers get confused in conversation. They are different things:

| Layer | What | Who owns it |
|---|---|---|
| Protocol | OIDC — discovery, JWKS, auth code flow | Nobody here. `go-oidc` does it |
| **App plumbing** | `/auth/oidc` routes, sessions, cookie, user upsert, avatar, `SSO_ONLY`, role hook | **`porte`** |
| Identity provider | The server that issues identities | **Authentik**, at `sso.facile.studio` |

`porte` is the middle layer only. It is not an OIDC implementation — OIDC is a standard, and
Authentik and Dex both ship no client SDK at all for exactly that reason. What `porte` holds is
the *Facile-specific* glue that six Go apps have each written separately.

`porte` names the door **in your app**, matching how `tronc`, `caisse` and `enveloppe` are
named: each names the component the library contributes inside the app, never an external
service.

### What it is not

- **Not the IdP.** Authentik keeps its own name. We do not alias a third-party product.
- **Not a future IdP's client SDK.** If Facile ever builds its own identity provider, that
  server gets a fresh name chosen then, and `porte` keeps working against it unchanged —
  because `porte` only speaks standard OIDC. That is the whole point of freezing the contract.
- **Not MFA.** TOTP, WebAuthn and passkeys belong in the IdP, never in this library. That is
  the reason a central IdP exists: the day it has TOTP, all seven apps have it with zero lines
  changed. If MFA touched this library, adding a factor would mean redeploying seven apps.

---

## 2. Why it exists — the evidence

Read from source, 2026-08-07.

Six Go apps ship near-identical OIDC client code: **Nuage, Courrier, Plume, Agenda, Vision,
Sablier**. Each has the same 6 files under `apps/api/modules/auth/` plus `internal/authcrypto/`.
Journal has `authcrypto` and sessions but no OIDC at all. Capsule has no auth by design.

Measured 2026-08-07, `modules/auth/` only:

| App | total | `oidc.go` | `service.go` | `types.go` |
|---|---|---|---|---|
| Sablier | 664 | 162 | 314 | 27 |
| Courrier | 766 | 164 | 336 | 27 |
| Nuage | 761 | 169 | 373 | 35 |
| Agenda | 779 | 180 | 351 | 27 |
| Vision | 839 | 161 | 367 | 46 |
| Plume | 969 | 224 | 390 | 48 |

`types.go` is **byte-identical in three of the six** (Courrier, Agenda and Sablier share one
hash). `diff Nuage/oidc.go Sablier/oidc.go` is 27 lines out of ~165 — roughly 85% identical,
which is the number that justifies the extraction. Plume is the outlier at both ends because it
carries the CLI exchange flow (§5b).

`internal/oidcavatar/` — HTTPS validation, private-IP/SSRF rejection, download, store — is
copy-pasted in six apps as well, and has drifted **further than the auth code**: six files,
**five** distinct hashes, only Courrier and Sablier matching. It is a security guard, so this is
the worst place in the suite for five divergent copies.

`internal/authcrypto/` — the argon2 and token helpers — has drifted less but is not one file
either: **three** distinct hashes across seven apps (Nuage = Courrier = Agenda = Sablier; Plume =
Vision; Journal its own). Its API is small and stable — `NewToken`, `HashToken`, `HashPassword`,
`VerifyPassword` — which is what makes it a clean v0.2 extraction. Measured 2026-08-07.

The OIDC surface is **identical across all six**:

```
GET  /auth/config          → {"sso_only": bool, "oidc_enabled": bool}
GET  /auth/oidc            → redirect to the IdP
GET  /auth/oidc/callback   → exchange, verify, upsert, issue session
```

Everything else has drifted:

| Route | Apps that have it |
|---|---|
| `POST /auth/logout` | Nuage, Courrier, Agenda (others via a different registration shape) |
| `POST /auth/sync-profile` | Nuage, Plume, Vision |
| `POST /auth/register`, `POST /auth/login` | Nuage, Plume, Vision, Sablier — gated on `!SSO_ONLY` |
| `GET/PUT /auth/me`, `PUT /auth/password` | Plume, Vision |
| `POST /auth/oidc/exchange` | Plume only — believed to be the CLI token path |

That split is the map of what belongs in v0.1 (the identical part) and what has to be settled
before v0.2 (the drifted part).

---

## 3. Decisions already taken

Do not re-litigate these. Each has a reason recorded.

| Decision | Reason |
|---|---|
| Name is `porte`, not `sésame` | Matches the naming convention (the component *in* your app); no accent to lose in the module path; "sésame" names the act of opening, not identity |
| Authentik keeps its name | Never rename someone else's product; it removes the permanent "Porte is actually Authentik" explanation |
| `porte.facile.studio` → **`sso.facile.studio`** | Frees the name, and `sso.` survives replacing Authentik whereas `authentik.` would need a second rename |
| The old `Porte` repo → `authentik-config` | 93 lines of Authentik policy scripts. It frees the name. **Renamed locally 2026-08-07. There was never a GitHub repo for it — no remote, no commits — so nothing to rename there.** ~~And it has no reason to grow~~ — **wrong, corrected 2026-08-07**: the scope mapping that emits the `roles` claim (§5c) is server-side Authentik Python, so it belongs there next to the two existing expression policies. `authentik-config` is now the home of the claim contract's producing half |
| **`porte` carries authorization data, owns no policy** | Authentik decides *who has which role*; the app decides *what a role may do*; `porte` owns only the transport and the freshness in between. The three role models in production (`is_admin` + first-user-admin, Vision's workspace roles, the TS `USER`/`ADMIN` enum) are product decisions, not drift — a library that arbitrated them would be routed around by the second app. See §5c |
| **The claim contract is a flat `roles` array, produced per-provider** | Chosen over parsing group names (`facile-nuage-admin`) and over a nested per-app JSON claim. Each Authentik provider gets a scope mapping emitting `{"roles": ["admin"]}` already filtered and stripped for that app — namespacing and least privilege for free, no parsing in Go, and the one moving part lives in Authentik where it is configuration rather than code. Decided 2026-08-07 |
| **Back-channel logout is in v0.1** | Authentik has shipped it since 2025-10, and their docs are explicit that it is the *only* way a downstream app learns a session was terminated administratively. One endpoint in `porte`, one URL per Authentik application, and deactivating a user drops them from all seven apps in seconds. Highest value-to-effort item in this document |
| A future in-house IdP gets a new name, chosen when built | Naming a project you have not decided to build is a free commitment. Right register when it comes: institutions that issue identity papers — `consulat`, `préfecture`, `mairie` |
| Rebuilding Authentik is **not** decided | `porte`'s value does not depend on it. "Zéro dépendance cloud" does not apply — Authentik is already self-hosted on la ruche. Reopen the question only when `porte` runs on all seven apps, the OIDC contract has been stable for months, and something concrete is blocked by Authentik |
| Forced logout of production users is acceptable | User decision, 2026-08-07. This removes the whole backward-compatible session migration problem: one canonical session model, no dual-read, no cookie compatibility shims |
| Password hashing is already settled | All seven Go apps use argon2 today. No hash migration needed. (~~Journal carries bcrypt alongside argon2~~ — checked 2026-08-07: the only bcrypt hit is a test fixture string in `crypto_test.go`. Journal is argon2-only. Closed) |
| **Browser sessions ride an `HttpOnly` cookie, not `localStorage`** | Today the callback hands the token via URL fragment and the SvelteKit apps store it in `localStorage` — one XSS reads every token (RFC 9700, OWASP session cheat sheet). The suite's mono-container topology (P1b: same-origin `/api` behind Traefik, SPA served by `tronc/spa`) makes `HttpOnly; Secure; SameSite=Lax` cookies free. Bearer stays for CLIs and API clients; the middleware accepts both. Forced logout is already accepted, and the frontends are already being rewritten — this train does not pass twice. Decided 2026-08-07 |
| **`porte/pg` ships default identity tables; `UserStore` stays the escape hatch** | The six `schemas/user.go` share 12 byte-identical columns plus 0–4 business ones — the identity/profile split already exists, hand-written six times. Owning the shape once is what makes `facile_id`, `updated_at` (P4.3) and the `sub` fix land once instead of six times. Supabase (`auth.users`/`public.profiles`) and Ory Kratos both converge on the same boundary. Provided, never imposed — the `caisse` pattern. Decided 2026-08-07 |
| **OIDC account matching keys on `(provider, subject)`, never on email** | Email is mutable in Authentik: an email change silently orphans the account; a deleted-then-recreated IdP account silently inherits the old one. Wiki: `bugs/facile-oidc-email-matching.md`. Decided 2026-08-07. ~~All six apps match by email today and no schema stores `sub`~~ — **corrected 2026-08-07 after the §11 diff**: the fix already landed in all six (HEAD commit of each `modules/auth/`: *"Match OIDC accounts on sub instead of the mutable email"*). Every `schemas/user.go` now carries `OIDCSubject *string uniqueIndex`, and the lookup is subject-first with an email fallback gated on `email_verified`. So `porte` **preserves** this behaviour rather than introducing it — see §6 |
| **Identities are a separate table from users, from v0.1** | v0.2 explicitly plans local password *and* OIDC on the same human — one credential column set on the user row cannot represent that. `porte_identities(provider, subject)` also models "Login with Google" as a config change instead of a schema break, which client projects will want. Wire a single provider in v0.1; design the table for several. Decided 2026-08-07 |
| Pilot is the e-commerce demo + Nuage | Comptoir does not exist (checked 2026-08-07 — not locally, not in the org). A greenfield demo forces the works-outside-the-suite test on day one; Nuage is the extraction source, so the most honest feedback on fit |

---

## 4. Scope

### v0.1 — OIDC only — built 2026-08-08

The identical part, extracted and hardened. No local password: apps that still need
email/password keep their own code in parallel for now, and lose roughly 60% of their auth
code rather than 100%.

In scope for v0.1, beyond the routes (all three decided 2026-08-07):

- **`oidcavatar`** — the HTTPS-validation / private-IP / SSRF guard. Six distinct hashes exist
  across the apps today; this is the file where drift is a security bug rather than cosmetics.
  The ROADMAP (task 3.4, exit criterion "exists exactly once") already required it; an earlier
  revision of this spec cited it as evidence but left it out of scope. That was the mistake.
- **`POST /auth/oidc/exchange`** — Plume's one-time-code CLI flow, generalized. Plume stores
  pending codes in a `sync.Map`, which dies on redeploy and breaks at two replicas. `porte`
  stores them in `porte_login_codes` (TTL 60s, single-use, hashed like sessions). Six CLIs
  need this; see §5b.
- **Cookie session issuance** — the callback sets the session cookie instead of putting the
  token in the URL fragment. See §5, Sessions.
- **Back-channel logout** — `POST /auth/backchannel-logout`, so an administrative deactivation
  in Authentik actually reaches the apps. See §5c.
- **Role claims, optional** — the `roles` claim, its startup guard and its refresh TTL. Off
  unless configured; no app reads claims today, so nothing regresses by leaving it off. See §5c.

### v0.2 — local password — built 2026-08-09

Shipped as `porte/local`: `Register`, `Login`, `SetPassword`, argon2id at the parameters already
in the apps' databases, and `POST /auth/register` / `POST /auth/login` for an app with no opinion
about its login response. Journal runs it.

Three corrections to what this section planned:

- **`me` is not in it, and will not be.** `porte` authenticates a session, which tells it a user
  id and nothing else; the profile behind `/auth/me` lives in the app's own user table, in a shape
  `porte` has no opinion about. Journal serves its own, as every app already did.
- **It is a separate package, not routes bolted to the engine** — and getting there meant
  extracting `porte/session` first. See §8.
- **It was not blocked on the role model in §7.** That was a misreading: a password login issues
  the same opaque session a federated one does and never touches a claim. What it was actually
  blocked on was the layering, which only the first adoption made visible.

The `authcrypto` extraction this section anticipated did not happen either. There was nothing to
extract to: the hashing is forty lines with one caller, and a second module to hold it would have
been a dependency for the sake of a filename.

Note for client projects: v0.2 is where `porte` starts serving **two user populations** — staff
logs in through SSO, end customers (an e-commerce site's buyers, who will never have an
Authentik account) through email/password or a second OIDC provider. `porte_identities` is
designed for that from v0.1 even though v0.1 wires only one provider.

### v0.5 — `porte/spaces`

A **subpackage**, same repo, same tags. Membership queries and role resolution: `Role`, `Ladder`,
`Membership`, `Store`, `Scope` and the `Guard` that answers `Resolve`, `Require`, `CanLeave` and
`AssignableBy`. Apps without spaces (Journal, Comptoir) simply do not import it — the pattern
already used by `tronc/migrate`, `tronc/testdb` and `caisse/pg`.

**Two names changed between the plan and the code.** The package is `spaces`, not `espace`,
because every other subpackage is an English mechanism name (`oidc`, `pg`, `session`, `local`,
`avatarfs`) and every app already calls the concept `Space` in its own source — `schemas/space.go`,
`modules/spaces`, `SpaceMember`, muse's `SpaceSwitcher`. And **v0.4 was taken** by the OIDC device
exchange before this was built, so it lands as **v0.5.0**.

**The role ladder is configurable rather than fixed at three roles.** This is the one design change
against the plan, and Vision forced it: `internal/siteaccess/siteaccess.go:29` gates every write on
`owner|admin|editor`, with `viewer` below. A package hard-coding owner/admin/member would have left
Vision holding its own copy of the guard, which is one more copy from the package built to end them.
So `Ladder` is a list of roles ordered by privilege, `Default()` is the suite's three, and every
check the package makes is a comparison inside a ladder rather than a switch over named roles. A
role the ladder does not rank is unknown, not weak: it fails every check rather than scoring zero.

**Four invariants, each currently violated by at least one adopter.** They are the reason the
package exists, and `spaces/spacestest.Conformance(t, newStore)` is how an adopter inherits the
proof rather than only the code — it runs them against the app's *own* `Store`.

1. **Membership is the only key.** No instance-admin bypass anywhere in the package, and no hook to
   add one. An app that wants staff inside a space grants them a membership, where it is listed and
   revocable. A bypass inside `porte` would be invisible to every app importing it.
2. **A space id is checked before it is usable.** An empty id is personal scope in `Resolve`, and
   never touches the `Store`; any non-empty id goes through `Store.Membership`. A `Scope` that
   reports `Resolved` and carries a space id is therefore proof of membership in that space. A
   refusal returns the zero `Scope`, and the zero `Scope` reports `Resolved() == false` and
   `Personal() == false`, so a caller that ignores the error holds nothing usable.
3. **A space always has a reachable owner.** `CanLeave` refuses with `ErrSoleOwner` when the caller
   holds the ladder's top rank and is the only member who does. It does *not* refuse an owner with a
   peer, which three apps do — that makes ownership transfer the only exit from a space two people
   own equally.
4. **No privilege escalation, in both directions.** `AssignableBy(actor Scope, target Role)` is
   false when target outranks actor: an admin may appoint a peer admin and may not mint an owner.
   That is only the grant. `AssignableOver(actor Scope, current, target Role)` adds the role being
   taken away and is the check for modifying an existing member, so an admin cannot hand "member"
   to the owner and strand the space. Added in v0.5.1 after review; v0.5.0 shipped `AssignableBy`
   alone and its godoc oversold it.

**Three signatures answer to the invariants rather than to convenience.**

- `Require(ctx, userID, spaceID, min)` refuses an empty space id with `ErrNotMember`. Passing every
  minimum on an absent id is fail-open on empty input, and the realistic exploit is a gate and a use
  reading the id from different places — `Require(ctx, uid, r.Header.Get("X-Space"), RoleAdmin)`
  with no header, then a handler acting on the id in the body. A handler that genuinely serves both
  shapes calls `Resolve` and branches on `Scope.Personal`.
- `AssignableBy` and `AssignableOver` take the resolved `Scope`, not the actor's `Role`. Two plain
  roles invite passing both straight off the wire, which checks the request against itself. `Scope`
  carries an unexported marker only `Guard` sets, so `Scope{UserID: "mallory", Role: RoleOwner}`
  still compiles and grants nothing. `AssignableOver`'s `current` is a plain `Role` on purpose: it
  is the row the caller has just read, usually inside the transaction about to update it, and a
  second lookup through the `Store` would reintroduce `CanLeave`'s time-of-check gap. The godoc
  carries the obligation that it comes from the app's own row and never from the request.
- `Resolve` requires the returned row to carry **both** ids and both to equal what was asked for. An
  absent id is not agreement: `SELECT role FROM ... WHERE space_id=$1 AND user_id=$2` is the most
  natural `Store`, and treating its blank ids as a match would disarm the cross-check for exactly
  the implementation most apps write. `Spaces` applies the same rule to every row it lists, because
  that list is what a space switcher renders.

**`CanLeave` is time-of-check to time-of-use, and the adopter has to close it.** It counts, the
caller deletes, and two owners leaving at the same instant both count two and both pass. The
package cannot fix that without a database dependency it refuses to take, so the contract is on the
caller: run `CanLeave` and the `DELETE` **in one transaction, with the space's membership rows
locked** — `SELECT ... FOR UPDATE` over the rows `CountRole` counts, or a serializable transaction
with a retry. Sablier, Agenda and Plume ship the unlocked count-then-delete today; importing the
package without the lock reproduces their bug.

**The conformance suite asserts content, not row counts.** `Conformance` checks that every
`Membership` a `Store` returns carries ids populated from the row and matching the arguments, that
roles come back as they were stored, and that `Memberships` lists the caller's own rows and no
others. Counting alone certified a store that blanked the ids and one that promoted every listed row
to owner. It does **not** catch a store that builds its result out of the arguments it was handed
rather than the row it read: a correctly scoped lookup returns the same ids either way, and no
black-box suite can separate them, whatever ids it seeds. What it catches is the half that causes
harm, a lookup not scoped to the space, and since v0.5.1 `UserCoOwner` holds a different rank in each
of the two fixture spaces so the *role* pins the scoping too. `ConformanceWithLadder(t, newStore, ladder)` runs the same suite on an app's own
vocabulary — Vision forced `Ladder` to be configurable, and proving the guard on the default ladder
would prove nothing about the guard Vision ships. The ladder must rank at least three roles: the
suite needs a top, a middle and a bottom to tell a refusal from an escalation.

**A ladder built by `NewLadder` is never the zero `Ladder`.** `Guard` substitutes `Default()` only
for the unset one, so `NewLadder(cfg.Roles...)` over a misconfigured config refuses every role
instead of silently inheriting the suite's three. `Ladder.Configured()` reports which one an app is
holding.

One argument order was reversed against this section's original sketch: `CanLeave(ctx, userID,
spaceID)` matches `Resolve` and `Require` rather than taking the space first. Both parameters are
strings, nothing would catch a swap, and the two calls sit in the same handler.

Evidence: `Space`/`SpaceMember` is copy-pasted in Nuage, Sablier, Agenda, Courrier and Plume,
and as `Workspace` in Vision. Already drifted — `modules/spaces/types.go` is 54/49/40/39/45
lines with five different hashes, and the wire contracts genuinely diverge (`ID` is a string in
Courrier and an int64 in Nuage; Nuage wraps lists, Courrier returns bare arrays).

**Scope limit, deliberate:** models, membership and role resolution only. **No CRUD, no
invitation flow, no HTTP routes, and no GORM.** Those stay in each app, because converging them
would mean migrating six SvelteKit frontends, and because the authorization logic is the only part
where drift is a security bug rather than cosmetics. If spaces ever grow a real product surface —
email invitations, quotas, per-space billing — that is when it earns its own repo.

The limit went one step further than planned once the code existed: **the package carries no
persisted model either.** `Space`/`SpaceMember` is where the five copies diverge hardest — `ID` is
a string in Courrier and an `int64` in Nuage — and a shared struct would have made adoption a
migration in six databases before the guard could run in any of them. So the app keeps its table
and implements a three-method `Store` over it, converting to `Membership` at the boundary; ids
cross that boundary as strings, which every app can produce. Standard library only, so a package
every app's authorization depends on drags neither an ORM nor a router into every binary.

`FacileID` still carries a unique index in the apps, which only makes sense if the same space is
meant to exist in several apps. Confirmed intent: **sync via Antenne later**, so the whole park's
spaces work together. Treat `FacileID` as the sync key from the start, even though the sync comes
later, and note that nothing in `porte/spaces` blocks it — the sync operates on the app's own table,
which the package never claims. Still to check: the `enveloppe` contract keys on `actor_email` and
will need to carry a *space* identity. That is a change to `enveloppe`, and it gates the sync, not
this package — which is why `porte/spaces` shipped without waiting for it.

---

## 5. The frozen contract

This is the part that costs the most if it is wrong, and the reason to read six
implementations before writing one line. Freezing it now is what makes a future
Authentik → in-house IdP swap a config change rather than a rewrite.

### Environment

| Variable | Required | Meaning |
|---|---|---|
| `OIDC_ISSUER` | no | Issuer URL. **Its presence is what enables OIDC** — this is the existing convention, keep it |
| `OIDC_CLIENT_ID` | with issuer | |
| `OIDC_CLIENT_SECRET` | with issuer | |
| `OIDC_REDIRECT_URL` | with issuer | Must match the Authentik application |
| `OIDC_SUCCESS_URL` | with issuer | Where the browser lands after a successful callback |
| `OIDC_APP_NAME` | no | The name on the two pages porte renders itself. Empty derives it from `OIDC_SUCCESS_URL`'s first DNS label |
| `OIDC_LOGO_URL` | no | The logo on those pages. Empty derives `OIDC_SUCCESS_URL`'s origin plus `/logo.svg` |
| `OIDC_CLAIMS_SCOPE` | no | The scope carrying the `roles` claim (§5c). Its presence enables claims handling |
| `OIDC_MACHINE_AUDIENCE` | no | This app's own client id. Its presence enables bearer-JWT verification on the header path |
| `OIDC_CLI_AUDIENCE` | no | The CLI's client id. Its presence mounts `POST /auth/oidc/device/exchange`, and nothing else does |
| `SSO_ONLY` | no | When true, local password routes are not registered at all |

Authentik application slug convention, unchanged: `sso.facile.studio/application/o/<app>/`.

### Routes

```
GET  /auth/config          {"sso_only": bool, "oidc_enabled": bool}   no auth
GET  /auth/oidc            302 to the IdP                             no auth
GET  /auth/oidc/callback   sets session cookie, 302 to OIDC_SUCCESS_URL   no auth
POST /auth/oidc/exchange   one-time code → bearer token (CLI path)    no auth, code required
POST /auth/oidc/device/exchange  issuer access token → this app's session   no auth, token required
POST /auth/logout          {"logged_out": true}, clears the cookie    session required
POST /auth/sync-profile    refresh profile from the IdP               session required
POST /auth/backchannel-logout   IdP-initiated session kill            logout token, no session
POST /auth/register        v0.2, porte/local. {user_id, token} + cookie   no auth
POST /auth/login           v0.2, porte/local. {user_id, token} + cookie   no auth
```

`POST /auth/logout` is mounted by `porte/session`, not by either kit (§8). The two local routes
are mounted by `porte/local` and are optional in a second sense as well: an app whose frontend
expects the `{token, user}` body every existing Facile app answers calls `Register` and `Login`
from its own handlers instead. `porte` keeps the credential, the app keeps its wire shape.

Scopes requested: `openid email profile offline_access` — unchanged, plus the scope carrying
`roles` when claims are enabled (§5c). `offline_access` is already requested by every app today,
which is what makes silent claim refresh possible without a second login.

### Sessions — one model, two transports

Opaque token, generated randomly, **stored hashed** (the existing `authcrypto.NewToken` /
`HashToken` pair). Keep this — it is stateful by design, revocable, and already what all six do.

What changes (2026-08-07) is how the browser carries it:

**Prior art inside the suite, found 2026-08-07:** Courrier and Agenda **already do this**. Cookie
name `session`, `Path=/`, `MaxAge=SessionTTL`, `HttpOnly`, `SameSite=Lax`, and `Secure` derived
from `r.TLS != nil || X-Forwarded-Proto == https` — which is the correct test behind Traefik and
better than Nuage's `strings.HasPrefix(successURL, "https://")`. Take Courrier's `isSecure(r)`
and its cookie name verbatim: two of the six then need no logout at all, and there is no third
spelling of the cookie to reconcile later.

- **Browser: `HttpOnly; Secure; SameSite=Lax` cookie**, set by the callback. The token never
  appears in a URL, never touches `localStorage`, and the frontends' token-handling code is
  deleted rather than migrated — `fetch` gains `credentials: 'include'` and that is all.
  CSRF: `SameSite=Lax` plus a custom-header check on mutating routes.
- **CLI and API clients: `Authorization: Bearer …`**, unchanged.

The middleware accepts both, cookie first. Same table, same hash, same revocation.

The sessions table carries `label`, `created_at` and `last_used_at` — absent today, and the
reason no app can show "your active sessions" or revoke one token without revoking them all.
`label` is what turns a session row into a named API token (§5b).

`porte` owns the sessions table and ships `porte/pg` for it, following `caisse/pg`: an
interface plus a `database/sql` implementation, so no ORM is forced on consumers.

### 5b. CLI login and API tokens

One flow, one table, both use cases.

**Interactive CLI login** — the pattern `gh auth login` and Claude Code use: the CLI opens
`/auth/oidc?flow=cli` in a browser, the user logs in through the IdP, and the callback — instead
of setting a cookie — issues a **one-time code** (stored hashed in `porte_login_codes`, TTL 60s,
single-use). The success page displays it for copy-paste; a CLI that started a loopback listener
receives it directly and the paste is the fallback. The CLI exchanges it at
`POST /auth/oidc/exchange` for a bearer token. This is Plume's existing flow with the `sync.Map`
replaced by the database, so it survives redeploys and replicas.

**API tokens** are not a separate mechanism: an API token is a session row with a `label`, a
long or absent expiry, and `last_used_at` for auditability. Courrier and Agenda have already
each grown their own `ApiToken` types — same drift pattern as everything else in this document.
`porte`'s session store exposes create/list/revoke-by-id; the HTTP routes for "personal access
tokens" stay app-side in v0.1 (they are product surface), backed by the store.

Deferred, explicitly: token **scopes** (a `porte` token is currently as powerful as its user)
and the full RFC 8628 device flow delegated to Authentik, which supports it natively — that
upgrade replaces the copy-paste UX with `gh`-style `user_code` entry and belongs in v1.x.
Store refresh material in the OS keychain, never a JSON file — a CLI-side concern, documented
here because six CLIs will read this spec.

**The device half landed 2026-08-25, and it is not the deferred item above.** Registre runs the
RFC 8628 grant; `porte` never speaks it. What `porte` added is the last hop, `POST
/auth/oidc/device/exchange`: the CLI runs the grant once against the provider, gets one access
token, and trades it at each tool for that tool's own session. Writing the provider's token into
the slot where a CLI keeps its session is a login that stops working when that token expires, so
the trade is not optional. The handler verifies the token through the same JWKS verifier the
Authorization header path uses, resolves `sub` through `porte_identities`, and calls
`Manager.Issue`. It mounts only when `OIDC_CLI_AUDIENCE` is set, a **second** audience holding
the CLI's client id, distinct from `OIDC_MACHINE_AUDIENCE`, which holds this app's own and
arms the header path for service accounts. One variable could not serve both: a service-account
token is addressed to one app (`suite-ci` declares `audiences: [courrier]`) while the CLI's is
addressed to the CLI and presented at all of them. The CLI audience builds its own verifier and
never reaches `session.Manager.WithJWT`, so a CLI token is not a credential on every route and
the exchange stays a boundary. Its 404 when unset is the signal `facile login` reads as
"not shipped". The client half lives in `facile`, which probes the route before running the
grant so a tool without it never makes a human read a code off one screen for nothing.

### 5c. Claims, roles, and freshness

Authorization has three layers, and conflating them is how auth libraries turn into frameworks:

| Layer | Question | Owner |
|---|---|---|
| Assignment | is saravenpi an admin of Nuage? | **Authentik** — groups and attributes |
| **Transport + freshness** | how the app learns it, and re-learns it | **`porte`** |
| Policy | what may an admin actually do? | **the app** |

`porte` is the middle row, exactly as it is for authentication. It decides nothing.

Two facts from source, 2026-08-07, that make this a clean slate: every app requests
`openid email profile offline_access` and **no app requests or reads a `groups` claim**. Nothing
to migrate, and refresh tokens are already in hand.

**The claim.** A flat array of opaque strings, in a `roles` claim:

```json
{ "roles": ["admin"] }
```

Produced by a per-provider scope mapping in Authentik, already filtered and stripped for that
application — so Nuage's token never mentions Sablier's roles. The alternatives were parsing
group names (`facile-nuage-admin` — an implicit contract in strings, which is what drifts) and a
nested `{"facile": {"apps": {...}}}` claim (explicit but needs parsing in six apps). A flat array
per provider gets namespacing and least privilege for free while keeping the Go side trivial.
`porte` exposes the strings; it never assigns them meaning.

The producing half — a small Python scope mapping, parameterised by app slug — lives in
`authentik-config` alongside the two existing expression policies.

**The startup guard, and why it exists.** authentik's docs carry a specific trap: group-based
authorization needs the scope attached to the provider's property mappings *and* requested by
the client, and if either half is missing **the rules silently deny everyone**. A silent deny in
the auth path is the worst failure mode there is. So when claims are enabled, `porte` verifies at
startup that the scope was granted and a `roles` claim actually arrived, and refuses to boot with
a message naming the missing half. Written once, six apps protected.

**Freshness — the part that is actually hard.** Sessions are opaque and long-lived; an Authentik
group change does not reach them. For comparison, Entra's group-membership changes take up to a
day to apply without a dedicated mechanism, which is not acceptable for *removing* a right.
Three mechanisms, and v0.1 ships the first two:

1. **TTL on claims.** `profile_synced_at` already exists to rate-limit profile refreshes; the
   same pattern covers claims, refreshed lazily against the IdP using the refresh token every app
   already holds. Short TTL, no new infrastructure, covers the routine case.
2. **Back-channel logout.** `POST /auth/backchannel-logout` validates the logout token and
   deletes that user's sessions. This is the security-critical case — deactivation, admin
   session termination — and per authentik's own documentation it is the only mechanism that
   covers it. One endpoint here, one URL per application there.
3. **Antenne**, later. A group-change event invalidating cached claims is the suite-native version
   and composes with P4, but 1 and 2 cover the need; do not build it for this.

Because sessions are opaque and claims are stored server-side (`porte_identities.claims`, JSONB),
there is no token-size ceiling — the *group overage* problem that forces Entra to invent a
`hasgroups` claim and a directory round-trip simply does not arise here.

**What `porte` hands the app, and where it stops:**

```go
id, _ := porte.From(ctx)
id.HasRole("admin")   // typed, parsed, kept fresh — meaning is the app's business
```

No `RequireRole` middleware for IdP roles, and no policy engine. The app writes its own guard in
five lines, which is the smaller of the two options §7 Q4 posed and the one that keeps all three
production role models working untouched.

**Do not confuse this with `porte/spaces` (v0.5).** Space membership is app-local data — the
spaces live in the app's own database — so `spaces.Guard.Require(ctx, userID, spaceID, role)`
resolves membership, not IdP claims. Two different axes that must not be merged: a claim says what
Authentik thinks of you globally; a membership says what this app's data says about you in one
space.

**What not to build.** If centralised fine-grained authorization ever becomes a real requirement,
do not invent an API for it: the OpenID **AuthZEN Authorization API 1.0** went Standards Track in
March 2026 with wide vendor interop, and Keycloak already ships experimental support. `porte`'s
job in that world is to be a good PEP, never a PDP. That is a v2 conversation, recorded here so
nobody starts a home-grown authorization service in the meantime.

**The gap claims will never close.** A claim describes only the user who just logged in — it
never tells an app who *else* exists. Today every app knows only the users who have signed in at
least once, which is why "share this file with a colleague" and "invite someone to a space" have
no directory to draw on. The standard answer is authentik's **SCIM provider** (backchannel
provisioning, pushed on change plus an hourly resync), which would give each app a local, fresh
copy of users *and* groups. Deliberately **not** in scope: it is a second protocol and an endpoint
per app, and mechanisms 1–2 already solve freshness. Recorded as the known direction for the day
an invite screen needs to list people who have never logged in.

### The app owns its *profile* — `porte/pg` offers the identity tables

An earlier revision said "`porte` must never own the users table — apps have genuinely
different user columns." Read against source, that premise is false: the six `schemas/user.go`
share **12 byte-identical columns**, plus 0 (Courrier, Agenda, Vision) to 4 (Sablier) business
columns. The identity/profile split already exists in every app, hand-written.

The shared core, identical in all six: `id`, `email` (uniqueIndex), `name`, `avatar_url`,
`avatar_source`, `oidc_picture_url`, `password_hash`, `oidc_access_token`, `oidc_refresh_token`,
`oidc_token_expiry`, `profile_synced_at`, `created_at`. Everything else, per app:

| App | Business columns beyond the core |
|---|---|
| Courrier, Agenda | none — **byte-identical files** |
| Vision | none (roles live in `workspace_members`) |
| Plume | `reminder_interval_days` |
| Nuage | `color`, `is_admin` |
| Sablier | `color`, `rate`, `rate_type`, `workday_hours` |

Note what is *missing* from the core: no `updated_at` anywhere (ROADMAP P4.3) and no `facile_id`
outside Nuage (P4.1). ~~and **no `sub`**~~ — **corrected 2026-08-07**: `oidc_subject *string`
with a `uniqueIndex` is now present in all six, so the shared core is **thirteen** columns, not
twelve. Six of them — the `oidc_*` group plus `profile_synced_at` — are `porte` plumbing that
has no business being in an app's domain table at all.

**One more delta the earlier reading missed:** Courrier and Agenda **encrypt the OIDC access and
refresh tokens at rest** (`service.encryptToken` → `crypto.Encrypt` with an app key, falling back
to plaintext when no key is configured); the other four store them in the clear. `porte` hands
the store plaintext and the store decides — encryption at rest is a deployment property and
`porte` has no key management to offer. Recorded so the difference is a choice rather than a
surprise during extraction.

So `porte/pg` ships, as the *default* implementation of the interfaces:

```
porte_users        id, facile_id, email (unique), email_verified, name,
                   avatar_url, avatar_source, created_at, updated_at
porte_identities   user_id, provider, subject, password_hash,
                   access_token, refresh_token, token_expiry, synced_at
                   UNIQUE(provider, subject)
porte_sessions     token_hash PK, user_id, label, created_at, last_used_at, expires_at
porte_login_codes  code_hash PK, user_id, expires_at
```

The app keeps its business columns in its own `user_profiles` (or wherever it likes),
`user_id PK REFERENCES porte_users(id)` — existing FKs keep pointing at the same `int64`.
Schemas are constants applied through the app's own migrations, never at boot (`caisse/pg`
precedent). Matching on callback: `(provider, subject)` lookup; fallback linking by email only
when `email_verified` is true.

The interface remains the escape hatch, and is all `porte` itself compiles against:

```go
type UserStore interface {
	UpsertFromOIDC(ctx context.Context, claims Claims) (userID int64, err error)
}
```

An app with an exotic user model implements it and ignores `porte/pg` entirely. Fields the
current implementations write on upsert, as a checklist for `Claims`: subject, provider, email,
email_verified, name, picture URL, plus `avatar_url` / `avatar_source` / `oidc_picture_url` /
token material / `profile_synced_at`. `profile_synced_at` exists to rate-limit profile
refreshes — keep it.

**`UpsertFromOIDC` has side effects in real apps, and the interface must tolerate that.** Nuage's
version assigns a display colour through `usercolor.NextAvailable` and makes the first user ever
created an admin (`IsAdmin: userCount == 0`). That logic is product behaviour, it stays app-side,
and it is precisely why the app implements this method rather than `porte` owning the write.
`porte` calls it once per successful callback and cares only about the returned id.

### Roles are a hook, not an enum

`porte` must not arbitrate the role model. Today: Nuage uses `IsAdmin bool` with first-user-
admin, Vision uses workspace-scoped roles, the TS family uses a `USER`/`ADMIN` enum. All three
have to keep working. How the IdP's view of a user reaches the app without `porte` interpreting
it is §5c.

---

## 6. Security — fix these while extracting

Read from `Nuage/apps/api/modules/auth/oidc.go` on 2026-08-07. The existing flow is decent and
should be preserved, with three additions. Writing them once here instead of six times is a
concrete argument for the library.

**Already good, keep it:** `state` is random per request, stored in a cookie that is `HttpOnly`,
`Secure` when the success URL is https, `SameSite=Lax`, TTL-bounded, compared on callback and
cleared immediately after. The ID token is verified through `go-oidc`'s verifier.

**Missing — add:** *(re-measured 2026-08-07 during the §11 diff; the list below is what is
actually still missing, and two items that were on it are already done.)*

1. ~~**Match on `sub`, not email — the most severe item on this list.**~~ **Already fixed in all
   six**, and it is the HEAD commit of every `modules/auth/`. The lookup is `oidc_subject` first,
   with an email fallback taken only when `emailClaimTrusted(claims.EmailVerified)` allows it —
   an absent claim counts as trusted *(**changed in porte v0.2.6**: an absent claim no longer
   counts as trusted, and `Config.TrustEmailWithoutVerifiedClaim` is the opt-in — the six apps
   still carry the old rule in their own copies)*, an explicit `false` does not, and one app (Plume) rejects
   unverified email outright before upserting. `porte` **preserves** this logic; do not treat it
   as new work, and do not regress it. `porte_identities(provider, subject)` in §5 makes it
   structural rather than conventional, which is still worth doing.
2. **PKCE (S256).** Still missing everywhere: `AuthCodeURL(state)` is called with no code
   challenge in all six. Add `oauth2.S256ChallengeOption`, with the verifier carried in a cookie
   alongside the state.
3. **Nonce.** Still missing everywhere. No nonce is generated or checked against the ID token
   claim. Generate one per request, carry it with the state, and verify it after `Verify` returns.
4. **Constant-time state comparison.** Five of six use a plain `!=`. **Plume already uses
   `subtle.ConstantTimeCompare`** — copy Plume's line, it is the reference.
5. **Kill the `localStorage` token.** Three of six still do this — Nuage, Vision and Sablier put
   the token in the URL fragment. **Courrier and Agenda already ship the cookie** (see §5). The
   cookie transport removes both halves at once for the remaining three.
6. ~~**Drop the query-parameter credential paths.**~~ **Done 2026-08-07, outside this repo.**
   Courrier accepted `?token=` (deprecation warning, zero consumers) and Vision an undocumented
   `?api_key=`; both removed. `porte`'s middleware reads the cookie and the `Authorization`
   header, and nothing else.

   **The exception that must not be "fixed":** Vision's `?token=` on
   `GET /events/{siteId}/live` stays, because `EventSource` cannot set headers. Nuage has the
   same shape for file downloads, where a plain navigation cannot either. These are browser
   constraints, not shortcuts — and **the cookie transport in §5 retires both for free**, since
   `EventSource` and download navigations send cookies automatically. That is a second argument
   for the cookie beyond XSS, and it is why adopting `porte` deletes those paths rather than
   porting them.

**Non-negotiable:** do not hand-roll protocol or crypto. `go-oidc` for verification,
`golang.org/x/oauth2` for the flow, `argon2` for hashing. This is the one place custom code is
unacceptable.

---

## 7. Open questions — settle before coding the affected part

1. ~~**CLI token flow.**~~ **Settled 2026-08-07** — read Plume's implementation: the callback
   parks a one-time code in a `sync.Map`, the CLI exchanges it. Right flow, wrong store (dies on
   redeploy, breaks at two replicas). v0.1 owns it, DB-backed. See §5b.
2. ~~**`/auth/me` and `/auth/password`**~~ **Settled 2026-08-10 — split, and the split is not where
   the question assumed.** Read from all eight adopters in production rather than argued.

   **`/auth/me` and profile editing are app-side, permanently.** There is nothing to share. The
   path disagrees (`/users/me` in five, `/auth/me` in three), the verb disagrees (`PATCH` in five,
   `PUT` in three), and the payload has no intersection beyond id/email/name: Sablier carries
   `rate`, `rate_type` and `workday_hours`, Nuage and Sablier a `color`, Boutique a `role`, Plume a
   `reminder_interval_days`, Journal an `is_admin`. `porte` has no idea what a user looks like,
   which was always the reason, and the measurement agrees with it.

   **The credential half is `porte`'s, and it was missing.** Three security properties, eight apps,
   eight different subsets, and **no app had all three**:

   | | current password required | re-keys the identity on an address change | ends sessions after a change |
   |---|---|---|---|
   | Journal | n/a — no update route | n/a | n/a |
   | Sablier | no | **no** | no |
   | Boutique | no | yes | yes |
   | Courrier | no | yes | no |
   | Agenda | no | yes | no |
   | Nuage | yes | yes | no |
   | Vision | yes | yes | no |
   | Plume | yes | yes | no |

   That is not eight teams being careless. Column two was impossible to get right through the
   contract — there is no delete and no update on `IdentityStore` — so five apps wrote raw SQL
   against `porte_identities` and two did not. Column one was optional because one method served
   both "add a first password" and "replace one". Both are `porte` bugs wearing app clothing, and
   v0.3.0 closes them: the address stops being a credential key (§5), and `ChangePassword` is a
   separate method from `SetPassword`.

   So the answer to the original question is neither of its two options. `/auth/password` as a
   *route* stays app-side with `/auth/me`; the *operation* behind it belongs here.
3. ~~**Journal's bcrypt.**~~ **Settled 2026-08-07** — the only bcrypt hit in Journal is the
   fixture string `"$2y$n$nope"` in `authcrypto/crypto_test.go`. Argon2-only. Non-issue.
4. ~~**Role hook shape.**~~ **Settled 2026-08-07 — the smaller option.** Claims ride typed but
   uninterpreted on the identity; the app writes its own guard. `porte` ships no `RequireRole`
   for IdP roles and no policy engine. Full design, including the freshness mechanisms and the
   startup guard, in §5c.
5. ~~**CSRF header convention.**~~ **Proposed 2026-08-07: `X-Facile-CSRF`, any non-empty value.**
   No app sends a CSRF header today, so there is nothing to reconcile — the name is free. The
   header's *presence* is the entire signal: a browser will not attach a custom header to a
   simple cross-site request without a preflight, so there is no token to mint, distribute or
   rotate. Frozen as `porte.CSRFHeaderName`. Reject only if a frontend already reserves the name.
6. ~~**Claim refresh TTL.**~~ **Proposed 2026-08-07: five minutes**, deliberately the same number
   as the existing `profile_synced_at` rate limit (`time.Since(record.ProfileSyncedAt) < 5*time.Minute`
   in Nuage). One refresh cadence, not two: a revoked role stops mattering within five minutes,
   and the IdP sees at most one refresh per user per five minutes instead of one per request.
   Frozen as `porte.DefaultClaimsTTL`, overridable per app via `Config.ClaimsTTL`.

---

## 8. Package layout

```
porte/          the contract. types, interfaces, wire shapes. standard library only
porte/session   the credential: issuance, the cookie, the middleware, POST /auth/logout
porte/oidc      the engine: the flow, the six OIDC routes, the avatar guard
porte/local     email and password: argon2id, register, login, set-password
porte/pg        the identity tables and the four stores. database/sql only, no ORM
porte/avatarfs  a filesystem AvatarStore and the handler that serves it
porte/spaces    v0.5. membership and role resolution. no models, no routes, standard library only
porte/spaces/spacestest  the conformance suite an adopter runs against its own Store
```

**`porte/session` extracted 2026-08-09, and it is the decision this section got most wrong.**
v0.1 put session issuance, the cookie, the authenticator, the middleware and `POST /auth/logout`
inside `porte/oidc`, because OIDC was all there was and the layering question never came up. The
first adoption priced it: Journal has its own password form, and an app with a password form could
not mint a `porte` session or set `porte`'s cookie, so half its logins carried an `HttpOnly`
cookie and the other half kept a token in `localStorage` — the exact split the cookie transport
was adopted to end. Five of the six remaining apps have a password form too, so shipping them onto
v0.1 would have spread it rather than exposed it.

The general form is worth stating, because it will happen again: **the OIDC package owned the
session, so the session inherited OIDC's preconditions.** A package that owns a credential must
sit below every way of obtaining one. `porte/local` therefore depends on `porte/session` and not
on `porte/oidc` — an app that wants only passwords must not compile an OIDC client, which is the
whole reason the manager was extracted rather than merely re-exported.

One manager, not one per kit. Two over one table would each keep their own idea of the clock and
the cookie, and `oidc.New` refuses a kit whose config disagrees with its manager's about the
redirect URL, the success URL or the TTLs: those decide `Config.HTTPS()`, which decides whether the
cookie is `Secure` and `__Host-` prefixed, and nothing else would fail until an attacker noticed.

This also cost a breaking change one version after freezing the contract — `oidc.Deps.Sessions` is
a `*session.Manager` now — which is the argument for `v0.x` continuing a while longer.

**Corrected 2026-08-07.** This section originally put the engine in the root package. That
contradicts the zero-dependency decision taken with the contract in §11: an app implementing
`UserStore` would then compile against `go-oidc`. The literal promise is unreachable inside one
module — `go.mod` requirements are module-scoped, so any import of any `porte` package pulls them
all — but the *layering* is worth keeping and costs one import line in `main.go`. Only that file
imports `porte/oidc`.

Dependency rules, following `caisse`:

- `tronc/errors` for the suite error envelope, so `httpjson.WriteError` maps failures to the
  right status with no glue in the app. This couples `porte` to the suite deliberately.
  **Exception, v0.2.4:** the two handlers a browser navigates to — `/auth/oidc` and its
  callback — redirect to `Config.LoginFailure(reason)` instead. An error envelope is for a
  caller that parses it, and there the caller is a person looking at an address bar.
- **No GORM.** All six apps use it, but forcing it is a heavier commitment than anything the
  suite shares today, and it is what pushed `tronc` to split `migrate` and `testdb` into
  separate modules. `database/sql` keeps everything in one module.
- ~~Go floor 1.24~~ — **settled 2026-08-07: 1.25.** Not by consistency but by `go-oidc` v3.20,
  which requires `go >= 1.25`. The principle that a library floors low still holds; it just has
  nothing to bite on here, since the apps moved to 1.25 in Phase 2 and `tronc/migrate` already
  declares 1.25.7. `mise.toml` and `.github/workflows/ci.yml` pin the same number.

---

## 9. Validation and rollout

**Before tagging v0.1.0:** prove it on the **e-commerce demo** (greenfield, no live sessions to
break, and it forces the works-outside-the-suite test — a non-Authentik `OIDC_ISSUER` — on day
one) and **Nuage** (the extraction source, so the most honest feedback on whether the
abstraction fits). This is the sequence that made `tronc` right — written, proven on one,
rolled out after. *(An earlier revision named Comptoir here; Comptoir does not exist.)*

The cookie transport touches the six SvelteKit frontends as well as the six backends — the one
place this plan costs more than plain extraction. It rides the frontend rewrite (F track), and
forced logout is already accepted, so the marginal cost is one deleted token-handling file per
frontend.

**Then** the remaining five, in one pass each. Change `OIDC_ISSUER` to `sso.facile.studio` in
the same commit that adopts the library: the app's env is already being edited, so it costs one
extra line. Done separately it is a second tour of ten repos.

Ten backends carry OIDC config in total — six Go, three TS (Glouton, Ardoise, Perception), plus
Opus on better-auth. The TS siblings are out of scope for this repo but their issuer URL is not.

**Perception is a consumer already waiting.** Its Go rewrite deliberately built a seam for this
library: `apps/api/internal/identity/` holds an `Authenticator` interface that feature modules
depend on, `modules/auth` is reachable from `main.go` alone, and `apps/api/seam_test.go` is an
architecture test that *fails* if any other package imports the auth module. Its `ROADMAP.md` §7
says the adoption is "delete `modules/auth`, add the dependency, construct it in `main.go`, done."
Treat it as the third pilot: it is the only repo whose structure will tell you whether the
interface shape is right, because it was designed against the idea rather than extracted from it.
(That seam was written under the working name `sésame`; renamed to `porte` on 2026-08-07.)

---

## 10. Repo conventions

Same as `tronc` and `caisse`:

- `scripts/check.sh` — gofmt, vet, `go test -race`, golangci-lint. Depends on nothing but a
  `go`, and is not invoked through mise on purpose.
- lefthook runs it as a `pre-push` job, and `mise install` installs the hooks. `lefthook.yml`
  also pulls the shared conventional-commit check from `FacileStudio/hooks`, pinned by tag.
- CI on Go 1.25 exactly — the floor the module documents — plus the PostgreSQL 16 service that
  `porte/pg`'s tests need. Both live in `.github/workflows/ci.yml`, with
  `PORTE_TEST_DATABASE_URL` set so the pg tests never skip there.
- Docs follow `~/.mycelium/memory/standards/docs.md`: `README.md` plus `docs/` with
  `architecture.md`, `configuration.md`, `development.md`, `api.md`. English, no badges, no
  emoji.
- Semver from `main`. While on `v0`, a breaking change bumps the minor.
- Record decisions in `CHANGELOG.md` with their reasoning, as `caisse` does — the reasoning is
  the part that stops a future session from undoing a deliberate choice.

---

## 12. Prior art, and what it actually says

Consulted 2026-08-07. Listed because several of these argue *against* parts of this document,
and a future session should be able to check the reasoning rather than trust it.

**On owning the identity table.** [Supabase](https://supabase.com/docs/guides/auth/managing-user-data)
splits `auth.users` (owned by GoTrue) from `public.profiles` (owned by the app) — the same
boundary as §5, validated at scale. Read its *reasons* critically though: it cites "Supabase does
not allow adding columns to auth.users" and "the auth schema is not publicly accessible", which
are constraints of a hosted multi-tenant service, not design conclusions. The shape transfers;
the argument does not, so §5 makes its own.
[Ory Kratos](https://www.ory.com/docs/kratos/manage-identities/identity-schema) reaches the same
boundary from the opposite direction: identity owns traits, and the guidance is to "avoid complex
identity schemas" and keep app-specific data out.
[better-auth](https://better-auth.com/docs/concepts/database) tests the other road — one owned
table extended via `additionalFields` — and the friction is documented: you extend your own schema
to match, and [custom tables need a plugin](https://github.com/better-auth/better-auth/discussions/6717).
That is the in-house precedent too, since Opus and Hottake run it.

**The honest counter-argument.** [Beekeeper](https://www.beekeeperstudio.io/blog/one-to-one-database-relationships-complete-guide)
and [O'Reilly](https://www.oreilly.com/library/view/microsoft-sql-server/9780133408539/ch34lev2sec4.html)
are firm that needless 1:1 splits of closely-related data are a mistake. They are right, and it is
why §5 justifies the split as a **module boundary**, never as normalisation — there is no
performance win in separating 12 columns from 3, and claiming one would be dishonest.

**On account vs identity.** [Red-gate](https://www.red-gate.com/blog/user-authentication-module/) —
keep account and identities separate so one human can hold several authentication factors. That is
where `porte_identities` comes from.

**On sessions in the browser.** [OAuth 2.0 for Browser-Based Apps](https://datatracker.ietf.org/doc/draft-ietf-oauth-browser-based-apps/12/),
[OAuth 2.1 / RFC 9700](https://aembit.io/blog/oauth-2-1-guide-migration-security/) and the
[OWASP Session Management cheat sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
agree: never `localStorage`; `HttpOnly; Secure; SameSite` cookies, or a BFF. The suite's
mono-container already provides the same-origin backend that pattern needs.

**On CLI login.** [RFC 8628](https://www.rfc-editor.org/info/rfc8628/) plus two practitioner
write-ups ([abgeo](https://www.abgeo.dev/blog/cli-authentication-the-right-way/),
[Logto](https://dev.to/logto/getting-cli-authentication-right-the-complete-guide-to-all-4-methods-1mnf)):
device flow by default, refresh material in the OS keychain rather than a dotfile, `gh auth login`
as the reference implementation.

**On claims and freshness.**
[authentik property mappings](https://docs.goauthentik.io/add-secure-apps/providers/property-mappings/)
— including the trap where a half-configured scope silently denies everyone;
[authentik back-channel logout / SLO](https://goauthentik.io/blog/2025-10-21-authentik-now-supports-single-logout/)
— the only mechanism covering administrative termination;
[Entra continuous access evaluation](https://learn.microsoft.com/en-us/entra/identity/conditional-access/concept-continuous-access-evaluation)
— why an unrefreshed group membership can be a day stale, and the *group overage* problem that
server-side claim storage avoids entirely.

**On what not to build.** [AuthZEN Authorization API 1.0](https://openid.github.io/authzen/) went
Standards Track in March 2026, with [Keycloak shipping experimental support](https://www.keycloak.org/2026/05/authzen-as-experimental-feature).
If centralised fine-grained authorization is ever needed, `porte` is a PEP against that, never a
bespoke PDP. And [authentik's SCIM provider](https://docs.goauthentik.io/add-secure-apps/providers/scim/)
is the answer to the directory problem claims cannot solve — noted, deliberately unbuilt.

**On library design.** [authboss](https://github.com/volatiletech/authboss) ("plug it in, configure
it, start building") for the ambition, [goth](https://github.com/markbates/goth) for the shape:
small interfaces, implementations provided.

---

## 11. First step — done 2026-08-07

Not code. Read the six `modules/auth/` implementations and diff them properly — `oidc.go`,
`service.go`, `middleware/auth.go` — then write the frozen contract of §5 as Go types and get
it reviewed before anything is built on top of it.

**Done.** The diff produced the corrections marked *2026-08-07* throughout this document, and
the contract is `porte.go`, `identity.go` and `session.go` — types, interfaces and wire shapes
only, **standard library only, zero dependencies**. That constraint is the point: an app
implementing `UserStore` must not inherit `go-oidc`, `oauth2` or a database driver from `porte`.
It is why `Claims` carries plain fields and `TokenSet` exists instead of an `*oauth2.Token`.

Two shape changes the contract makes deliberately, both worth arguing about at review:

- **`Identity.UserID` is an `int64`**, not the decimal string every app passes around today.
  The conversion moves to the edge instead of being repeated per handler, and it matches
  `porte_users.id` and the existing `int64` foreign keys. The wire keeps `user_id` as a string
  in `ExchangeResponse`, because to a CLI it is an opaque identifier and breaking it buys
  nothing.
- **The middleware yields a typed `Identity`**, replacing the current
  `Authenticate(ctx, string) (string, any, error)` plus an `interface{ GetEmail() string }`
  assertion in every app. That assertion is a runtime failure mode sitting in the auth path.

The rest is extraction. This part is the one that is expensive to get wrong.

### The review the contract asked for — done 2026-08-07, by implementing it

Writing `porte/oidc` and `porte/pg` against the frozen contract was the review, and it is the
only kind that finds this class of problem. Three shapes were wrong:

- **`LoginCode.SessionID` could not work.** The session row stores only a hash, so a code that
  pointed at a pre-created session could never yield a usable token at exchange time without
  keeping the plaintext at rest. Removed; the session is created at exchange time. The stated
  justification — avoiding a second write path that could half-succeed — assumed a write more
  complicated than the single insert it actually is.
- **`AvatarStore.Put(userID, ...)` had an unsolvable ordering problem.** The avatar URL has to
  reach the app's user row; the app writes that row in `UpsertFromOIDC`; `Put` wanted a user id
  that does not exist until that call returns. Now keyed on the identity, so the fetch runs
  first and the URL rides into the upsert on `Claims.AvatarURL`. One write, no second interface.
- **`Config.Validate`'s issuer check tested nothing.** `url.Parse` accepts almost any string.
  Tightened to require an absolute http(s) URL.

The two shape changes flagged here for argument — `Identity.UserID` as an `int64` and the typed
`Identity` from the middleware — both survived. They made the implementation smaller, which is
the evidence that was being asked for.

## 13. Next — proving it

### The flow has been walked — done 2026-08-08

This section used to say the three interesting paths could not be covered honestly, because
PKCE, the nonce and the back-channel logout token are assertions about what the *provider*
does. That is true of a fake that echoes whatever it is handed, and it is what made the
statement feel like a law rather than a property of the fake.

`oidc/flow_test.go` is a conformant in-process issuer instead. It signs RS256 tokens behind a
real JWKS, and its token endpoint **enforces** PKCE, the redirect URI and client authentication
rather than assuming the client sent them. The browser login, the CLI code exchange, the
back-channel logout and the roles claim are walked end to end against it; a kit that dropped its
verifier, reused a nonce or accepted an ID token at the logout endpoint fails. What remains
untested against a real Authentik is now Authentik's own conformance, not `porte`'s behaviour —
a much smaller and much later question.

A security review ran against the result and found seven things worth fixing; they are recorded
in CHANGELOG.md with their reasoning. Three changed the contract and are noted in §5 and §11:
`Config.SessionIdleTTL` and the seven-day idle window, `IdentityStore.MarkRolesSynced`, and the
`__Host-` cookie prefix with the bare names still read for migration.

### Still unproven — recorded 2026-08-09, after the first adoption, revised the same day after the second

Journal and Sablier run `porte` against the suite's Authentik, so the browser login, the callback,
the upsert, the cookie and the password login are walked by real users. One of the two gaps below
has since closed; what closed it is recorded because the closing is what the rest of the suite
inherits.

**The CLI flow has never run against a real Authentik.** `?flow=cli`, the loopback `?port=N`
callback, the one-time code and `POST /auth/oidc/exchange` are walked end to end in
`oidc/flow_test.go` against the conformant in-process issuer, and no further. Journal has no CLI,
so the first app that does is the first real exercise.

This section used to say the untested part was "Authentik's behaviour on a redirect URI pointing
at `127.0.0.1` with a variable port, a provider configuration question the flow test cannot ask."
**That is wrong, and the design is why.** The port never reaches the IdP: `?port=N` is validated
and parked in `porte`'s own pending-state cookie (`oidc/cookie.go:25`, `handlers.go:85`), the
authorization request carries the app's single fixed `OIDC_REDIRECT_URL`, and the redirect to
`127.0.0.1:N` is issued by the *app* after the callback, once the code is minted
(`handlers.go:252`). Confirmed against the deployed provider: Sablier's has exactly one redirect
URI, `https://sablier.facile.studio/api/auth/oidc/callback`, in `STRICT` matching mode — and the
CLI flow does not need a second one. A loopback URI in the provider would be the design that
hands the CLI an authorization code worth something on its own, which is the design this one was
chosen over.

So the exercise is narrower than it looked, and worth stating so the result is not over-read:
`sablier-cli` (`src/login.rs:39`) against a Sablier that is `SSO_ONLY=true` proves the one-time
code survives a real provider's timing and a real browser's redirect chain. It cannot fail for
provider-configuration reasons, because the provider is not configured for it at all.

**Back-channel logout is wired — closed 2026-08-09.** This section used to say the endpoint could
not work, because the deployed provider was Authentik 2025.6.3 and the field to configure a
back-channel logout URI did not exist before 2025.10. The provider was taken to **2026.5.6** that
day, and the field exists: `OAuth2Provider` carries `logout_method` (`backchannel` /
`frontchannel`) and `logout_uri`.

Both `porte` adopters are pointed at their endpoint —
`https://journal.facile.studio/api/auth/backchannel-logout` and the Sablier equivalent. The other
eleven providers carry the default `logout_method=backchannel` with an **empty** `logout_uri`,
which means they receive nothing; that is correct rather than pending, because they have no
endpoint to receive it until they adopt `porte`. It is also the concrete per-app benefit to lead
with when proposing the next adoption: **disabling an account in Authentik now reaches sessions it
already issued**, which for the eleven is still not true.

Two things about the upgrade are load-bearing for `porte` and are not Authentik trivia:

- **Sessions minted before the upgrade become unreadable** and 500 inside Authentik's own login
  middleware. Anonymous requests pass, so discovery and health checks stay green while a real
  user sees a server error. Purging `Session` and `AuthenticatedSession` is part of the upgrade,
  not an optional tidy-up. This is Authentik's session table, not `porte`'s — a `porte` session
  survives, which is exactly the decoupling the opaque-token design was for.
- **Authentik 2026.x no longer serves `/media/` over HTTP at all.** Every Facile app stores the
  absolute `https://porte.facile.studio/media/user-avatars/<uuid>.<ext>` in `oidc_picture_url` and
  receives it as `Claims.AvatarURL`, so the whole suite's avatars died at once and are now served
  by an nginx sidecar on the same volume, behind a higher-priority Traefik route. The lesson for
  `porte` is narrower than "pin the IdP": an absolute URL to the IdP's file storage is a contract
  with an internal detail that has now proven it changes between versions, which is an argument
  for `porte/avatarfs` — apps that store the bytes rather than the URL were unaffected.

What has *not* changed is the offboarding runbook for an app that has not adopted `porte`:
revoking in each app, and for `porte` apps the mechanism remains `session.Manager.RevokeUser`,
which is still what a manual revocation goes through when the IdP is not the trigger.

### Done since this list was written — 2026-08-09

- ~~**The roles scope mapping in `authentik-config`**, the producing half of §5c.~~ The
  `Facile roles` scope mapping exists on the provider (`scope_name = roles`), it is granted to
  Journal's application and to no other, and **Journal runs `OIDC_CLAIMS_SCOPE=roles` in
  production**. The claim arrives from a real provider, so §5c is no longer paper on the
  producing side either.

  **It carries a trap that must be settled before any app authorizes on it.** The mapping is
  built from the user's groups, and Facile groups are *per-app membership markers*
  (`journal-users`, `nuage-users`, `coffre-users`, …) rather than roles — a real user sits in
  around twelve. So the `roles` claim is a cross-app membership list, and an app that authorizes
  on it is reading other apps' access as its own. Nothing reads it today: Journal requests the
  scope, `porte` stores it on the identity, and no handler consults it. That is a safe place to
  stand, not a resolution. The two exits are Authentik's built-in `entitlements` mapping, which
  is already filtered to the application the token is for and needs RBAC entitlements defined,
  or filtering the groups by app prefix in the mapping. Decide before the first `RequireRole`,
  not during.

  Related, and found only by upgrading: `request.user.ak_groups` is deprecated in 2026.2 and the
  mapping has been migrated to `request.user.groups`. The startup guard in §5c is aimed at
  exactly this failure — a scope granted with nothing behind it — and this is the first evidence
  that the shape it guards against occurs in practice.

- ~~**Back-channel logout blocked on the provider version.**~~ See above: Authentik is on
  2026.5.6 and both adopters receive logout tokens.

- ~~**`authentik-config` is version control.**~~ It is not. The directory holds the two policy
  scripts and a README, and the git repository in it **has no commits at all** — so the mirror
  the avatar recipe tells you to copy from is not actually tracking anything. Its README also
  still documents `FILE_PATH = /media/user-avatars/`, while the volume now mounts at `/data`, so
  a new avatar upload would land in the container layer and vanish on the next deploy. Verify
  against the live policy object before trusting either file. This is not a `porte` change, but
  it is the thing most likely to silently break the avatar path `porte` depends on.

### Still ahead, in order

1. **The Sablier CLI login against the real provider** — the last unwalked path in §5b, and now
   a ten-minute test rather than a missing consumer. See the CLI note above for what it does and
   does not prove.
2. **`docs/architecture.md`** — both conditions are met: a production request path, and a second
   adopter of the opposite shape to the first. Drawn from Journal and Sablier together it
   documents `porte`; drawn from either alone it documents that app.
3. **Nuage**, the extraction source, then **Perception**, whose `internal/identity/` seam and
   `seam_test.go` were designed against this idea rather than extracted from it — it is the repo
   whose structure will say whether the interface shape is right. Nuage is also where the
   `roles`-versus-`entitlements` question above stops being theoretical, because it has real
   per-space authorization.
4. **The e-commerce demo**, greenfield and outside the suite, which forces the non-Authentik
   issuer test on day one.
5. **v0.5 `porte/spaces`** — shipped, and the two conditions this item set turned out to apply to
   different halves of it. The *models* were what Nuage and the `enveloppe` space identity had to
   decide, and the package no longer ships models, so neither gates it; §4 records why. The
   *guard* was never in question — seven apps had already written it, three wrong — so it was
   read out of them rather than designed ahead of them. First adopter still decides the ergonomics:
   the shape to watch is whether `Store` over an app's own table is genuinely five lines, and
   whether `spacestest.Conformance` fails on the three apps that got an invariant wrong.
