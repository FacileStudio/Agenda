package oidcavatar

import "testing"

// The two claim shapes are what porte.facile.studio actually returns: an https URL for a
// user who uploaded a photo, and a base64 SVG of their initials for one who did not.
func TestPhotoURL(t *testing.T) {
	const uploaded = "https://porte.facile.studio/media/user-avatars/f81acba4-2d12-4d68-9b77-dbbe3bae274d.png"
	const initials = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjwvc3ZnPg=="

	cases := []struct {
		name  string
		claim string
		want  string
	}{
		{"a photo the user set in Porte", uploaded, uploaded},
		{"Authentik's generated initials are not a photo", initials, ""},
		{"no claim at all", "", ""},
		{"plain http is not Porte", "http://porte.facile.studio/media/user-avatars/x.png", ""},
		{"scheme casing still counts", "HTTPS://porte.facile.studio/x.png", "HTTPS://porte.facile.studio/x.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PhotoURL(tc.claim); got != tc.want {
				t.Errorf("PhotoURL(%.40q) = %q, want %q", tc.claim, got, tc.want)
			}
		})
	}
}
