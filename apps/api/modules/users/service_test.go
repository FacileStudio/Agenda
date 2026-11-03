package users

import "testing"

// The password floor counted bytes while porte's counts runes, so "aébcdé" —
// eight bytes, six characters — cleared the app's check and was then refused
// by porte, with porte's wording. The floor has to count what porte counts.
// TestThePasswordFloorCountsRunesNotBytes is the regression.
func TestThePasswordFloorCountsRunesNotBytes(t *testing.T) {
	short := "aébcdé"
	if len(short) < 8 {
		t.Fatalf("the fixture is not eight bytes: %d", len(short))
	}

	if _, _, err := normalizeProfile(&UpdateRequest{Password: &short}); err == nil {
		t.Fatal("a six-character password cleared the eight-character floor")
	}
}
