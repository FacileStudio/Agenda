package oidc

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/internal/handoff"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// backchannelLogoutEvent is the event key an IdP puts in a logout token, from
// OpenID Connect Back-Channel Logout 1.0.
const backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"

// Mount registers porte's routes on router, at the paths the contract froze.
// They are relative to wherever router itself is mounted, so an app serving
// its API under /api gets /api/auth/config and the frontends do not move.
//
// RouteConfig is always served; the rest appear only when a provider is
// configured, because an unconfigured provider should mean no endpoint to probe
// rather than an endpoint that 500s. RouteLogout is not here at all any more —
// it belongs to porte/session, which owns the credential whether or not this
// app federates. Mount the manager as well as the kit.
//
// RouteDeviceExchange has a second condition on top of that one: it appears
// only when Config.CLIAudience is set, because without a CLI audience there is
// no verifier for the CLI's tokens and the route could do nothing but refuse.
// Config.MachineAudience does not mount it: that field arms the Authorization
// header path for service-account tokens, which are addressed to a different
// audience entirely. Its absence is a 404, which is exactly what the CLI reads
// as "this app has not shipped the exchange" before falling back to the
// loopback flow.
func (k *Kit) Mount(router chi.Router) {
	router.Get(porte.RouteConfig, k.handleConfig)
	if !k.Enabled() {
		return
	}
	router.Group(func(authenticated chi.Router) {
		authenticated.Use(k.RequireAuth)
		authenticated.Post(porte.RouteSyncProfile, k.handleSyncProfile)
	})
	router.Get(porte.RouteLogin, k.handleLogin)
	router.Get(porte.RouteCallback, k.handleCallback)
	router.Post(porte.RouteExchange, k.handleExchange)
	router.Post(porte.RouteBackchannelLogout, k.handleBackchannelLogout)
	if k.cliTokens != nil {
		router.Post(porte.RouteDeviceExchange, k.handleDeviceExchange)
	}
}

func (k *Kit) handleConfig(w http.ResponseWriter, _ *http.Request) {
	if k.deps.ConfigExtra == nil {
		httpjson.WriteJSON(w, http.StatusOK, porte.ConfigResponse{
			SSOOnly:     k.cfg.SSOOnly,
			OIDCEnabled: k.Enabled(),
		})
		return
	}
	body := map[string]any{}
	for key, value := range k.deps.ConfigExtra() {
		body[key] = value
	}
	body["sso_only"] = k.cfg.SSOOnly
	body["oidc_enabled"] = k.Enabled()
	httpjson.WriteJSON(w, http.StatusOK, body)
}

// fail sends a browser back to the login page with a reason it can render,
// rather than writing a JSON error body into the address bar. Only the handlers
// a browser navigates to use it.
func (k *Kit) fail(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, k.cfg.LoginFailure(reason), http.StatusFound)
}

// failureReason picks the text a user should see for err. Anything porte
// classified as the caller's problem says what it was; everything else is an
// internal failure and says nothing, because its message is for the log.
func failureReason(err error) string {
	var apiErr *errors.Error
	if stderrors.As(err, &apiErr) {
		switch apiErr.Code {
		case "invalid_argument", "already_exists", "unauthenticated", "permission_denied":
			return apiErr.Message
		}
	}
	return "could not sign you in"
}

// handleLogin starts the flow. Beyond what the apps do today it adds PKCE and
// a nonce, both missing from all six: PKCE binds the authorization code to
// this browser, and the nonce binds the ID token to this request.
func (k *Kit) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := porte.NewToken()
	if err != nil {
		k.logger.Error("porte: failed to start the login", slog.Any("error", err))
		k.fail(w, r, "could not start the login")
		return
	}
	nonce, err := porte.NewToken()
	if err != nil {
		k.logger.Error("porte: failed to start the login", slog.Any("error", err))
		k.fail(w, r, "could not start the login")
		return
	}

	pending := flow{State: state, Nonce: nonce, Verifier: oauth2.GenerateVerifier()}
	if r.URL.Query().Get(porte.FlowParam) == porte.FlowCLI {
		pending.CLI = true
		pending.Port = loopbackPort(r.URL.Query().Get(porte.PortParam))
		pending.CLIState = cliState(r.URL.Query().Get(porte.StateParam))
	}

	encoded, err := pending.encode()
	if err != nil {
		k.logger.Error("porte: failed to start the login", slog.Any("error", err))
		k.fail(w, r, "could not start the login")
		return
	}
	k.sessions.SetCookie(w, r, flowCookie, encoded, int(flowTTL.Seconds()))

	http.Redirect(w, r, k.oauth.AuthCodeURL(state,
		oauth2.S256ChallengeOption(pending.Verifier),
		gooidc.Nonce(nonce),
	), http.StatusFound)
}

