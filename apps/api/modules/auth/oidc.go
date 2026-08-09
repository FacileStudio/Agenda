package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	stderrors "errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/internal/oidcavatar"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

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
		Secure:   isSecure(r),
	})
	http.Redirect(w, r, h.oauth2Cfg.AuthCodeURL(state), http.StatusFound)
}

func (h *oidcHandler) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		h.fail(w, r, "login session expired, please try again")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure(r),
	})

	oauth2Token, err := h.oauth2Cfg.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		h.fail(w, r, "identity provider rejected the login")
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.fail(w, r, "identity provider returned no id_token")
		return
	}

	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.fail(w, r, "identity provider returned an invalid id_token")
		return
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     any    `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Picture           string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		h.fail(w, r, "could not read the identity provider response")
		return
	}
	if claims.Email == "" {
		h.fail(w, r, "identity provider did not return an email")
		return
	}

	// Authentik (and other providers) often omit picture from the ID token but
	// include it in the UserInfo endpoint. Supplement here so the avatar is
	// fetched immediately on first login instead of waiting for the sync cooldown.
	if claims.Picture == "" {
		tokenSrc := h.oauth2Cfg.TokenSource(r.Context(), oauth2Token)
		if userInfo, uiErr := h.provider.UserInfo(r.Context(), tokenSrc); uiErr == nil {
			var uiClaims struct {
				Picture string `json:"picture"`
			}
			if uiErr = userInfo.Claims(&uiClaims); uiErr == nil {
				claims.Picture = uiClaims.Picture
			}
		}
	}

	profile := oidcavatar.Profile{
		Name:              claims.Name,
		PreferredUsername: claims.PreferredUsername,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		Picture:           claims.Picture,
	}
	_, token, err := h.service.upsertOIDCUser(r.Context(), idToken.Subject, claims.Email, emailClaimTrusted(claims.EmailVerified), profile, oauth2Token)
	if err != nil {
		h.fail(w, r, signInFailure(err))
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

func signInFailure(err error) string {
	var apiErr *errors.Error
	if stderrors.As(err, &apiErr) && apiErr.Code == "invalid_argument" {
		return apiErr.Message
	}
	return "could not sign you in"
}

func (h *oidcHandler) fail(w http.ResponseWriter, r *http.Request, reason string) {
	dest, err := url.Parse(h.successURL)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid(reason))
		return
	}
	dest.Path = strings.TrimSuffix(dest.Path, "/") + "/login"
	q := dest.Query()
	q.Set("error", reason)
	dest.RawQuery = q.Encode()
	http.Redirect(w, r, dest.String(), http.StatusFound)
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// emailClaimTrusted reports whether the email in an ID token may be used to
// match an existing account. Matching on a mutable, unproven email is how an
// identity provider that lets a user set any address becomes an account
// takeover primitive, so the fallback is refused when the provider explicitly
// says the address is unverified. An absent claim is treated as trusted: the
// provider is making no assertion either way, and refusing it would strand
// every account created before oidc_subject was recorded.
func emailClaimTrusted(v any) bool {
	switch val := v.(type) {
	case nil:
		return true
	case bool:
		return val
	case string:
		return !strings.EqualFold(val, "false")
	default:
		return false
	}
}
