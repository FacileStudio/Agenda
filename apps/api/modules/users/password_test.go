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

// A combined body ran the password change first and the profile columns
// second, with no transaction between them. When the profile half failed on a
// taken address the caller got a 409, but the password was already stored, the
// account's other logins were already gone and the cookie was already rotated
// — a request reported as failed that had changed the credential.
//
// porte's session manager writes outside any transaction the app can hold, so
// there is nothing to roll back and reordering only moves which half is
// orphaned. The request is refused instead.
// TestACombinedPasswordAndProfileUpdateIsRefused is the regression.
func TestACombinedPasswordAndProfileUpdateIsRefused(t *testing.T) {
	h := newHarness(t)
	h.account("taken@facile.studio")
	userID := h.account("combined@facile.studio")
	token := h.login(userID)

	if response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"password": "original secret",
	}); response.Code != http.StatusOK {
		t.Fatalf("seed the first password: got %d %s", response.Code, response.Body.String())
	}

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"current_password": "original secret",
		"password":         "replacement secret",
		"email":            "taken@facile.studio",
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("a combined update: got %d %s, want 400", response.Code, response.Body.String())
	}
	if code := errorCode(t, response); code != "invalid_argument" {
		t.Fatalf("error code: got %q want invalid_argument", code)
	}

	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "combined@facile.studio",
		"password": "replacement secret",
	}); login.Code == http.StatusOK {
		t.Fatal("the refused request changed the password anyway")
	}
	if login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "combined@facile.studio",
		"password": "original secret",
	}); login.Code != http.StatusOK {
		t.Fatalf("the original password stopped working: got %d %s", login.Code, login.Body.String())
	}
}

// The refusal is about the combination, not about either half, so a body
// carrying only profile columns still applies them.
func TestAProfileOnlyUpdateStillApplies(t *testing.T) {
	h := newHarness(t)
	userID := h.account("profile@facile.studio")
	token := h.login(userID)

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{"name": "Renamed"})
	if response.Code != http.StatusOK {
		t.Fatalf("profile-only update: got %d %s", response.Code, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, "Renamed") {
		t.Fatalf("the name was not applied: %s", body)
	}
}

// porte's sentinels all read "porte: …" and tronc copies an error's message
// straight into the response envelope, so an unmapped one names a library the
// API mentions nowhere else. ErrPasswordSet was already remapped by hand;
// ErrNoPassword was not, and a change offered against an account with no
// password answered "porte: this account has no password".
// TestChangingAPasswordOnAnAccountWithNoneSaysNothingAboutPorte is the
// regression, and it holds for every sentinel because the mapping is one place.
func TestChangingAPasswordOnAnAccountWithNoneSaysNothingAboutPorte(t *testing.T) {
	h := newHarness(t)
	userID := h.account("federated@facile.studio")
	token := h.login(userID)

	response := h.do(http.MethodPatch, "/users/me", token, map[string]string{
		"current_password": "there is no password",
		"password":         "replacement secret",
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("changing a password that does not exist: got %d %s, want 400",
			response.Code, response.Body.String())
	}
	if body := response.Body.String(); strings.Contains(body, "porte") {
		t.Fatalf("porte's own error text reached the client: %s", body)
	}
}

// A wrong password is the sentinel most callers will ever see, so it is worth
// pinning beside the one that was reported.
func TestAWrongPasswordSaysNothingAboutPorte(t *testing.T) {
	h := newHarness(t)
	h.account("leak@facile.studio")

	login := h.do(http.MethodPost, "/auth/login", "", map[string]string{
		"email":    "leak@facile.studio",
		"password": "not the password",
	})
	if login.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: got %d %s, want 401", login.Code, login.Body.String())
	}
	if body := login.Body.String(); strings.Contains(body, "porte") {
		t.Fatalf("porte's own error text reached the client: %s", body)
	}
}