// maxCLIState bounds the echoed nonce. It is opaque to the server, so the only
// question worth asking is whether it is small enough to be safe to reflect.
const maxCLIState = 128

// cliState accepts an opaque nonce the caller will recognise and nothing else.
// The character set is the one a nonce needs, so a value that tries to be a
// second query parameter or a header never reaches the redirect.
func cliState(value string) string {
	if value == "" || len(value) > maxCLIState {
		return ""
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return ""
		}
	}
	return value
}

// loopbackPort returns the port a CLI is listening on, or "" if the value is
// not a plausible port. Only the number is taken from the request: the host is
// hardcoded to loopback at redirect time, so this parameter cannot be turned
// into an open redirect.
func loopbackPort(value string) string {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1024 || port > 65535 {
		return ""
	}
	return value
}

func (k *Kit) handleCallback(w http.ResponseWriter, r *http.Request) {
	encoded, ok := k.sessions.ReadCookie(r, flowCookie)
	if !ok {
		k.fail(w, r, "the login has expired, start again")
		return
	}
	k.sessions.ClearCookie(w, r, flowCookie)

	pending, ok := decodeFlow(encoded)
	if !ok || !porte.SecureCompare(pending.State, r.URL.Query().Get("state")) {
		k.fail(w, r, "the login could not be verified, start again")
		return
	}

	claims, tokens, err := k.completeFlow(r.Context(), r.URL.Query().Get("code"), pending)
	if err != nil {
		k.logger.Warn("porte: the login flow did not complete", slog.Any("error", err))
		k.fail(w, r, failureReason(err))
		return
	}

	userID, err := k.persist(r.Context(), claims, tokens)
	if err != nil {
		k.logger.Warn("porte: could not record the identity", slog.Any("error", err))
		k.fail(w, r, failureReason(err))
		return
	}

	if pending.CLI {
		k.issueLoginCode(w, r, userID, pending.Port, pending.CLIState)
		return
	}

	if _, _, err := k.sessions.IssueCookie(r.Context(), w, r, userID); err != nil {
		k.logger.Error("porte: could not issue the session", slog.Any("error", err))
		k.fail(w, r, "could not sign you in")
		return
	}
	http.Redirect(w, r, k.cfg.SuccessURL, http.StatusFound)
}

// completeFlow exchanges the authorization code and verifies what comes back.
func (k *Kit) completeFlow(ctx context.Context, code string, pending flow) (porte.Claims, porte.TokenSet, error) {
	if code == "" {
		return porte.Claims{}, porte.TokenSet{}, errors.Invalid("missing authorization code")
	}
	token, err := k.oauth.Exchange(ctx, code, oauth2.VerifierOption(pending.Verifier))
	if err != nil {
		return porte.Claims{}, porte.TokenSet{}, errors.Internal("failed to exchange the authorization code", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return porte.Claims{}, porte.TokenSet{}, errors.Internal("the provider returned no id_token", nil)
	}
	idToken, err := k.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return porte.Claims{}, porte.TokenSet{}, errors.Unauthorized("invalid id_token")
	}
	if !porte.SecureCompare(idToken.Nonce, pending.Nonce) {
		return porte.Claims{}, porte.TokenSet{}, errors.Unauthorized("id_token nonce mismatch")
	}

	claims, err := k.claimsFromIDToken(idToken)
	if err != nil {
		return porte.Claims{}, porte.TokenSet{}, errors.Internal(err.Error(), err)
	}
	if claims.Email == "" {
		return porte.Claims{}, porte.TokenSet{}, errors.Invalid("the identity provider returned no email")
	}
	return claims, toTokenSet(token), nil
}

