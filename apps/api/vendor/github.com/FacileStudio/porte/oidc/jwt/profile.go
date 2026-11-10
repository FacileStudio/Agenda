package jwt

// Profile is the part of a token that describes the person, read by an app
// that provisions an account from a bearer token rather than from a login.
// Registre carries it on access tokens bound to an MCP resource only.
type Profile struct {
	// EmailVerified is the email_verified claim: nil when the token asserts
	// nothing, which is not the same answer as false.
	EmailVerified *bool

	PreferredUsername string
	Picture           string
	Locale            string
}

// profileClaims is the wire form of Profile. email_verified and locale are
// decoded loosely, so a provider that sends an unexpected type for either
// loses that claim instead of having every token refused.
type profileClaims struct {
	EmailVerified     any    `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
	Locale            any    `json:"locale"`
}

func (p profileClaims) profile() Profile {
	locale, _ := p.Locale.(string)
	return Profile{
		EmailVerified:     verifiedClaim(p.EmailVerified),
		PreferredUsername: p.PreferredUsername,
		Picture:           p.Picture,
		Locale:            locale,
	}
}

// verifiedClaim reads email_verified as a JSON boolean or its string spelling,
// and answers nil for silence or anything malformed.
func verifiedClaim(value any) *bool {
	var verified bool
	switch typed := value.(type) {
	case bool:
		verified = typed
	case string:
		switch typed {
		case "true", "True", "TRUE", "1":
			verified = true
		case "false", "False", "FALSE", "0":
		default:
			return nil
		}
	default:
		return nil
	}
	return &verified
}
