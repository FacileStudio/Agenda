package mcp

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "agenda-web"

// pendingUser is the signed-in person of a confirm page request. A delegated
// token is refused: an agent must never confirm its own request.
func pendingUser(r *http.Request) (string, error) {
	if identity, ok := porte.From(r.Context()); ok && identity.Actor != "" {
		return "", mcpkit.ErrDelegated
	}
	identity, ok := kit.FromContext(r.Context())
	if !ok {
		return "", nil
	}
	return strconv.FormatInt(identity.UserID, 10), nil
}

// describe adds to one open action the event it would delete, as it is now.
func (t *toolset) describe(ctx context.Context, item mcpkit.PendingItem, a mcpkit.Action) PendingItem {
	out := PendingItem{PendingItem: item}
	if e, err := t.loadEvent(ctx, a); err == nil {
		out.Event = &PendingEvent{
			ID: e.ID, CalendarID: e.CalendarID, Title: e.Title, StartAt: e.StartAt, EndAt: e.EndAt,
			IsAllDay: e.IsAllDay, Location: e.Location, Recurring: e.RecurrenceRule != "",
		}
	}
	return out
}

// done is the answer to a confirmed action: the event is gone.
func done(context.Context, mcpkit.Outcome) mcpkit.PendingResult {
	return mcpkit.PendingResult{Status: "deleted"}
}