// persist runs the avatar fetch, the app's upsert and the identity row, in
// that order. The avatar first, so its URL rides into the upsert and the whole
// callback stays one write on the app's side.
func (k *Kit) persist(ctx context.Context, claims porte.Claims, tokens porte.TokenSet) (int64, error) {
	claims.AvatarURL = k.syncAvatar(ctx, claims)
	claims.Tokens = tokens

	userID, err := k.deps.Users.UpsertFromOIDC(ctx, claims)
	if err != nil {
		return 0, err
	}

	stored := porte.StoredIdentity{
		UserID:   userID,
		Provider: claims.Provider,
		Subject:  claims.Subject,
		Tokens:   tokens,
		Roles:    claims.Roles,
		SyncedAt: k.now(),
	}
	if k.cfg.ClaimsEnabled() {
		stored.RolesSyncedAt = k.now()
	}
	if err := k.deps.Identities.Save(ctx, stored); err != nil {
		return 0, errors.Internal("failed to record the identity", err)
	}
	return userID, nil
}

// syncAvatar is best effort by design: a provider that serves a broken picture
// URL must not be able to block every login with it.
func (k *Kit) syncAvatar(ctx context.Context, claims porte.Claims) string {
	if k.deps.Avatars == nil || claims.Picture == "" {
		return ""
	}
	data, contentType, err := FetchAvatar(ctx, claims.Picture)
	if err != nil {
		k.logger.Warn("porte: avatar fetch skipped", slog.String("subject", claims.Subject), slog.Any("error", err))
		return ""
	}
	avatarURL, err := k.deps.Avatars.Put(ctx, claims.AvatarKey(), data, contentType)
	if err != nil {
		k.logger.Warn("porte: avatar store failed", slog.String("subject", claims.Subject), slog.Any("error", err))
		return ""
	}
	return avatarURL
}

// issueLoginCode ends the CLI flow. The code is a bearer credential for sixty
// seconds, so it is stored hashed exactly like a session token.
//
// A CLI that started a loopback listener gets the code by redirect and never
// sees a page. The rest get one, and it is one of the two places porte draws
// HTML rather than redirecting to the app: there is no app page that can show
// a code porte just minted. It carries the app's name and logo for the same
// reason Config.LoginFailure lands on the app's own login page, which is that
// a human reading it should be able to tell whose login this was.
//
// The response answers no-store because it carries the credential itself,
// which is what OAuth 2.1 §7.1 asks of any response that does.
func (k *Kit) issueLoginCode(w http.ResponseWriter, r *http.Request, userID int64, port, cliState string) {
	code, err := porte.NewToken()
	if err != nil {
		k.logger.Error("porte: failed to issue a login code", slog.Any("error", err))
		k.fail(w, r, "could not complete the login")
		return
	}
	if err := k.deps.Codes.Create(r.Context(), porte.LoginCode{
		CodeHash:  porte.HashToken(code),
		UserID:    userID,
		ExpiresAt: k.now().Add(k.cfg.LoginCodeTTL),
	}); err != nil {
		k.logger.Error("porte: failed to store the login code", slog.Any("error", err))
		k.fail(w, r, "could not complete the login")
		return
	}

	if port != "" {
		// The host is ours, only the port came from the request, so
		// this cannot be pointed anywhere but at the local machine.
		query := url.Values{"code": {code}}
		if cliState != "" {
			query.Set("state", cliState)
		}
		target := url.URL{
			Scheme:   "http",
			Host:     net.JoinHostPort("127.0.0.1", port),
			Path:     "/",
			RawQuery: query.Encode(),
		}
		http.Redirect(w, r, target.String(), http.StatusFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	handoff.Write(w, http.StatusOK, handoff.Page{
		AppName: k.cfg.AppLabel(),
		LogoURL: k.cfg.Logo(),
		Heading: "Signed in",
		Body:    "Paste this code into your terminal.",
		Hint: fmt.Sprintf("The code is valid for %d seconds and works once.",
			int(k.cfg.LoginCodeTTL.Seconds())),
		Code: code,
	})
}

// handleExchange is the CLI's half: one-time code in, bearer token out. This
// is porte's token endpoint in everything but name, so it answers under the
// no-store OAuth 2.1 §7.1 requires of any response carrying a credential.
func (k *Kit) handleExchange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	var request porte.ExchangeRequest
	if err := httpjson.DecodeJSON(w, r, &request); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if request.Code == "" {
		httpjson.WriteError(w, errors.Invalid("missing code"))
		return
	}

	code, err := k.deps.Codes.Consume(r.Context(), porte.HashToken(request.Code))
	switch {
	case err == nil:
	case isErr(err, porte.ErrCodeConsumed):
		// A replay, not a typo. Worth a log line: the code was valid
		// once, so either the CLI retried or someone else has it.
		k.logger.Warn("porte: login code replayed")
		httpjson.WriteError(w, errors.Unauthorized("invalid or expired code"))
		return
	case isErr(err, porte.ErrNotFound):
		httpjson.WriteError(w, errors.Unauthorized("invalid or expired code"))
		return
	default:
		httpjson.WriteError(w, errors.Internal("failed to read the login code", err))
		return
	}
	if code.Expired(k.now()) {
		httpjson.WriteError(w, errors.Unauthorized("invalid or expired code"))
		return
	}

	token, _, err := k.sessions.Issue(r.Context(), code.UserID, "")
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, porte.ExchangeResponse{
		UserID: strconv.FormatInt(code.UserID, 10),
		Token:  token,
	})
}

