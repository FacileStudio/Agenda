// Package logger builds the suite's structured logger: JSON to stdout, one
// level parsed from configuration, and a seam for wrapping the handler.
//
// The seam is how log shipping stays out of tronc. An app that ships to
// Journal supplies Config.Wrap and keeps the dependency in its own go.mod:
//
//	log := logger.New(logger.Config{Level: env.LogLevel, Wrap: func(h slog.Handler) slog.Handler {
//		if env.JournalURL == "" {
//			return h
//		}
//		return journal.NewHandler(journal.New(journal.Config{URL: env.JournalURL, Token: env.JournalToken}), h)
//	}})
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config describes the logger an app wants. The zero value is valid and
// yields info-level JSON on stdout.
type Config struct {
	// Level is one of debug, info, warn, error. Anything else means info.
	Level string
	// Output defaults to os.Stdout.
	Output io.Writer
	// Wrap, when set, replaces the handler with the one it returns. It is
	// called exactly once, at construction.
	Wrap func(slog.Handler) slog.Handler
}

// New builds a logger from cfg.
func New(cfg Config) *slog.Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	var handler slog.Handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: ParseLevel(cfg.Level),
	})
	if cfg.Wrap != nil {
		handler = cfg.Wrap(handler)
	}
	return slog.New(handler)
}

// ParseLevel maps a configured level name onto a slog.Level, defaulting to
// info for anything it does not recognise.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
