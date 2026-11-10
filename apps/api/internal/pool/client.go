package pool

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	antenne "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/antenne-client/go/outbox"
)

// Client keeps the Antenne Pool connection alive, feeds task events to a
// Mirror and drains Agenda's outbox onto the pool. The pool keeps a durable
// per-app ack cursor, so a restart resumes where the last run stopped, and
// queued changes wait in antenne_outbox while the pool is down.
type Client struct {
	client  *antenne.Client
	drainer *outbox.Drainer
	logger  *slog.Logger
	cancel  context.CancelFunc
}

// NewClient subscribes the mirror to the Opus task channels and declares the
// calendar channels Agenda publishes. Call Start to connect.
func NewClient(cfg antenne.Config, mirror *Mirror, db *sql.DB, logger *slog.Logger) *Client {
	channels := TaskChannels()
	cfg.Events.Listen = channels
	cfg.Events.Emit = emitted()
	client := antenne.NewClient(&cfg,
		antenne.WithPersistentConnect(),
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
	for _, channel := range channels {
		client.Listen(channel, handler(mirror, channel, logger))
	}
	return &Client{client: client, drainer: newDrainer(db, client, logger), logger: logger}
}

// newDrainer sends the outbox on client, logging each published channel and
// treating a pool that is not connected yet as a wait rather than a failure.
func newDrainer(db *sql.DB, client *antenne.Client, logger *slog.Logger) *outbox.Drainer {
	emit := func(channel string, payload any) error {
		if err := client.EmitNow(channel, payload); err != nil {
			return err
		}
		logger.Info("pool: published", slog.String("channel", channel))
		return nil
	}
	return outbox.NewDrainer(db, emit, outbox.WithOnError(func(err error) {
		if stderrors.Is(err, antenne.ErrNotConnected) {
			logger.Debug("pool: outbox waits for the connection")
			return
		}
		logger.Warn("pool: outbox drain failed", slog.Any("error", err))
	}))
}

func emitted() []string {
	channels := bus.Channels()
	out := make([]string, len(channels))
	for i, channel := range channels {
		out[i] = string(channel)
	}
	return out
}

const applyTimeout = 30 * time.Second

func handler(mirror *Mirror, channel string, logger *slog.Logger) antenne.EventHandler {
	return func(payload json.RawMessage, meta antenne.EventMeta) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("pool: handler panic", slog.String("channel", channel), slog.Int64("offset", meta.Offset), slog.Any("panic", r))
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
		defer cancel()
		if err := mirror.Apply(ctx, channel, payload); err != nil {
			logger.Error("pool: apply failed", slog.String("channel", channel), slog.Int64("offset", meta.Offset), slog.Any("error", err))
		}
	}
}

// Start connects in the background and runs the outbox drainer. The antenne
// client retries the first connection and every later one by itself.
func (c *Client) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	if err := c.client.Connect(ctx); err != nil {
		c.logger.Error("pool: connect", slog.Any("error", err))
	}
	go c.drainer.Run(ctx)
}

// Stop ends the drainer and closes the pool connection.
func (c *Client) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.client.Disconnect()
}
