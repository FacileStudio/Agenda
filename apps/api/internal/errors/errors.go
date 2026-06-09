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

func Internal(message string, cause error) *Error {
	return New("internal", message, cause)
}

func Status(err error) int {
	var appErr *Error
	if !stderrors.As(err, &appErr) {
		return http.StatusInternalServerError
	}

	switch appErr.Code {
	case "invalid_argument":
		return http.StatusBadRequest
	case "unauthenticated":
		return http.StatusUnauthorized
	case "permission_denied":
		return http.StatusForbidden
	case "not_found":
		return http.StatusNotFound
	case "already_exists":
		return http.StatusConflict
	case "failed_precondition":
		return http.StatusPreconditionFailed
	case "resource_exhausted":
		return http.StatusRequestEntityTooLarge
	case "rate_limited":
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
