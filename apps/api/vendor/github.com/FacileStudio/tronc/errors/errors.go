// Package errors carries the suite's error envelope: a stable machine-readable
// code, a human-readable message, and the wrapped cause. Every Facile API
// answers failures as {"error":{"code":...,"message":...}}.
package errors

import (
	stderrors "errors"
	"net/http"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func Invalid(message string) *Error      { return New("invalid_argument", message, nil) }
func Unauthorized(message string) *Error { return New("unauthenticated", message, nil) }
func Forbidden(message string) *Error    { return New("permission_denied", message, nil) }
func NotFound(message string) *Error     { return New("not_found", message, nil) }
func Conflict(message string) *Error     { return New("already_exists", message, nil) }
func Failed(message string) *Error       { return New("failed_precondition", message, nil) }
func TooLarge(message string) *Error     { return New("resource_exhausted", message, nil) }
func RateLimited(message string) *Error  { return New("rate_limited", message, nil) }
func NotAllowed(message string) *Error   { return New("method_not_allowed", message, nil) }
func Unavailable(message string) *Error  { return New("unavailable", message, nil) }

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
