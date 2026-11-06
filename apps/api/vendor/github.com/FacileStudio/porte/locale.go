package porte

import "strings"

// DefaultLocale is the locale of a user whose provider asserts none, or one
// the suite does not translate.
const DefaultLocale = "en"

// NormalizeLocale maps an OIDC `locale` claim, a BCP 47 tag such as "fr-FR"
// or "de_CH", onto one of the suite's languages: fr, en, es or de. Only the
// primary subtag counts, case-insensitively. Anything else, the empty string
// included, is DefaultLocale.
func NormalizeLocale(tag string) string {
	primary, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	primary, _, _ = strings.Cut(primary, "_")
	switch primary {
	case "fr", "en", "es", "de":
		return primary
	}
	return DefaultLocale
}
