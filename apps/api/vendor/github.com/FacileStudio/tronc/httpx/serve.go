package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Defaults Serve applies to a zero ServeConfig field.
const (
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 60 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultShutdownGrace     = 15 * time.Second
)

// ServeConfig tunes Serve. Every duration follows one rule: zero means the
// default, a negative value means no limit at all.
//
// No limit is the right setting for an app whose responses outlive any fixed
// budget: a model answer streamed for minutes, a multi-gigabyte upload or
// download, a WebSocket. net/http cuts a response at WriteTimeout after the
// header has promised a Content-Length, and over HTTP/2 an expired write
// deadline resets the stream with INTERNAL_ERROR. Such an app sets
// WriteTimeout (and often ReadTimeout) to -1 and bounds slow clients where
// the work happens, with http.ResponseController or WriteBudget.
type ServeConfig struct {
	// Addr is the listen address, such as ":8080". Ignored when Listener is set.
	Addr string
	// Listener serves instead of Addr, which tests use to pick a free port.
	Listener net.Listener
	// Logger receives the start and stop records. Nil means slog.Default().
	Logger *slog.Logger
	// ReadHeaderTimeout bounds reading request headers: the slow-loris guard.
	ReadHeaderTimeout time.Duration
	// ReadTimeout bounds reading the whole request, body included.
	ReadTimeout time.Duration
	// WriteTimeout bounds writing the whole response. A handler lifts or moves
	// it for one response with http.NewResponseController(w).SetWriteDeadline.
	WriteTimeout time.Duration
	// IdleTimeout bounds how long a keep-alive connection waits for its next request.
	IdleTimeout time.Duration
	// ShutdownGrace is how long in-flight requests get once the context ends.
	// Past it the remaining connections are closed.
	ShutdownGrace time.Duration
	// MaxHeaderBytes caps request headers. Zero keeps net/http's 1 MiB.
	MaxHeaderBytes int
	// OnShutdown functions start when shutdown begins, before the grace
	// period, so a long-lived stream (SSE, WebSocket) can end itself and let
	// Shutdown finish. net/http does not close hijacked connections.
	OnShutdown []func()
}

// Serve runs handler until ctx ends, then drains in-flight requests for
// ShutdownGrace. It returns nil after a clean stop, the listen error when the
// server cannot run, and an error when the grace period ran out and
// connections had to be closed.
//
// Serve never wraps the ResponseWriter, so Flusher, Hijacker and
// http.ResponseController keep working for streaming and WebSocket routes.
func Serve(ctx context.Context, handler http.Handler, cfg ServeConfig) error {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	server := newServer(handler, cfg)
	defer func() { _ = server.Close() }()
	for _, hook := range cfg.OnShutdown {
		server.RegisterOnShutdown(hook)
	}

	failed := make(chan error, 1)
	go func() { failed <- listen(server, cfg.Listener) }()
	logger.Info("server starting", slog.String("addr", address(server, cfg)))

	select {
	case err := <-failed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx: serve: %w", err)
	case <-ctx.Done():
	}
	logger.Info("server shutting down")
	return drain(server, orDefault(cfg.ShutdownGrace, DefaultShutdownGrace), logger)
}

func newServer(handler http.Handler, cfg ServeConfig) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: orDefault(cfg.ReadHeaderTimeout, DefaultReadHeaderTimeout),
		ReadTimeout:       orDefault(cfg.ReadTimeout, DefaultReadTimeout),
		WriteTimeout:      orDefault(cfg.WriteTimeout, DefaultWriteTimeout),
		IdleTimeout:       orDefault(cfg.IdleTimeout, DefaultIdleTimeout),
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}

func listen(server *http.Server, listener net.Listener) error {
	if listener != nil {
		return server.Serve(listener)
	}
	return server.ListenAndServe()
}

func address(server *http.Server, cfg ServeConfig) string {
	if cfg.Listener != nil {
		return cfg.Listener.Addr().String()
	}
	return server.Addr
}

func drain(server *http.Server, grace time.Duration, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		_ = server.Close()
		return fmt.Errorf("httpx: shutdown after %s: %w", grace, err)
	}
	logger.Info("server stopped")
	return nil
}

func orDefault(value, fallback time.Duration) time.Duration {
	switch {
	case value == 0:
		return fallback
	case value < 0:
		return 0
	}
	return value
}

// WriteBudget returns middleware that gives the response its own write
// deadline of budget from now, replacing the server's WriteTimeout for this
// route. A budget of zero or less removes the deadline. Mount it on the
// routes that stream or take long, with chi's With or Group, and leave the
// rest of the app on the server default.
//
// A writer that cannot move its deadline (an HTTP/2 stream on old Go, a test
// recorder) is left alone: the request still runs under the server default.
func WriteBudget(budget time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deadline := time.Time{}
			if budget > 0 {
				deadline = time.Now().Add(budget)
			}
			_ = http.NewResponseController(w).SetWriteDeadline(deadline)
			next.ServeHTTP(w, r)
		})
	}
}
