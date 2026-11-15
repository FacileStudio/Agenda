package live

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	firstBackoff = time.Second
	maxBackoff   = 30 * time.Second
)

// Listen holds one dedicated connection LISTENing on Channel and publishes
// each notification to hub until ctx ends. A lost connection is retried with
// a doubling backoff, and every reconnect publishes a resync, since changes
// made while it was down were never delivered.
func Listen(ctx context.Context, dsn string, hub *Hub, logger *slog.Logger) {
	backoff := firstBackoff
	for attempt := 0; ctx.Err() == nil; attempt++ {
		err := listenOnce(ctx, dsn, hub, attempt > 0, func() { backoff = firstBackoff })
		if ctx.Err() != nil {
			return
		}
		logger.Warn("live: listener lost, reconnecting", slog.Any("error", err), slog.Duration("in", backoff))
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func listenOnce(ctx context.Context, dsn string, hub *Hub, resync bool, connected func()) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	connected()
	if resync {
		hub.Publish(Change{Kind: KindResync})
	}
	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if change, ok := ParseChange(notification.Payload); ok {
			hub.Publish(change)
		}
	}
}
