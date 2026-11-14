# porte

The authentication kit for the [Facile Suite](https://facile.studio). The OIDC plumbing every
Facile API needs and none of them should be re-writing.

**In production, in two apps.** [Journal](https://github.com/FacileStudio/Journal) and
Sablier run on it against the suite's Authentik:
discovery, PKCE, nonce, callback, upsert and the session cookie are walked by real users, not
only by tests — and since v0.2 their password logins land in the same session, the same cookie
and the same logout as their federated ones. Journal is also the first app running with
`OIDC_CLAIMS_SCOPE` set, so the roles claim arrives from a real provider and not only from the
flow test. Each adoption has moved the contract: the first priced v0.1's layering — the session
belonged to the OIDC package, so an app with a password form could not mint one — and the second
produced v0.2.1 and v0.2.2 inside two days. Five apps still have their own copy. Read
[SPEC.md](SPEC.md) before writing any code — it carries the decisions, the contract, and the
reasoning behind both.

## What it does

- Mounts `/auth/oidc` and its callback against any OIDC provider, Authentik today, with PKCE, a
  nonce and a constant-time state comparison — all three missing from the six apps it replaces
- Verifies the ID token, upserts the user through an interface the app implements, issues a
  session
- Signs people in with an email and a password too — argon2id at the parameters already in the
  suite's databases, so adopting it is a code change and not a password reset — and lands them on
  the same session as a federated login, because a human may hold both and is one account
- Serves `/auth/config` so a frontend knows whether SSO is available and whether it is mandatory
- Carries the suite's `SSO_ONLY` and `OIDC_*` environment conventions
- Keeps the role model pluggable, so `IsAdmin`, workspace roles and enum roles all still work
- Stores sessions as hashed opaque tokens — `HttpOnly`, `__Host-` prefixed cookie in browsers,
  `Bearer` for CLIs and API tokens — with a `database/sql` implementation in `pg/`
- Retires a browser session nobody has used for a week, inside the thirty-day absolute lifetime,
  and leaves CLI and API tokens alone because a nightly job is idle by design
- Gives every CLI the same login: browser opens, user signs in, a one-time code comes back.
  The listener is included, in `porte/loopback`, so five CLIs stop each writing their own
- Signs a CLI in over SSH too: `porte/devicelogin` runs the device grant against Registre and
  trades the token at the app's device exchange, with one link to open and nothing to paste
- Draws the two pages it cannot hand back to the app, the code page and a CLI's loopback
  page, with the app's own name and logo, derived from `OIDC_SUCCESS_URL` and needing no
  deployment to set them
- Fetches IdP avatars behind an SSRF guard that checks the address at connect time, closing the
  DNS-rebinding window every existing copy leaves open, and unwraps the IPv6 forms that smuggle
  an IPv4 metadata address past `net.IP`'s own predicates
- Puts those avatars on disk and serves them back, once, instead of five apps each keeping their
  own copy of the same directory-plus-URL-prefix store

## Packages

| Package | What it is | Depends on |
|---|---|---|
| `porte` | The contract: types, interfaces, wire shapes. No behaviour | the standard library |
| `porte/session` | The credential: issuance, the cookie, the middleware, `POST /auth/logout` | the contract, tronc, chi |
| `porte/oidc` | The engine: the flow, the routes, the avatar fetch | `porte/session`, go-oidc, oauth2, tronc, chi |
| `porte/local` | Email and password: argon2id, register, login | `porte/session`, `x/crypto`, tronc, chi |
| `porte/magiclink` | Passwordless sign-in: a one-time emailed link, single use, rate limited, landing in the same session | `porte/session`, tronc, chi |
| `porte/pg` | The identity tables and the stores over them | `database/sql` |
| `porte/exchange` | Per-user RFC 8693 token exchange from the stored Registre refresh token to another app's audience: refreshes serialized per user, rotations persisted, tokens cached, `ErrReconnect` | the contract, tronc |
| `porte/kit` | Opt-in glue for an app with its own `users` table: the request identity, a `UserStore` over that table, `RequireAuth`, the erase callback, the picture backfill, `Mount`, and the opt-in `MountInvitations` user-menu route. See [docs/kit.md](docs/kit.md) | `porte/session`, `porte/oidc`, `database/sql`, tronc, chi |
| `porte/avatarfs` | A filesystem `AvatarStore`: atomic writes, a guarded key, and an `http.Handler` that serves them | the standard library |
| `porte/loopback` | The CLI's half of the CLI login: the loopback listener, the login URL, the pages a browser lands on | the standard library |
| `porte/devicelogin` | The CLI's default login: the RFC 8628 device grant against Registre, then the app's device exchange | the standard library |
| `porte/spaces` | Space membership authorization: the role ladder, `Resolve`, `Require`, `CanLeave`, `AssignableBy`/`AssignableOver`. No CRUD, no routes, no ORM | the standard library |
| `porte/spaces/spacestest` | `Conformance(t, newStore)` and `ConformanceWithLadder`: the invariants, run against an app's own `Store` and its own role vocabulary | `porte/spaces`, `testing` |
| `porte/magiclink/magiclinktest` | `Conformance(t, newStore, newUser)`: single use, supersession and the rate counts, run against an app's own `MagicLinkStore` | `porte`, `testing` |
| `porte/directory` | Nested module: Registre's spaces, members and plan entitlements, pulled as one snapshot and nudged on change, plus the GDPR deletion receiver and a person's pending invitations | the standard library |
| `porte/forfait` | Nested module: plan enforcement over the directory snapshot. `Enabled`, `Limit`, `Check`, fail-open until the first snapshot | `porte/directory` |
| `porte/mcpauth` | Nested module: the MCP resource-server side of Registre tokens. A go-sdk `TokenVerifier`, the RFC 9728 metadata, `Require` with step-up challenges, and the `ai.access` plan gate for third-party clients | `porte/oidc`, `porte/forfait`, go-sdk, tronc |

The contract package depends on nothing outside the standard library, and that is a constraint
rather than a coincidence: an app's stores and domain code never compile against `go-oidc`,
`oauth2` or a database driver. Only `main.go` imports the rest.

`porte/session` sits below both kits rather than inside either, and that split is the whole
shape of v0.2. It was in `porte/oidc` until the first adoption showed what that costs: an app
with a password form could not mint a `porte` session, so half its logins carried an `HttpOnly`
cookie and the other half a token in `localStorage`. `porte/local` therefore depends on the
manager and not on the engine — an app that wants only passwords must not compile an OIDC client.

## Wiring it

One manager over one sessions table, and both kits issue through it. Two managers would each
keep their own idea of the clock and the cookie.

```go
store := pg.New(db)

sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: log})

kit, err := oidc.New(ctx, cfg, oidc.Deps{
	Users:      store.Users(),
	Identities: store.Identities(),
	Codes:      store.LoginCodes(),
	Sessions:   sessions,
	Logger:     log,
})

passwords, err := local.New(local.Config{AllowRegistration: allow}, local.Deps{
	Users:      store.Users(),
	Identities: store.Identities(),
	Sessions:   sessions,
	Count:      countUsers, // the app's, under the app's lock
	Logger:     log,
})

sessions.Mount(router)  // POST /auth/logout, whether or not SSO is configured
kit.Mount(router)       // /auth/config and, when an issuer is set, the OIDC routes
passwords.Mount(router) // POST /auth/login, and /auth/register when registration is open

router.Group(func(r chi.Router) {
	r.Use(sessions.RequireAuth)
	r.Get("/things", handler) // porte.From(ctx) inside
})
```

To seal the OIDC access and refresh tokens at rest, give the store a key. With none, nothing changes.

```go
sealer, err := pg.ParseTokenSealer(os.Getenv("PORTE_TOKEN_KEY"))
store := pg.New(db, pg.WithTokenSealer(sealer))
```


Build the kit and the manager from the same `porte.Config`: `oidc.New` refuses a pair that
disagrees about the redirect URL, the success URL or the TTLs, because those decide whether the
cookie is `Secure` and nothing else would fail until an attacker noticed.

An app whose frontend expects a richer login response than `{user_id, token}` — every existing
Facile app answers `{token, user}`, and `porte` has no idea what a user looks like — skips
`passwords.Mount` and calls `Register` and `Login` from its own handlers. That is the supported
path, not a workaround.

A service that acts for a signed-in user, such as Bulle, presents a short-lived JWT addressed to the
app with a `scope` claim. Set `MachineAudience` to verify it and `ScopePrefix` to gate it:

```go
cfg.MachineAudience = "<this app's client id>"
cfg.ScopePrefix = "courrier"
```

A GET then needs `courrier.read` or `courrier.write`, any other method needs `courrier.write`, and
a token with no `scope` claim gets a 403. `porte.ActorFrom(ctx)` returns the client acting for the
user, for audit lines. Cookie sessions are never gated.

## Plan limits

Ardoise computes each space's plan and Registre serves it in the directory snapshot, so an app
only declares the features it enforces and asks before it creates something:

```go
plans := forfait.NewStore(
	forfait.Feature{Key: "nuage.storage_gb", Kind: forfait.Limit, Default: json.RawMessage(`5`)},
	forfait.Feature{Key: "glouton.access", Kind: forfait.Bool, Default: json.RawMessage(`false`)},
)
plans.Issuer = cfg.Issuer

syncer := &directory.Syncer{Client: client, Apply: func(ctx context.Context, s directory.Snapshot) error {
	plans.Apply(s)
	return applySpaces(ctx, s)
}}

err := plans.Check(forfait.Scope{SpaceID: space.RegistreID}, "nuage.storage_gb", usedGB, addingGB)
var refusal *forfait.LimitError
if errors.As(err, &refusal) {
	// 402 with the suite error envelope and upgrade_url = refusal.UpgradeURL
}
```

An empty `SpaceID` reads the personal scope of `Subject`, never a space: pass the Registre id for
a space's resources and refuse a space that is not linked to Registre yet. Only new creation is
gated; reading, downloading and deleting never are. `adding == 0` always passes, so round counts up
(a 1 KiB upload against a GB limit adds 1). `Check` and the create are not atomic: an app that
needs strictness checks again inside the transaction that creates the resource.

Until a snapshot with a non-zero `EntitlementsRevision` arrives the store allows everything and
logs one warning; after that it keeps the last values it applied, through any outage, and refuses
an older snapshot with a warning: a lower revision, or a lower directory version at the same
revision.

The store lives in memory, so persist it to avoid failing open again after a restart:

```go
saved, err := loadPlans(ctx)
if err != nil {
	return err
}
if err := plans.Import(saved); err != nil {
	return err
}

syncer := &directory.Syncer{Client: client, Apply: func(ctx context.Context, s directory.Snapshot) error {
	plans.Apply(s)
	state, err := plans.Export()
	if err != nil {
		return err
	}
	if err := savePlans(ctx, state); err != nil {
		return err
	}
	return applySpaces(ctx, s)
}}
go syncer.Run(ctx)
```

`loadPlans` and `savePlans` are the app's: one row or one file. `Import` of empty input (nothing
saved yet) is a no-op, and `Import` after an `Apply` answers `forfait.ErrAlreadyApplied` without
changing anything. `Import` also refuses a document over four `directory.MaxSnapshotBytes`, an
unknown version or unknown fields, and filters every value through the same rules as `Apply`.

After Registre's database is restored from a backup, the entitlements revision can go back, and
`Apply` then refuses every snapshot with a logged warning. Call `plans.Reset()` and delete the
saved state; the next pull applies normally.

## MCP endpoints

`porte/mcpauth` is its own module so go-sdk reaches only the apps that serve MCP. Registre mints
the access token with `aud` = the endpoint's resource URI and `scope` = `<app>.read` and
`<app>.write`; the verifier checks it offline and resolves `(iss, sub)` the way the Authorization
header path does.

```go
verify, err := mcpauth.Verifier(ctx, mcpauth.Config{
	Issuer:      cfg.OIDC.Issuer,
	Resource:    "https://courrier.facile.studio/mcp",
	ScopePrefix: "courrier",
	Audience:    cfg.OIDC.ClientID,
	Identities:  store.Identities(),
})
scopes := mcpauth.ScopesFor("courrier")
metadataURL, _ := mcpauth.MetadataURL("https://courrier.facile.studio/mcp")
router.Handle("/.well-known/oauth-protected-resource/mcp", auth.ProtectedResourceMetadataHandler(
	mcpauth.ProtectedResourceMetadata("https://courrier.facile.studio/mcp", cfg.OIDC.Issuer, scopes.List())))
router.Handle("/mcp", mcpauth.Require(verify, &auth.RequireBearerTokenOptions{
	ResourceMetadataURL: metadataURL, Scopes: []string{scopes.Read},
})(mcpauth.PlanGate(plans, cfg.OIDC.Issuer)(mcpHandler)))
```

`Audience` keeps Bulle's exchanged tokens, which still name the app's client id, verifying until
they are resource-bound. A tool that writes checks `mcpauth.HasScope(req.Extra.TokenInfo, scopes.Write)`. `plans` is the
app's `forfait.Store` declaring `mcpauth.AIAccessFeature()`. `mcpauth.Identity(info)` gives back
the `porte.Identity` for handlers written against `porte.From`.

A tool checks that the token's account still exists with
`mcpauth.AccountExists(ctx, req.Extra.TokenInfo, accounts.Identity)`. The confirm page of pending
actions takes its user from `mcpauth.RequireOwnSession(r)`, which passes only a session cookie
minted by a browser login (`IssueCookieFor`) and refuses every bearer, API key, delegated token and
CLI or device session token sent as a cookie. A cookie from before v0.20.1 is refused too, so the
person signs in once more before their first confirmation.

## Why

Six Go apps — Nuage, Courrier, Plume, Agenda, Vision, Sablier — ship roughly 900 lines of
near-identical OIDC client code each, and it has drifted. `porte` is the single version. Five of
those six also carry their own password login, which is why v0.2 exists: what drifts there is not
the flow, which is easy, but the constant-time compare, the equalised timing on an unknown
address and the refusal to say which half of the pair was wrong.

It also fixes three things while extracting: the flow gains PKCE, a nonce, and a constant-time
state comparison. Written once here instead of six times in the apps.

## Stack

| Layer | Tech |
|---|---|
| Runtime | Go 1.25, [go-oidc](https://github.com/coreos/go-oidc), `golang.org/x/oauth2`, `golang.org/x/crypto` for argon2id, [tronc](https://github.com/FacileStudio/tronc) |
| Storage | Sessions through an interface; `pg/` needs only a `*sql.DB` |

## Status

| Version | Scope |
|---|---|
| **v0.1.0** | OIDC only. The flow walked end to end against a conformant in-process issuer, and the security surface reviewed |
| **v0.1.1** | What the first adoption found — see the [changelog](CHANGELOG.md). Journal runs this against a real Authentik. Still deliberately `v0.x`: proven, and not yet promising an unchanged API |
| **v0.2.0** | `porte/session` extracted out of `porte/oidc`, `porte/local` with argon2id passwords, `porte/avatarfs`. **Breaking:** `oidc.Deps.Sessions` is a `*session.Manager` and the app builds it |
| **v0.2.1** | The error sentinels are wrapped rather than stringified, so `errors.Is` works; `ErrWeakPassword` is actually returned |
| **v0.2.2** | `session.Manager` gained `List` and `Revoke`, the two `SessionStore` methods a "your active sessions" screen needs and the manager did not expose. Purely additive |
| **v0.3.0** | Password identities keyed on the account id instead of the mutable email, per OIDC Core §5.7. `local.Kit.ChangePassword` confirms the current password, ends the other logins and rotates the caller's session; `SetPassword` refuses an account that already has one. **Breaking:** `SetPassword` loses its `email` parameter, `SessionStore` gains `DeleteLogins` |
| **v0.4.0** | `POST /auth/oidc/device/exchange`: a CLI trades its provider token for this app's own session, on its own audience |
| **v0.5.0** | `porte/spaces` — membership and role resolution, and the conformance suite an adopter runs against its own table. Purely additive: an app without spaces does not import it |
| **v0.6.0** | `porte/magiclink` — passwordless sign-in by an emailed one-time link, and its store conformance suite. Purely additive: an app without links does not import it |
| **v0.7.0** | Delegated bearer JWTs: `Identity.Scopes` and `Identity.Actor` from the `scope` and `act` claims, `porte.ActorFrom`, and `Config.ScopePrefix` (`OIDC_SCOPE_PREFIX`), which gates a bearer JWT to `<prefix>.read` for reads and `<prefix>.write` for writes. Purely additive: unset, nothing changes |
| **v0.14.0** | `porte/mcpauth/v0.1.0`: MCP resource-server auth. `jwt.Claims` gains `Audience`, `ClientID`, `ID` and `ExpiresAt`, `jwt.Config.ExtraAudiences`, and `oidc.ResolveBearer` exports the bearer mapping. Purely additive |

The first adopter turned out to be Journal — the one app with no OIDC of its own, so the wiring
added a path instead of replacing six hundred lines of one. It kept its own `users` table and
implemented `UserStore` and `PasswordUserStore` over it; the other three stores came from
`porte/pg` unchanged, with their foreign keys repointed. That is the shape the remaining apps
should copy. Sablier followed and is the first adoption of the other kind — a hand-written OIDC
client deleted rather than a gap filled, `SSO_ONLY=true`, and a CLI on the loopback flow. What
each one finds lands in the next `v0.x`: v0.2 is entirely what the first one found, v0.2.1 and
v0.2.2 entirely what the second one did.

## Documentation

| Doc | What's in it |
|---|---|
| [Specification](SPEC.md) | What to build, what was decided and why, the evidence behind both |
| [Configuration](docs/configuration.md) | Every environment variable and `Config` field, and their traps |
| [Development](docs/development.md) | Local setup, the quality gate, versioning |
| [API](docs/api.md) | Every exported symbol — the frozen contract, package by package |

There is still no `docs/architecture.md`, and its two conditions are now both met: a request path
walked in production, and a second adopter to draw it from. Sablier is that second adopter, and
it is deliberately the opposite shape to Journal — SSO-only against a deleted hand-written
client, with a CLI — so a page drawn from the pair documents `porte` rather than one app's
wiring. It is next.

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.

## License

porte is licensed under the [MIT License](LICENSE).
