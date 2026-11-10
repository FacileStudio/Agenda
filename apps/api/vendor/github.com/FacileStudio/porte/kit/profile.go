package kit

import (
	"net/mail"
	"strings"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

type profile struct {
	email    string
	name     string
	picture  string
	locale   string
	verified bool
}

func (p profile) verifiedEmail() string {
	if p.verified {
		return p.email
	}
	return ""
}

func profileFrom(claims porte.Claims) (profile, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		return profile{}, errors.Invalid("the identity provider returned no usable email")
	}
	return profile{
		email:    email,
		name:     claims.DisplayName(),
		picture:  PictureURL(claims.Picture),
		locale:   claims.Locale,
		verified: claims.EmailVerified,
	}, nil
}

func localeOrDefault(locale string) string {
	if locale == "" {
		return porte.DefaultLocale
	}
	return locale
}
