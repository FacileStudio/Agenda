package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/live"
	"github.com/FacileStudio/porte/kit"
)

const (
	heartbeat  = 25 * time.Second
	writeSlack = 10 * time.Second
	maxStream  = 30 * time.Minute
)

type controller struct {
	hub    *live.Hub
	access live.Access
	logger *slog.Logger
}

// stream holds the response open and writes one "change" event per change
// the user may see, plus a comment every heartbeat so proxies keep it open.
// The server's ReadTimeout and WriteTimeout would cut it within seconds, so
// the read deadline is lifted and each write moves the write deadline. A
// stream ends after maxStream; the browser reconnects through the session
// guard, so a revoked session stops receiving.
func (c *controller) stream(w http.ResponseWriter, r *http.Request) {
	identity, _ := kit.FromContext(r.Context())
	out := &sse{w: w, rc: http.NewResponseController(w)}
	if err := out.open(); err != nil {
		c.logger.Warn("live: cannot lift the connection deadlines", slog.Any("error", err))
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	changes, stop := c.hub.Subscribe()
	defer stop()
	if err := out.write("retry: 3000\n\n"); err != nil {
		return
	}
	c.serve(r.Context(), out, changes, live.NewFilter(identity.UserID, c.access))
}

func (c *controller) serve(ctx context.Context, out *sse, changes <-chan live.Change, filter *live.Filter) {
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(ctx, maxStream)
	defer cancel()
	for {
		var err error
		select {
		case <-ctx.Done():
			return
		case <-c.hub.Done():
			return
		case <-ticker.C:
			err = out.write(": ping\n\n")
		case change := <-changes:
			err = c.send(ctx, out, filter, change)
		}
		if err != nil {
			return
		}
	}
}

func (c *controller) send(ctx context.Context, out *sse, filter *live.Filter, change live.Change) error {
	frame, allowed, err := filter.Frame(ctx, change)
	if err != nil {
		c.logger.Warn("live: access check failed, change skipped", slog.Any("error", err))
	}
	if err != nil || !allowed {
		return nil
	}
	data, _ := json.Marshal(frame)
	return out.write(fmt.Sprintf("event: change\ndata: %s\n\n", data))
}

type sse struct {
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (s *sse) open() error {
	if err := s.rc.SetReadDeadline(time.Time{}); err != nil {
		return err
	}
	return s.extend()
}

func (s *sse) extend() error {
	return s.rc.SetWriteDeadline(time.Now().Add(heartbeat + writeSlack))
}

func (s *sse) write(chunk string) error {
	_ = s.extend()
	if _, err := s.w.Write([]byte(chunk)); err != nil {
		return err
	}
	return s.rc.Flush()
}
