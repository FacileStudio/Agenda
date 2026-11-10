package mcp

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
)

// TestServe runs the endpoint over the scratch database on MCP_SERVE until
// interrupted, for the conformance suite and MCP Inspector. A request
// without a token gets a valid one added, because the conformance suite
// sends none; MCP_SERVE_TOKEN=1 prints a token for Inspector instead.
func TestServe(t *testing.T) {
	addr := os.Getenv("MCP_SERVE")
	if addr == "" {
		t.Skip("set MCP_SERVE=127.0.0.1:<port> to serve the endpoint")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	f := newFixtureOn(t, listener, func(f *fixture, next http.Handler) http.Handler {
		token := f.issuer.Mint(t, map[string]any{"aud": []string{f.resource}, "sub": subject, "scope": "agenda.read agenda.write",
			"exp": time.Now().Add(12 * time.Hour).Unix()})
		t.Logf("resource %s\ntoken %s", f.resource, token)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" && os.Getenv("MCP_SERVE_TOKEN") == "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			next.ServeHTTP(w, r)
		})
	})
	start := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
	f.event(t, events.CreateEventRequest{Title: "Kickoff", StartAt: start, EndAt: start.Add(time.Hour)})
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