// handleDeviceExchange is the suite's half: an access token the provider's
// device grant already issued goes in, this app's own session token comes out.
// It is what makes one `facile login` serve every CLI. The alternative, a CLI
// storing the provider's token in the slot where it keeps its own session, is
// a login that stops working when that token expires an hour later.
//
// It composes three pieces that already existed and adds no verification of
// its own: the same bearerVerifier the Authorization header path uses, the
// same (issuer, sub) lookup the login callback matches on, and the same
// Manager.Issue that mints every other session. What is new is only that the
// token arrives in a request body instead of a header.
//
// # The audience this trusts, and why it is its own setting
//
// The token is verified against Config.CLIAudience (OIDC_CLI_AUDIENCE), which
// an app running the suite CLI sets to `facile-cli`. Registre issues the CLI's
// token to that client and the client declares no audiences of its own, so the
// token carries `aud: ["facile-cli"]`.
//
// That is deliberately not Config.MachineAudience, and the two cannot hold the
// same value. MachineAudience is this app's own client id, because that is
// what a service account's token is addressed to: Registre's suite-ci account
// declares `audiences: [courrier]`, so courrier checks `courrier` there. The
// CLI's token is addressed to the CLI and presented at every tool. One setting
// would force an app to pick one population, and an app that picked the CLI
// would start rejecting every service-account token it had been accepting,
// silently, with nothing in its own configuration having changed meaning.
//
// The split is also a boundary, not only a naming fix. Setting CLIAudience
// verifies tokens for this route and nothing else: it never reaches
// session.Manager.WithJWT, so a facile-cli token is not a credential on every
// RequireAuth route. It buys a session here only by being exchanged for one,
// and that leaves a session row an app can list, expire and revoke, where a
// verified bearer JWT leaves nothing at all.
//
// What the split does not close: a stolen or phished facile-cli token can
// still be exchanged for a durable session at every app that trusts it, as
// whoever the token names. facile-cli is a public client, so anyone can start
// its device grant; what stands between that and a session here is the human
// approving the code at the provider, and the requirement below that the
// subject already hold a local account. Containing the rest is back-channel
// logout's job, and no app in the suite registers a backchannel_logout_uri
// today, so that containment is not in place yet. Do not read this endpoint as
// providing it.
//
// # Refusals
//
// No account is ever created here. A verified subject with no row in
// porte_identities is refused, because account creation belongs to the login
// callback, which is the path that holds a verified email. Provisioning from a
// bearer would let anyone the provider will mint a token for materialise an
// account in every app at once. Every refusal answers the same 401 with the
// same message, so the response says nothing about which check failed or
// whether the subject has an account here, and the identity store is only
// consulted after the cryptographic and claim checks have passed, so the
// latency says nothing either.
//
// No CSRF header is required, and that is not an omission. The suite's default
// is the opposite because the default transport is a cookie a browser attaches
// on its own; here there is no ambient credential to abuse, because the caller
// must put a token it holds into the request body, which a cross-site form
// cannot do. Requiring the header would only break the CLI, which is not a browser.
//
// The response carries a credential, so it is no-store per OAuth 2.1 §7.1, and
// the request carries one, so neither the token nor any error quoting it is
// ever logged.
func (k *Kit) handleDeviceExchange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	var request porte.DeviceExchangeRequest
	if err := httpjson.DecodeJSON(w, r, &request); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if request.AccessToken == "" {
		httpjson.WriteError(w, errors.Invalid("missing access_token"))
		return
	}

	identity, err := k.cliTokens.VerifyJWT(r.Context(), request.AccessToken)
	if err != nil {
		k.logger.Debug("porte: device exchange refused", slog.Any("error", err))
		httpjson.WriteError(w, errors.Unauthorized("invalid access token"))
		return
	}

	token, _, err := k.sessions.Issue(r.Context(), identity.UserID, "")
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, porte.ExchangeResponse{
		UserID: strconv.FormatInt(identity.UserID, 10),
		Token:  token,
	})
}

