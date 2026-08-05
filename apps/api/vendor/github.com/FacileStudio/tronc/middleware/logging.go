// Package middleware holds the HTTP middleware every Facile API runs:
// request logging with credential redaction, CORS, and panic recovery that
// answers in the suite error envelope.
package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// SensitiveQueryKeys names query parameters whose values are credentials.
// Logs leave the host, so these are replaced with [redacted] before a request
// line is written.
var SensitiveQueryKeys = []string{
	"access_token",
	"api_key",
	"apikey",
	"code",
	"id_token",
	"key",
	"password",
	"refresh_token",
	"secret",
	"signature",
	"token",
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (writer *loggingResponseWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *loggingResponseWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}

	n, err := writer.ResponseWriter.Write(body)
	writer.bytes += n
	return n, err
}

func (writer *loggingResponseWriter) Flush() {
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap exposes the wrapped writer to http.ResponseController, so streaming
// and deadline control keep working through this middleware.
func (writer *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

// Kind classifies a request for logging, so API traffic can be told apart from
// the static assets of the client the same binary now serves.
type Kind string

const (
	KindAPI    Kind = "api"
	KindHealth Kind = "health"
	KindStatic Kind = "static"
)

// RequestLoggerConfig tunes how requests are classified and at which level each
// class is recorded.
type RequestLoggerConfig struct {
	// APIPrefix marks API traffic. Defaults to "/api".
	APIPrefix string
	// QuietLevel is the level for health probes and static assets. It defaults
	// to debug, which keeps them out of Journal: the shipping handler only
	// forwards what the underlying handler admits, and apps run at info.
	QuietLevel slog.Level
}

// RequestLogger logs one record per request with the fields every Facile app
// agreed on: kind, request_id, method, path, query, remote_addr, client_ip,
// user_agent, status, bytes, duration.
//
// API requests are logged at info. Health probes and the client's static assets
// are logged at debug, because one binary now serves both halves of an app: at
// info they would bury the API traffic in bundle requests and ship thousands of
// asset lines a day to Journal.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return RequestLoggerWith(logger, RequestLoggerConfig{})
}

// RequestLoggerWith is RequestLogger with explicit classification rules.
func RequestLoggerWith(logger *slog.Logger, cfg RequestLoggerConfig) func(http.Handler) http.Handler {
	prefix := cfg.APIPrefix
	if prefix == "" {
		prefix = "/api"
	}
	quiet := cfg.QuietLevel
	if quiet == 0 {
		quiet = slog.LevelDebug
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			writer := &loggingResponseWriter{ResponseWriter: w}

			next.ServeHTTP(writer, request)

			if writer.status == 0 {
				writer.status = http.StatusOK
			}

			kind := Classify(request.URL.Path, prefix)
			level := slog.LevelInfo
			if kind != KindAPI {
				level = quiet
			}
			// A failure is worth hearing about whatever it was serving.
			if writer.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}

			logger.LogAttrs(request.Context(), level, "http request",
				slog.String("kind", string(kind)),
				slog.String("request_id", chimiddleware.GetReqID(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.String("query", RedactQuery(request.URL.RawQuery)),
				slog.String("remote_addr", request.RemoteAddr),
				slog.String("client_ip", ClientIP(request)),
				slog.String("user_agent", request.UserAgent()),
				slog.Int("status", writer.status),
				slog.Int("bytes", writer.bytes),
				slog.Duration("duration", time.Since(startedAt)),
			)
		})
	}
}

// Classify reports what a path is serving. Health probes are separated from the
// API because a container healthcheck every ten seconds is not signal.
func Classify(path, apiPrefix string) Kind {
	switch path {
	case "/health", "/ready", apiPrefix + "/health", apiPrefix + "/ready":
		return KindHealth
	}
	if path == apiPrefix || strings.HasPrefix(path, apiPrefix+"/") {
		return KindAPI
	}
	return KindStatic
}

// ClientIP reports the caller's address for logging only. It reads
// Cf-Connecting-Ip, then the first X-Forwarded-For hop, then RemoteAddr.
//
// Both headers are client-controlled unless a trusted proxy overwrites them,
// so this value must never key a rate limiter or an authorization decision.
// Use RemoteAddr for those.
func ClientIP(request *http.Request) string {
	if cloudflare := request.Header.Get("Cf-Connecting-Ip"); cloudflare != "" {
		return cloudflare
	}
	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		if comma := strings.IndexByte(forwarded, ','); comma >= 0 {
			return strings.TrimSpace(forwarded[:comma])
		}
		return strings.TrimSpace(forwarded)
	}
	return request.RemoteAddr
}

// RedactQuery replaces the value of every SensitiveQueryKeys parameter. A
// query it cannot parse is reported as [unparsable] rather than logged raw,
// because an unparsable query may still contain a token.
func RedactQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "[unparsable]"
	}

	redacted := false
	for key := range values {
		if slices.Contains(SensitiveQueryKeys, strings.ToLower(key)) {
			values.Set(key, "[redacted]")
			redacted = true
		}
	}
	if !redacted {
		return rawQuery
	}
	return values.Encode()
}
