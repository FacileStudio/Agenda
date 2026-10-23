package pool

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"math"
	"time"

	antenne "github.com/FacileStudio/antenne-client/go"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
)

const appName = "Agenda"

// Client keeps the Antenne Pool connection alive and feeds task events to a
// Mirror. The pool keeps a durable per-app ack cursor, so a restart resumes
// where the last run stopped.
type Client struct {
	client *antenne.Client
	logger *slog.Logger
}

// NewClient subscribes the mirror to the Opus task channels. Call Start to
// connect.
func NewClient(cfg env.AntenneConfig, mirror *Mirror, logger *slog.Logger) *Client {
	client := antenne.NewClient(&antenne.Config{
		App:        appName,
		Instance:   cfg.URL,
		Secret:     cfg.Secret,
		InstanceID: cfg.InstanceID,
		Events:     antenne.EventConfig{Listen: TaskChannels},
	},
		antenne.WithMaxReconnect(math.MaxInt32),
		antenne.WithOnConnect(func() { logger.Info("pool: connected") }),
		antenne.WithOnError(func(err error) {
			var transient *antenne.TransientError
			if stderrors.As(err, &transient) {
				logger.Warn("pool: reconnecting", slog.Int("attempt", transient.Attempt), slog.Any("error", transient.Err))
				return
			}
			logger.Error("pool: error", slog.Any("error", err))
		}),
	)
	for _, channel := range TaskChannels {
		channel := channel
		client.Listen(channel, func(payload json.RawMessage, meta antenne.EventMeta) {
			if err := mirror.Apply(context.Background(), channel, payload); err != nil {
				logger.Error("pool: apply failed", slog.String("channel", channel), slog.Int64("offset", meta.Offset), slog.Any("error", err))
			}
		})
	}
	return &Client{client: client, logger: logger}
}

// Start dials the pool in the background, retrying with backoff until the
// first connection succeeds. The antenne client reconnects by itself after that.
func (c *Client) Start(ctx context.Context) {
	go func() {
		backoff := 5 * time.Second
		for {
			attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.client.Connect(attempt)
			cancel()
			if err == nil {
				return
			}
			if stderrors.Is(err, antenne.ErrClosed) {
				return
			}
			c.logger.Warn("pool: dial failed, retrying", slog.Any("error", err), slog.Duration("retry_in", backoff))
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff *= 2; backoff > time.Minute {
				backoff = time.Minute
			}
		}
	}()
}

// Stop closes the pool connection.
func (c *Client) Stop() {
	c.client.Disconnect()
}
