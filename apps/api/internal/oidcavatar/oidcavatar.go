// Package oidcavatar reads the profile claims Authentik returns at login.
//
// It used to download the picture and keep a copy on disk. It no longer does: the URL
// Authentik hands over is public, cacheable and served by Porte, so a local copy was a
// cache of something we are not the source of — one that a container rebuild silently
// emptied while the database went on pointing at it. Storing the URL removes the copy, the
// volume it needed, and the SSRF surface that fetching an attacker-influenced URL from
// inside our own network opened.
package oidcavatar

import "strings"

// Profile carries the identity fields decoded from an OIDC id-token claim set.
type Profile struct {
	Name              string
	PreferredUsername string
	GivenName         string
	FamilyName        string
	Picture           string
}

func (p Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.PreferredUsername != "" {
		return p.PreferredUsername
	}
	full := strings.TrimSpace(p.GivenName + " " + p.FamilyName)
	if full != "" {
		return full
	}
	return ""
}

// PhotoURL returns the picture claim when it is a photo somebody actually chose, and ""
// when it is not.
//
// The distinction is the whole reason this function exists. Authentik never omits the
// claim: a user with no photo gets "data:image/svg+xml;base64,…", its own rendering of
// their initials. Testing `picture != ""` — which every app in the suite did — therefore
// reads "has an avatar" as always true, which is why an upload could never serve as the
// fallback and why the old fetch failed its own HTTPS check every five minutes in silence.
//
// An https URL is a file in Porte's media store. Anything else is a placeholder we can
// draw better ourselves, so it is reported as no photo at all.
func PhotoURL(pictureClaim string) string {
	if strings.HasPrefix(strings.ToLower(pictureClaim), "https://") {
		return pictureClaim
	}
	return ""
}
