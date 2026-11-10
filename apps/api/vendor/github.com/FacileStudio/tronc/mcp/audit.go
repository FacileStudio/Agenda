package mcp

import (
	"context"
	"log/slog"
)

// Event is one audited write: the tool, how it ended, who asked through
// which client, and the app's own name for what it touched.
type Event struct {
	Tool    string
	Outcome string
	Caller  Caller
	Target  string
}

// Audit logs one write. It never logs arguments or content, so a mail body
// or an event title never reaches the logs. The suite's logger ships to
// Journal.
func Audit(ctx context.Context, logger *slog.Logger, e Event) {
	logger.InfoContext(ctx, "mcp write",
		slog.String("tool", e.Tool),
		slog.String("outcome", e.Outcome),
		slog.String("user_id", e.Caller.UserID),
		slog.String("client_id", e.Caller.ClientID),
		slog.String("actor", e.Caller.Actor),
		slog.String("target", e.Target),
	)
}
