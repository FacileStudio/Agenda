package mcp

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

// ErrUnavailable is the text a model gets for a failure it cannot fix.
var ErrUnavailable = stderrors.New("the app could not finish this request. Retry once; if it fails again, tell the user")

// Failure turns a service error into one a model can act on. A tronc error
// for a bad argument or a missing precondition keeps its message. A missing
// record keeps its message plus hint, which should name the tool that lists
// valid ids. Anything else becomes ErrUnavailable, so internal detail stays
// out of the result. A nil err stays nil. Tools return the value as their
// error, and the SDK sends it as an isError result.
func Failure(err error, hint string) error {
	if err == nil {
		return nil
	}
	var known *errors.Error
	if !stderrors.As(err, &known) {
		return ErrUnavailable
	}
	switch known.Code {
	case "not_found":
		return fmt.Errorf("%s", strings.TrimSpace(sentence(known.Message)+". "+hint))
	case "invalid_argument", "failed_precondition", "already_exists", "permission_denied", "rate_limited", "resource_exhausted":
		return fmt.Errorf("%s", sentence(known.Message))
	default:
		return ErrUnavailable
	}
}

// sentence capitalises the first letter of a service message.
func sentence(message string) string {
	if message == "" {
		return message
	}
	return strings.ToUpper(message[:1]) + message[1:]
}
