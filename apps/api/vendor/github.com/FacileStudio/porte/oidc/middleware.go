package oidc

import (
	"context"
	"log/slog"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"
)

// Kit implements session.ClaimsSource: the manager owns the credential, and
// this is the one thing about an identity that only a federated provider can
// answer for.
var _ session.ClaimsSource = (*Kit)(nil)

// Attach fills Roles from the stored identity and refreshes them when
// they have gone stale. It does not fill Email or Name: no store porte reads
// on this path holds either, and going to the app's user table for them would
// double the cost of every authenticated request to serve the handlers that
// do not need them.
//
// It costs a query per request, so it runs only when claims are configured —
// which no app does today. An app that never sets OIDC_CLAIMS_SCOPE pays
// exactly one query per authenticated request, the session lookup, which is
// what it pays now.
func (k *Kit) Attach(ctx context.Context, identity *porte.Identity) {
	if !k.cfg.ClaimsEnabled() {
		return
	}
	stored, ok, err := k.identityFor(ctx, identity.UserID)
	if err != nil || !ok {
		if err != nil {
			k.logger.Warn("porte: failed to load claims", slog.Int64("user_id", identity.UserID), slog.Any("error", err))
		}
		return
	}

	identity.Roles = stored.Roles
	identity.RolesSyncedAt = stored.RolesSyncedAt

	if !stored.RolesStale(k.now(), k.cfg.ClaimsTTL) {
		return
	}
	refreshed, err := k.refreshRoles(ctx, stored)
	if err != nil {
		// Keeping the stale roles is the right failure: an IdP blip
		// must not silently strip everyone's permissions. The stamp is
		// still moved so a dead refresh token is not retried on every
		// single request — through the narrow write, because `stored`
		// is a pre-refresh read and saving it whole would roll back a
		// token another request may have rotated in the meantime.
		k.logger.Warn("porte: role refresh failed, keeping the cached claim",
			slog.Int64("user_id", identity.UserID), slog.Any("error", err))
		if err := k.deps.Identities.MarkRolesSynced(ctx, stored.Provider, stored.Subject, k.now()); err != nil {
			k.logger.Warn("porte: failed to record the refresh attempt", slog.Any("error", err))
		}
		return
	}
	identity.Roles = refreshed.Roles
	identity.RolesSyncedAt = refreshed.RolesSyncedAt
}

// refreshRoles trades the refresh token for a new ID token and reads the roles
// claim off it. This is what offline_access — already requested by every app —
// was for.
func (k *Kit) refreshRoles(ctx context.Context, stored porte.StoredIdentity) (porte.StoredIdentity, error) {
	if stored.Tokens.RefreshToken == "" {
		return stored, errors.Failed("no refresh token")
	}
	source := k.tokenSource(ctx, stored.Tokens)
	token, err := source.Token()
	if err != nil {
		return stored, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return stored, errors.Failed("the refresh response carried no id_token")
	}
	idToken, err := k.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return stored, err
	}
	claims, err := k.claimsFromIDToken(idToken)
	if err != nil {
		return stored, err
	}

	stored.Tokens = toTokenSet(token)
	stored.Roles = claims.Roles
	stored.RolesSyncedAt = k.now()
	if err := k.deps.Identities.Save(ctx, stored); err != nil {
		return stored, err
	}
	return stored, nil
}
