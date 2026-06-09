package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"api/internal/authcontext"
	"api/internal/env"
	"api/internal/errors"
	"api/internal/httpjson"
	"api/internal/oidcavatar"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie = "oidc_state"
	oidcStateTTL    = 10 * time.Minute
)

type oidcHandler struct {
	provider   *gooidc.Provider
	verifier   *gooidc.IDTokenVerifier
	oauth2Cfg  oauth2.Config
	service    *Service
	successURL string
}

func newOIDCHandler(ctx context.Context, cfg *env.OIDCConfig, service *Service) (*oidcHandler, error) {
	provider, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}
	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "email", "profile", "offline_access"},
	}
	verifier := provider.Verifier(&gooidc.Config{ClientID: cfg.ClientID})
	return &oidcHandler{
		provider:   provider,
		verifier:   verifier,
		oauth2Cfg:  oauth2Cfg,
		service:    service,
		successURL: cfg.SuccessURL,
	}, nil
}

func (h *oidcHandler) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to generate state", err))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   int(oidcStateTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.oauth2Cfg.AuthCodeURL(state), http.StatusFound)
}

func (h *oidcHandler) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		httpjson.WriteError(w, errors.Invalid("invalid oauth2 state"))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	oauth2Token, err := h.oauth2Cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to exchange code", err))
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		httpjson.WriteError(w, errors.Internal("missing id_token in response", nil))
		return
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		httpjson.WriteError(w, errors.Unauthorized("invalid id_token"))
		return
	}

	var claims struct {
		Email            string `json:"email"`
		EmailVerified    bool   `json:"email_verified"`
		Name             string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName        string `json:"given_name"`
		FamilyName       string `json:"family_name"`
		Picture          string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse claims", err))
		return
	}
	if claims.Email == "" {
		httpjson.WriteError(w, errors.Invalid("OIDC provider did not return an email"))
		return
	}

	profile := oidcavatar.Profile{
		Name:             claims.Name,
		PreferredUsername: claims.PreferredUsername,
		GivenName:        claims.GivenName,
		FamilyName:       claims.FamilyName,
		Picture:          claims.Picture,
	}
	_, token, err := h.service.upsertOIDCUser(r.Context(), claims.Email, profile, oauth2Token)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	setSessionCookie(w, r, token)
	dest := h.successURL
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (h *oidcHandler) syncProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return
	}

	synced, err := h.service.SyncOIDCProfile(r.Context(), identity.UserID, h.provider, h.oauth2Cfg)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"synced": synced})
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
