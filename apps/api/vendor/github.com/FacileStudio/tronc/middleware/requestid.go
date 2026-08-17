package middleware

import (
	"context"
	"crypto/rand"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestIDHeader is the header a request id travels in, in both directions.
const RequestIDHeader = "X-Request-Id"

// MaxRequestIDLength bounds an id read off the wire. It is generous enough for
// a UUID, a traceparent or another Facile app's id, and small enough that a
// caller cannot append a paragraph to every log line the request produces.
const MaxRequestIDLength = 128

// requestIDBytes is the alphabet an inbound id may use: alphanumerics plus the
// separators the formats in circulation actually contain — `-` and `.` for
// UUIDs and traceparents, `_`, and `:` and `/` for the chi-style ids that
// predate this middleware. Everything else is rejected rather than escaped,
// because the value ends up in log lines, in Journal's `meta.request_id`, and
// in a URL as a filter, and none of those want to think about quoting.
func requestIDByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '-', '_', '.', ':', '/':
		return true
	}
	return false
}

// RequestID attaches an id to the request, echoes it on the response, and puts
// it where GetReqID finds it, so RequestLogger and Recoverer keep working
// unchanged.
//
// It replaces chi's middleware of the same name, for three reasons that only
// became visible once a browser started sending ids of its own:
//
//   - **chi takes the header verbatim.** Any bytes, any length. That value is
//     written to every log line the request produces, stored by Journal as
//     `meta.request_id`, and offered in the dashboard as a clickable filter. An
//     id off the wire is attacker-controlled by definition, so it is bounded and
//     charset-checked here and replaced with a fresh one when it is neither.
//   - **chi never echoes it.** A page cannot learn the id its request was logged
//     under, which is the whole mechanism that makes a browser error reachable
//     from the server logs that explain it.
//   - **chi's minted id embeds os.Hostname().** Echoing that would hand every
//     caller the container's hostname and a per-process request counter. Ours is
//     opaque and says nothing about the machine.
//
// Accepting a caller's id is deliberate: `@facile/journal` mints one before the
// request leaves the browser so that a failed fetch can name the id the server
// will log it under. That makes a request id a hint, never a credential — it
// must not authorize anything, and nothing here treats it as proof of anything.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		id := acceptRequestID(request.Header.Get(RequestIDHeader))
		if id == "" {
			id = rand.Text()
		}

		// Set before the handler runs, so the echo survives a handler that
		// writes its response immediately.
		w.Header().Set(RequestIDHeader, id)

		ctx := context.WithValue(request.Context(), chimiddleware.RequestIDKey, id)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

// RequestIDFrom returns the id RequestID attached, or "" outside the chain. It
// exists so an app can reach the id without importing chi, which is otherwise
// the only reason a handler would need to.
func RequestIDFrom(ctx context.Context) string {
	return chimiddleware.GetReqID(ctx)
}

// acceptRequestID returns the caller's id when it is well formed, or "" to say
// mint a new one.
func acceptRequestID(value string) string {
	if value == "" || len(value) > MaxRequestIDLength {
		return ""
	}
	for i := 0; i < len(value); i++ {
		if !requestIDByte(value[i]) {
			return ""
		}
	}
	return value
}
