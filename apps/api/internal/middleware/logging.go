package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

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

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			writer := &loggingResponseWriter{ResponseWriter: w}

			next.ServeHTTP(writer, request)

			if writer.status == 0 {
				writer.status = http.StatusOK
			}

			logger.Info("http request",
				slog.String("request_id", chimiddleware.GetReqID(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.String("query", redactQuery(request.URL.RawQuery)),
				slog.String("remote_addr", request.RemoteAddr),
				slog.Int("status", writer.status),
				slog.Int("bytes", writer.bytes),
				slog.Duration("duration", time.Since(startedAt)),
			)
		})
	}
}

func redactQuery(rawQuery string) string {
	if rawQuery == "" || !strings.Contains(rawQuery, "token=") {
		return rawQuery
	}
	parsed, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	if parsed.Get("token") != "" {
		parsed.Set("token", "[REDACTED]")
	}
	return parsed.Encode()
}