// handleBackchannelLogout is the only mechanism by which a deactivation in the
// IdP reaches an app that issued its own opaque, long-lived session. Claim
// TTLs cover a changed role; nothing but this covers a disabled account.
func (k *Kit) handleBackchannelLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	if err := r.ParseForm(); err != nil {
		httpjson.WriteError(w, errors.Invalid("malformed logout request"))
		return
	}
	rawToken := r.PostForm.Get("logout_token")
	if rawToken == "" {
		httpjson.WriteError(w, errors.Invalid("missing logout_token"))
		return
	}

	subject, err := k.verifyLogoutToken(r.Context(), rawToken)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	stored, err := k.deps.Identities.Find(r.Context(), k.cfg.Issuer, subject)
	if isErr(err, porte.ErrNotFound) {
		// This user never signed in here. Nothing to revoke, and
		// saying so would confirm who does have an account.
		httpjson.WriteJSON(w, http.StatusOK, porte.LogoutResponse{LoggedOut: true})
		return
	}
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to resolve the identity", err))
		return
	}

	deleted, err := k.sessions.RevokeUser(r.Context(), stored.UserID)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to revoke the sessions", err))
		return
	}
	k.logger.Info("porte: back-channel logout",
		slog.Int64("user_id", stored.UserID), slog.Int64("sessions_revoked", deleted))
	httpjson.WriteJSON(w, http.StatusOK, porte.LogoutResponse{LoggedOut: true})
}

// verifyLogoutToken applies OpenID Connect Back-Channel Logout 1.0 §2.6. The
// nonce check is the one that matters: a logout token must not carry one, and
// accepting an ID token here would let anyone holding their own ID token log
// out anybody.
func (k *Kit) verifyLogoutToken(ctx context.Context, rawToken string) (string, error) {
	idToken, err := k.verifier.Verify(ctx, rawToken)
	if err != nil {
		return "", errors.Unauthorized("invalid logout token")
	}
	if idToken.Nonce != "" {
		return "", errors.Unauthorized("a logout token must not carry a nonce")
	}
	var payload struct {
		Events map[string]any `json:"events"`
		JTI    string         `json:"jti"`
	}
	if err := idToken.Claims(&payload); err != nil {
		return "", errors.Unauthorized("unreadable logout token")
	}
	if _, ok := payload.Events[backchannelLogoutEvent]; !ok {
		return "", errors.Unauthorized("logout token is missing the back-channel logout event")
	}
	if idToken.Subject == "" {
		// A sid-only token would need a session index porte does not
		// keep. Revoking every session of a subject is the coarser and
		// safer answer, so the subject is required.
		return "", errors.Unauthorized("logout token carries no subject")
	}
	return idToken.Subject, nil
}

