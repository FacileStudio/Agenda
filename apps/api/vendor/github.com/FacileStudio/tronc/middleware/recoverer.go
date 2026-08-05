package middleware

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Recoverer turns a panic below it into a logged 500 carrying the suite error
// envelope. chi's own Recoverer writes a bare text body, which is the one
// response in these apps a client cannot parse.
//
// http.ErrAbortHandler is re-panicked, as net/http expects.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if err, ok := recovered.(error); ok && stderrors.Is(err, http.ErrAbortHandler) {
					panic(recovered)
				}

				logger.Error("panic recovered",
					slog.String("request_id", chimiddleware.GetReqID(request.Context())),
					slog.String("method", request.Method),
					slog.String("path", request.URL.Path),
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)

				httpjson.WriteError(w, errors.Internal("internal server error", nil))
			}()

			next.ServeHTTP(w, request)
		})
	}
}
