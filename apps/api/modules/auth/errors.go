package auth

import (
	stderrors "errors"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

// TranslateError replaces a porte sentinel's own text with wording meant for
// the caller, keeping the status code and the sentinel itself so that
// errors.Is still reaches through downstream.
//
// porte builds its refusals by copying the sentinel into the envelope's
// message, and every sentinel reads "porte: …". tronc writes that message
// straight into the response body, so an unmapped one names a library the API
// mentions nowhere else — "porte: this account has no password" is what a
// federated account got back from a password change. The users module already
// had to remap ErrPasswordSet by hand for the same reason; doing it once here,
// at the only place that calls porte, covers the ones nobody has hit yet.
func TranslateError(err error) error {
	message := porteMessage(err)
	if message == "" {
		return err
	}
	var envelope *errors.Error
	if !stderrors.As(err, &envelope) {
		return err
	}
	return errors.New(envelope.Code, message, err)
}

// porteMessage returns Agenda's wording for a porte sentinel, or "" when the
// error is not one of them. ErrNotFound is absent on purpose: porte returns it
// as a lookup result rather than as an answer to a caller.
func porteMessage(err error) string {
	switch {
	case err == nil:
		return ""
	case stderrors.Is(err, porte.ErrWrongPassword):
		return "invalid email or password"
	case stderrors.Is(err, porte.ErrEmailTaken):
		return "an account with this email already exists"
	case stderrors.Is(err, porte.ErrRegistrationClosed):
		return "registration is disabled"
	case stderrors.Is(err, porte.ErrWeakPassword):
		return "password must be at least 8 characters"
	case stderrors.Is(err, porte.ErrInvalidEmail):
		return "invalid email"
	case stderrors.Is(err, porte.ErrNoPassword):
		return "this account has no password to change — set one instead"
	case stderrors.Is(err, porte.ErrPasswordSet):
		return "this account already has a password"
	case stderrors.Is(err, porte.ErrCodeConsumed):
		return "this login code has already been used"
	}
	return ""
}
