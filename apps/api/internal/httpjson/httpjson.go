package httpjson

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"

	"api/internal/errors"
)

const maxJSONBodyBytes int64 = 1 << 20

func DecodeJSON(w http.ResponseWriter, request *http.Request, dst any) error {
	defer request.Body.Close()
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("invalid JSON body")
	}
	if err := decoder.Decode(new(struct{})); !stderrors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("request body must contain a single JSON object")
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

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
