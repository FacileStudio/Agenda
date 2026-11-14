package session

import (
	"strings"

	"github.com/FacileStudio/porte"
)

// browserSession reports whether a credential is a browser login: it came in
// the session cookie, its row is not a named API token, and IssueCookieFor
// minted it. A session token a CLI, device or mobile exchange received, sent
// as a cookie, is not one, and neither is a cookie issued before v0.20.1,
// including a v0.20.0 browser token, whose "b-" mark a plain token can carry.
func browserSession(fromCookie bool, token string, stored porte.Session) bool {
	return fromCookie && !stored.IsAPIToken() && strings.HasPrefix(token, porte.BrowserTokenPrefix)
}

// plainToken wraps a mint so it never returns a token carrying
// porte.BrowserTokenPrefix: a session handed to a CLI, a device or a mobile
// app must not pass as a browser login when it is sent as the cookie.
func plainToken(mint func() (string, error)) func() (string, error) {
	return func() (string, error) {
		for {
			token, err := mint()
			if err != nil || !strings.HasPrefix(token, porte.BrowserTokenPrefix) {
				return token, err
			}
		}
	}
}
