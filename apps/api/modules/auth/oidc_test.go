package auth

import "testing"

func TestEmailClaimTrustedRefusesOnlyAnExplicitDenial(t *testing.T) {
	cases := []struct {
		name    string
		claim   any
		trusted bool
	}{
		{"absent claim is trusted, or every pre-subject account is stranded", nil, true},
		{"verified", true, true},
		{"unverified", false, false},
		{"verified as a string", "true", true},
		{"unverified as a string", "false", false},
		{"unverified as a string, mixed case", "False", false},
		{"unexpected type", 1, false},
	}
	for _, c := range cases {
		if got := emailClaimTrusted(c.claim); got != c.trusted {
			t.Errorf("%s: emailClaimTrusted(%#v) = %v, want %v", c.name, c.claim, got, c.trusted)
		}
	}
}
