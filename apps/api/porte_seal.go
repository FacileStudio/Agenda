package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	portepg "github.com/FacileStudio/porte/pg"
)

const porteTokenKeyEnv = "PORTE_TOKEN_KEY"

func newPorteStore(sqlDB *sql.DB) (*portepg.Store, error) {
	sealer, err := portepg.ParseTokenSealer(os.Getenv(porteTokenKeyEnv))
	if err != nil {
		return nil, fmt.Errorf("%s is set but invalid: %w", porteTokenKeyEnv, err)
	}
	return portepg.New(sqlDB, portepg.WithTokenSealer(sealer)), nil
}

func resealInBackground(ctx context.Context, store *portepg.Store, logger *slog.Logger) {
	if os.Getenv(porteTokenKeyEnv) == "" {
		return
	}
	go func() {
		count, err := store.Identities().ResealIdentities(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Warn("porte token reseal failed", slog.Int("resealed", count), slog.Any("error", err))
			return
		}
		logger.Info("porte token reseal complete", slog.Int("resealed", count))
	}()
}
