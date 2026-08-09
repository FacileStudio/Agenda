package session

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/porte"
)

// isSecure is Courrier's test, which is the correct one behind Traefik: the
// TLS terminates at the proxy, so r.TLS is nil on every production request and
// deriving the flag from the success URL — as three apps do — marks the cookie
// insecure exactly where it matters.
func isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// secure decides the Secure attribute. The per-request test above is kept
// because it is the right one behind a proxy, but the configuration overrides
// it upward and never downward: an app whose redirect URL is https is served
// over https, and a proxy that was misconfigured into dropping
// X-Forwarded-Proto must not be able to talk porte into shipping the session
// cookie in the clear. The BFF half of the browser-apps BCP is a MUST here.
func (m *Manager) secure(r *http.Request) bool {
	return m.cfg.HTTPS() || isSecure(r)
}

// hostPrefix is the one cookie attribute an attacker on a sibling host cannot
// forge. A browser accepts a __Host- cookie only from a request that is Secure,
// with Path=/ and no Domain — so it is necessarily host-locked, and the server
// can tell it apart from a look-alike.
//
// Without it every app in a suite sharing one parent domain is one XSS, one
// rogue app or one subdomain takeover away from session fixation on all the
// others: a plain cookie named `session` scoped to Domain=example.com is
// indistinguishable at the server from the app's own host-only one, and a
// planted value that the browser happens to send first silently wins. porte
// puts eight bytes in front of the name instead.
//
// The prefix is dropped over plain http, because a browser rejects it there
// outright and local development would stop working.
const hostPrefix = "__Host-"

// cookieName is the name porte writes a cookie under. porte.SessionCookieName
// remains the base — the constant is what Courrier and Agenda already ship, and
// the reader below still accepts it, so adopting porte does not log their users
// out. New cookies are always written prefixed.
func (m *Manager) cookieName(r *http.Request, base string) string {
	if m.secure(r) {
		return hostPrefix + base
	}
	return base
}

// ReadCookie returns a cookie by base name.
//
// Over https it reads the prefixed name and, unless the app opted in to the
// migration, only the prefixed name. An unconditional fallback would make the
// prefix decoration: the attack it exists to stop is a sibling planting a
// look-alike, and a reader that accepts the unprefixed name accepts exactly
// that cookie from exactly that attacker. It is worse against a victim who is
// *not* signed in, who has no prefixed cookie for the preference order to
// prefer.
//
// Over plain http the bare name is the only name a browser will keep, so it is
// what local development reads.
func (m *Manager) ReadCookie(r *http.Request, base string) (string, bool) {
	names := []string{base}
	if m.secure(r) {
		names = []string{hostPrefix + base}
		if m.cfg.AcceptLegacyCookie {
			names = append(names, base)
		}
	}
	for _, name := range names {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value, true
		}
	}
	return "", false
}

func (m *Manager) SetCookie(w http.ResponseWriter, r *http.Request, base, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName(r, base),
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   m.secure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie expires both spellings. Clearing only the one porte would write
// today would leave a legacy cookie behind on the logout that is meant to
// migrate the user off it.
func (m *Manager) ClearCookie(w http.ResponseWriter, r *http.Request, base string) {
	m.SetCookie(w, r, base, "", -1)
	if m.secure(r) {
		http.SetCookie(w, &http.Cookie{
			Name: base, Value: "", Path: "/", MaxAge: -1,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
	}
}

// SetSessionCookie is the whole reason the frontends stop touching tokens:
// HttpOnly puts the credential out of reach of an XSS, and SameSite=Lax with
// the header check on mutating routes covers the CSRF that a cookie invites.
//
// Lax rather than Strict is deliberate and is the one browser-apps BCP
// deviation porte makes: these apps link to each other and send links by mail,
// and Strict logs a user out of every inbound link. The custom-header check the
// BCP offers as the alternative CSRF defence is enforced on every mutating
// cookie request, so the protection is not weaker, only differently spelled.
func (m *Manager) SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	m.SetCookie(w, r, porte.SessionCookieName, token, int(m.cfg.SessionTTL.Seconds()))
}