// handleSyncProfile refreshes name and avatar from the IdP, rate-limited by
// the same profile_synced_at the apps already keep.
func (k *Kit) handleSyncProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := porte.From(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return
	}
	stored, ok, err := k.identityFor(r.Context(), identity.UserID)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to load the identity", err))
		return
	}
	if !ok || stored.Tokens.AccessToken == "" {
		httpjson.WriteJSON(w, http.StatusOK, porte.SyncProfileResponse{Synced: false})
		return
	}
	if !stored.SyncedAt.IsZero() && k.now().Sub(stored.SyncedAt) < porte.DefaultProfileSyncInterval {
		httpjson.WriteJSON(w, http.StatusOK, porte.SyncProfileResponse{Synced: false})
		return
	}

	source := k.tokenSource(r.Context(), stored.Tokens)
	info, err := k.provider.UserInfo(r.Context(), source)
	if err != nil {
		// The refresh token is dead. Clearing it stops this call
		// retrying a doomed round trip every five minutes forever.
		k.logger.Warn("porte: profile sync failed, clearing tokens",
			slog.Int64("user_id", identity.UserID), slog.Any("error", err))
		stored.Tokens = porte.TokenSet{}
		stored.SyncedAt = k.now()
		_ = k.deps.Identities.Save(r.Context(), stored)
		httpjson.WriteJSON(w, http.StatusOK, porte.SyncProfileResponse{Synced: false})
		return
	}

	// OpenID Connect Core §5.3.2: the sub in a UserInfo response MUST be
	// compared to the sub of the ID token, and the response rejected when
	// they differ. go-oidc does not do it — it fetches and parses, nothing
	// more — so without this line a provider or anything sitting between it
	// and here can rewrite one user's profile with another user's claims.
	if !porte.SecureCompare(info.Subject, stored.Subject) {
		k.logger.Warn("porte: userinfo returned a different subject, refusing the sync",
			slog.Int64("user_id", identity.UserID))
		httpjson.WriteError(w, errors.Unauthorized("the userinfo response is for a different subject"))
		return
	}

	var raw struct {
		Email             string `json:"email"`
		EmailVerified     any    `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Picture           string `json:"picture"`
	}
	if err := info.Claims(&raw); err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse the userinfo claims", err))
		return
	}

	claims := porte.Claims{
		Provider:          stored.Provider,
		Subject:           stored.Subject,
		Email:             raw.Email,
		EmailVerified:     k.emailTrusted(raw.EmailVerified),
		Name:              raw.Name,
		PreferredUsername: raw.PreferredUsername,
		GivenName:         raw.GivenName,
		FamilyName:        raw.FamilyName,
		Picture:           raw.Picture,
		Roles:             stored.Roles,
	}
	if claims.Email == "" {
		claims.Email = identity.Email
	}

	if refreshed, err := source.Token(); err == nil {
		stored.Tokens = toTokenSet(refreshed)
	}
	claims.AvatarURL = k.syncAvatar(r.Context(), claims)
	claims.Tokens = stored.Tokens

	// The same upsert the callback uses. Reusing it is why the app's
	// product behaviour on first login — a display colour, first-user
	// admin — cannot drift away from what a sync does.
	if _, err := k.deps.Users.UpsertFromOIDC(r.Context(), claims); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	stored.SyncedAt = k.now()
	if err := k.deps.Identities.Save(r.Context(), stored); err != nil {
		httpjson.WriteError(w, errors.Internal("failed to record the identity", err))
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, porte.SyncProfileResponse{Synced: true})
}

// identityFor returns this app's OIDC identity for a user. A human may hold
// several rows from v0.2 on — a password and a subject — so the provider is
// part of the lookup rather than an assumption about there being only one.
func (k *Kit) identityFor(ctx context.Context, userID int64) (porte.StoredIdentity, bool, error) {
	identities, err := k.deps.Identities.ListByUser(ctx, userID)
	if err != nil {
		if isErr(err, porte.ErrNotFound) {
			return porte.StoredIdentity{}, false, nil
		}
		return porte.StoredIdentity{}, false, err
	}
	for _, candidate := range identities {
		if candidate.Provider == k.cfg.Issuer {
			return candidate, true, nil
		}
	}
	return porte.StoredIdentity{}, false, nil
}

func isErr(err, target error) bool { return stderrors.Is(err, target) }
