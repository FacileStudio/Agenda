package users

import (
	"net/http"
	"strings"
	"testing"
)

// A federated-only account has no password to confirm, so `password` alone is
// the whole request. TestSettingAFirstPasswordLetsTheAccountLogIn checks the
// credential lands where porte reads it: keyed on the account id, not on the
// address, which is the half of the upgrade a green build says nothing about.
func TestSettingAFirstPasswordLetsTheAccountLogIn(t *testing.T) {
	h := newHarness(t)
	userID := h.account("first@facile.studio")
	token := h.login(userID)

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{"password": "correct horse"})
	if response.Code != http.StatusOK {
		t.Fatalf("set a first password: got %d %s", response.Code, response.Body.String())
	}

	login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "first@facile.studio",
		"password": "correct horse",
	})
	if login.Code != http.StatusOK {
		t.Fatalf("log in with the new password: got %d %s", login.Code, login.Body.String())
	}
}

// This is the security fix. Agenda shipped `PATCH /users/me {"password": …}`
// straight into SetPassword, so anyone holding a session — a borrowed laptop,
// a stolen cookie — could replace the password without knowing it, turning a
// temporary hold on an account into a permanent one.
// TestReplacingAPasswordWithoutTheCurrentOneIsRefused checks the request is
// refused and, more importantly, that it changed nothing: the old password
// still signs in and the offered one does not.
func TestReplacingAPasswordWithoutTheCurrentOneIsRefused(t *testing.T) {
	h := newHarness(t)
	userID := h.account("hold@facile.studio")
	token := h.login(userID)

	if response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"password": "original secret",
	}); response.Code != http.StatusOK {
		t.Fatalf("seed the first password: got %d %s", response.Code, response.Body.String())
	}

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{"password": "stolen secret"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("replacing a password without the current one: got %d %s, want 400",
			response.Code, response.Body.String())
	}
	if code := errorCode(t, response); code != "invalid_argument" {
		t.Fatalf("error code: got %q want invalid_argument", code)
	}

	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "hold@facile.studio",
		"password": "stolen secret",
	}); login.Code == http.StatusOK {
		t.Fatal("the refused password signs in, so the account was taken over anyway")
	}
	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "hold@facile.studio",
		"password": "original secret",
	}); login.Code != http.StatusOK {
		t.Fatalf("the original password stopped working: got %d %s", login.Code, login.Body.String())
	}
}

// TestChangingAPasswordWithTheCurrentOneSucceeds is the path the refusal above
// exists to funnel people into.
func TestChangingAPasswordWithTheCurrentOneSucceeds(t *testing.T) {
	h := newHarness(t)
	userID := h.account("change@facile.studio")
	token := h.login(userID)

	if response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"password": "original secret",
	}); response.Code != http.StatusOK {
		t.Fatalf("seed the first password: got %d %s", response.Code, response.Body.String())
	}

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"current_password": "original secret",
		"password":         "replacement secret",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("change the password: got %d %s", response.Code, response.Body.String())
	}

	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "change@facile.studio",
		"password": "replacement secret",
	}); login.Code != http.StatusOK {
		t.Fatalf("the new password does not sign in: got %d %s", login.Code, login.Body.String())
	}
	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "change@facile.studio",
		"password": "original secret",
	}); login.Code == http.StatusOK {
		t.Fatal("the replaced password still signs in")
	}
}

// TestChangingAPasswordWithTheWrongCurrentOneIs401 checks the confirmation is
// a real check and not a field that merely has to be present.
func TestChangingAPasswordWithTheWrongCurrentOneIs401(t *testing.T) {
	h := newHarness(t)
	userID := h.account("wrong@facile.studio")
	token := h.login(userID)

	if response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"password": "original secret",
	}); response.Code != http.StatusOK {
		t.Fatalf("seed the first password: got %d %s", response.Code, response.Body.String())
	}

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"current_password": "not the password",
		"password":         "replacement secret",
	})
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current password: got %d %s, want 401", response.Code, response.Body.String())
	}
	if code := errorCode(t, response); code != "unauthenticated" {
		t.Fatalf("error code: got %q want unauthenticated", code)
	}
}

// A password's replacement should not leave credentials minted by the old one
// alive, and the screen that made the change should keep working.
// TestChangingAPasswordEndsTheOtherLoginsOnly checks porte rotated the caller's
// own session rather than dropping it, ended the other browser's, and spared
// the labelled API token a CalDAV client is holding.
func TestChangingAPasswordEndsTheOtherLoginsOnly(t *testing.T) {
	h := newHarness(t)
	userID := h.account("sessions@facile.studio")
	caller := h.login(userID)
	otherBrowser := h.login(userID)
	apiToken := h.apiToken(userID)

	if response := h.do(http.MethodPatch, "/users/me", caller, map[string]string{
		"password": "original secret",
	}); response.Code != http.StatusOK {
		t.Fatalf("seed the first password: got %d %s", response.Code, response.Body.String())
	}

	response := h.do(http.MethodPatch, "/users/me", caller, map[string]string{
		"current_password": "original secret",
		"password":         "replacement secret",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("change the password: got %d %s", response.Code, response.Body.String())
	}

	rotated := rotatedToken(response)
	if rotated == "" {
		t.Fatal("no rotated session cookie, so the browser that changed the password was signed out")
	}
	if rotated == caller {
		t.Fatal("the session token did not rotate, so the credential survives its own password change")
	}
	if !h.authenticates(rotated) {
		t.Fatal("the rotated session does not authenticate")
	}
	if h.authenticates(caller) {
		t.Fatal("the pre-change session still authenticates")
	}
	if h.authenticates(otherBrowser) {
		t.Fatal("the other browser's login survived the password change")
	}
	if !h.authenticates(apiToken) {
		t.Fatal("the labelled API token was revoked, which breaks every CalDAV client on the account")
	}
}

// The response body carries no token. Agenda's client is cookie-only —
// backend.ts sends credentials: 'include' and never reads localStorage or sets
// an Authorization header — so porte's rotated cookie is the whole story and a
// token field would be a shape change nobody consumes.
// TestTheProfileResponseCarriesNoToken pins that, because the sibling apps that
// do hold a bearer need one and copying them here is the easy mistake.
func TestTheProfileResponseCarriesNoToken(t *testing.T) {
	h := newHarness(t)
	userID := h.account("shape@facile.studio")
	token := h.login(userID)

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{"password": "original secret"})
	if response.Code != http.StatusOK {
		t.Fatalf("set a first password: got %d %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); strings.Contains(body, `"token"`) {
		t.Fatalf("the profile response grew a token field: %s", body)
	}
}
