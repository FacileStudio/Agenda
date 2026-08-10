// Package errors carries the suite's error envelope: a stable machine-readable
// code, a human-readable message, and the wrapped cause. Every Facile API
// answers failures as {"error":{"code":...,"message":...}}.
package errors

import (
	stderrors "errors"
	"net/http"
)

// Error is the envelope every Facile API returns for a failure. Code is the
// stable identifier clients branch on, Message is for humans, and Cause is the
// underlying error, kept out of the JSON payload.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error returns the human-readable message, satisfying the error interface.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the wrapped cause so errors.Is and errors.As reach through the
// envelope.
func (e *Error) Unwrap() error {
	return e.Cause
}

// New builds an Error from a code, a message and an optional cause. Prefer one
// of the named constructors below unless the code is not one of them.
func New(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// Invalid reports a malformed or unacceptable request. HTTP 400.
func Invalid(message string) *Error { return New("invalid_argument", message, nil) }

// Unauthorized reports a missing or unusable credential. HTTP 401.
func Unauthorized(message string) *Error { return New("unauthenticated", message, nil) }

// Forbidden reports a valid credential that lacks the right. HTTP 403.
func Forbidden(message string) *Error { return New("permission_denied", message, nil) }

// NotFound reports that the addressed resource does not exist. HTTP 404.
func NotFound(message string) *Error { return New("not_found", message, nil) }

// Conflict reports that the resource already exists or has moved on. HTTP 409.
func Conflict(message string) *Error { return New("already_exists", message, nil) }

// Failed reports that the system state rules the operation out. HTTP 412.
func Failed(message string) *Error { return New("failed_precondition", message, nil) }

// TooLarge reports a payload the server refuses to accept. HTTP 413.
func TooLarge(message string) *Error { return New("resource_exhausted", message, nil) }

// RateLimited reports that the caller has spent its quota. HTTP 429.
func RateLimited(message string) *Error { return New("rate_limited", message, nil) }

// NotAllowed reports a verb the route does not serve. HTTP 405.
func NotAllowed(message string) *Error { return New("method_not_allowed", message, nil) }

// Unavailable reports a dependency that is down or still starting. HTTP 503.
func Unavailable(message string) *Error { return New("unavailable", message, nil) }

// Internal reports a server-side fault and keeps cause for the logs. HTTP 500.
func Internal(message string, cause error) *Error {
	return New("internal", message, cause)
}

var statusByCode = map[string]int{
	"invalid_argument":    http.StatusBadRequest,
	"unauthenticated":     http.StatusUnauthorized,
	"permission_denied":   http.StatusForbidden,
	"not_found":           http.StatusNotFound,
	"already_exists":      http.StatusConflict,
	"failed_precondition": http.StatusPreconditionFailed,
	"resource_exhausted":  http.StatusRequestEntityTooLarge,
	"rate_limited":        http.StatusTooManyRequests,
	"method_not_allowed":  http.StatusMethodNotAllowed,
	"unavailable":         http.StatusServiceUnavailable,
	"internal":            http.StatusInternalServerError,
}

// Status maps err to an HTTP status code, unwrapping through the chain to find
// an *Error. Anything it cannot recognise is a 500.
func Status(err error) int {
	var appErr *Error
	if !stderrors.As(err, &appErr) {
		return http.StatusInternalServerError
	}
	if status, ok := statusByCode[appErr.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}
