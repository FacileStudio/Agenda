package bulle

import (
	stderrors "errors"
	"net/http"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

var refusals = map[int]func(string) *errors.Error{
	http.StatusBadRequest:      errors.Invalid,
	http.StatusForbidden:       errors.Forbidden,
	http.StatusNotFound:        errors.NotFound,
	http.StatusConflict:        errors.Conflict,
	http.StatusTooManyRequests: errors.RateLimited,
}

// writeError answers the app's frontend. A 401 from Bulle is about the app's
// token, not the user's session, so it is never passed on as a 401.
func writeError(w http.ResponseWriter, err error) {
	var plan *PlanRequired
	var refused *StatusError
	var envelope *errors.Error
	switch {
	case stderrors.As(err, &plan):
		httpjson.WriteError(w, errors.PaymentRequired(orDefault(plan.Message, "Ideas need a plan with AI access.")).WithUpgradeURL(plan.UpgradeURL))
	case stderrors.Is(err, ErrBadSegment):
		httpjson.WriteError(w, errors.Invalid("invalid id"))
	case stderrors.As(err, &refused) && refusals[refused.Status] != nil:
		httpjson.WriteError(w, refusals[refused.Status](orDefault(refused.Message, "Bulle refused the request.")))
	case stderrors.As(err, &envelope):
		httpjson.WriteError(w, envelope)
	default:
		httpjson.WriteError(w, errors.Unavailable("Bulle is not reachable right now."))
	}
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
