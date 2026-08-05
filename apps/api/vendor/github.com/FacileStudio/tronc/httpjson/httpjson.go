// Package httpjson reads and writes the suite's JSON request and response
// bodies. Decoding is bounded and strict; encoding always goes through
// WriteJSON so responses cannot drift apart across apps.
package httpjson

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"

	"github.com/FacileStudio/tronc/errors"
)

// MaxBodyBytes bounds every decoded request body.
const MaxBodyBytes int64 = 1 << 20

// DecodeJSON reads exactly one JSON object from the request into dst. Unknown
// fields are rejected, the body is capped at MaxBodyBytes, and trailing data
// is an error. The returned error is always an *errors.Error.
func DecodeJSON(w http.ResponseWriter, request *http.Request, dst any) error {
	return DecodeJSONLimit(w, request, dst, MaxBodyBytes)
}

// DecodeJSONLimit is DecodeJSON with an explicit cap, for the endpoints whose
// payloads are legitimately larger than MaxBodyBytes — a log-ingest batch, for
// instance. Prefer DecodeJSON everywhere else.
func DecodeJSONLimit(w http.ResponseWriter, request *http.Request, dst any, maxBytes int64) error {
	defer func() { _ = request.Body.Close() }()
	request.Body = http.MaxBytesReader(w, request.Body, maxBytes)

	return decodeStrict(request.Body, dst)
}

func decodeStrict(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return decodeError(err, "invalid JSON body")
	}
	if err := decoder.Decode(new(struct{})); !stderrors.Is(err, io.EOF) {
		return decodeError(err, "request body must contain a single JSON object")
	}
	return nil
}

func decodeError(err error, message string) error {
	var maxBytesErr *http.MaxBytesError
	if stderrors.As(err, &maxBytesErr) {
		return errors.TooLarge("request body too large")
	}
	if stderrors.Is(err, errDecompressedTooLarge) {
		return errors.TooLarge("decompressed body too large")
	}
	return errors.Invalid(message)
}

// WriteJSON writes value as JSON with the given status. A nil value writes the
// status and no body.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError writes err as {"error":{"code":...,"message":...}} with the status
// its code maps to. Any error that is not an *errors.Error becomes a generic
// internal error, so a raw failure can never leak its text to a client.
func WriteError(w http.ResponseWriter, err error) {
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) {
		appErr = errors.Internal("internal server error", err)
	}

	WriteJSON(w, errors.Status(appErr), map[string]any{
		"error": map[string]string{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}
