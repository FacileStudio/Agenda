package oidc

import (
	"strings"

	"golang.org/x/oauth2"

	"github.com/FacileStudio/porte"
)

// localeClaim normalises the raw `locale` claim. It arrives as any so a
// provider that sends a non-string never fails the whole login. A missing or
// non-string claim is empty, so the store keeps the locale it already holds
// and a new row takes its column default.
func localeClaim(value any) string {
	tag, ok := value.(string)
	if !ok {
		return ""
	}
	return porte.NormalizeLocale(tag)
}

// normalizeEmail trims and lowercases an address claim, as porte/local does,
// so one mailbox spelled two ways cannot become two accounts.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// uiLocales forwards the sign-in page's language to the provider as the OIDC
// ui_locales parameter, when it is exactly one of the suite's four. Anything
// else, a list included, is dropped rather than passed through: the value
// lands in the provider's URL, and the provider falls back to its own default.
func uiLocales(raw string) []oauth2.AuthCodeOption {
	tag := strings.ToLower(strings.TrimSpace(raw))
	if tag == "" || porte.NormalizeLocale(tag) != tag {
		return nil
	}
	return []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("ui_locales", tag)}
}
