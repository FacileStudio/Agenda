package live

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/live"
	"github.com/FacileStudio/porte/kit"
)

func readable(_ context.Context, _, calendarID int64) (bool, error) { return calendarID == 1, nil }

// streamServer serves user 7's stream with deadlines far shorter than the
// heartbeat, the way main.go's 15 s WriteTimeout is.
func streamServer(t *testing.T, hub *live.Hub) *httptest.Server {
	t.Helper()
	c := &controller{hub: hub, access: readable, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.stream(w, r.WithContext(kit.WithIdentity(r.Context(), kit.Identity{UserID: 7})))
	}))
	server.Config.WriteTimeout = 300 * time.Millisecond
	server.Config.ReadTimeout = 300 * time.Millisecond
	server.Config.RegisterOnShutdown(hub.Close)
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func firstData(lines *bufio.Scanner) string {
	for lines.Scan() {
		if line := lines.Text(); strings.HasPrefix(line, "data: ") {
			return line
		}
	}
	return ""
}

func TestStreamOutlivesTheWriteTimeoutAndFiltersFrames(t *testing.T) {
	hub := live.NewHub()
	server := streamServer(t, hub)

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}
	lines := bufio.NewScanner(resp.Body)

	time.Sleep(time.Second)
	hub.Publish(live.Change{Kind: live.KindEvent, ID: 2})
	hub.Publish(live.Change{Kind: live.KindActivity, ID: 8})
	hub.Publish(live.Change{Kind: live.KindEvent, ID: 1})

	if data := firstData(lines); data != `data: {"calendar_id":1}` {
		t.Fatalf("first frame after the write timeout = %q, want only calendar 1", data)
	}

	if err := server.Config.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown waited on the stream: %v", err)
	}
}
