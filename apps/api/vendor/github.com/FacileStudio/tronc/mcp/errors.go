package mcp

import (
	stderrors "errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/FacileStudio/tronc/errors"
)

// ErrUnavailable is the text a model gets for a failure it cannot fix.
var ErrUnavailable = stderrors.New("the app could not finish this request. Retry once; if it fails again, tell the user")

// PlanLimit is a plan refusal from outside tronc, such as porte's
// forfait.LimitError: the feature the plan refused and where to upgrade,
// empty when there is no page.
type PlanLimit interface {
	error
	PlanLimit() (feature, upgradeURL string)
}

// Failure turns a service error into one a model can act on. A tronc error
// for a bad argument or a missing precondition keeps its message. A missing
// record keeps its message plus hint, which should name the tool that lists
// valid ids. A plan refusal, a tronc payment_required or a PlanLimit, keeps
// its message and names the upgrade page, so the model can tell the user
// where to go. An unavailable dependency keeps its message, which says what
// is down. Anything else becomes ErrUnavailable, so internal detail stays
// out of the result. A nil err stays nil. Tools return the value as their
// error, and the SDK sends it as an isError result.
func Failure(err error, hint string) error {
	if err == nil {
		return nil
	}
	var limit PlanLimit
	if stderrors.As(err, &limit) {
		return planLimit(limit)
	}
	var known *errors.Error
	if !stderrors.As(err, &known) {
		return ErrUnavailable
	}
	switch known.Code {
	case "not_found":
		return fmt.Errorf("%s", strings.TrimSpace(sentence(known.Message)+". "+hint))
	case "invalid_argument", "failed_precondition", "already_exists", "permission_denied", "rate_limited", "resource_exhausted", "unavailable":
		return fmt.Errorf("%s", sentence(known.Message))
	case "payment_required":
		return fmt.Errorf("%s", upgrade(known.Message, safeLink(known.UpgradeURL)))
	default:
		return ErrUnavailable
	}
}

// planLimit is the refusal of a PlanLimit, with its upgrade page when it is
// safe to show.
func planLimit(limit PlanLimit) error {
	feature, link := limit.PlanLimit()
	return fmt.Errorf("%s", upgrade("the plan limit for "+feature+" is reached", safeLink(link)))
}

// safeLink is link when a client may show it, and empty otherwise.
func safeLink(link string) string {
	if !errors.SafeUpgradeURL(link) {
		return ""
	}
	return link
}

// sentence capitalises the first letter of a service message.
func sentence(message string) string {
	first, size := utf8.DecodeRuneInString(message)
	if size == 0 || first == utf8.RuneError {
		return message
	}
	return string(unicode.ToUpper(first)) + message[size:]
}

// upgrade is a plan refusal's message followed by its upgrade page, when it
// has one.
func upgrade(message, link string) string {
	message = strings.TrimSuffix(sentence(message), ".")
	if link == "" {
		return message + "."
	}
	return message + ". Upgrade at " + link
}
