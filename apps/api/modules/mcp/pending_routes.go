package mcp

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "agenda-web"

// pendingUser is the person of a confirm page request, accepted only from
// their own browser session: a bearer token, an API key, a CLI session or a
// delegated token is refused, so an agent never confirms its own request.
func pendingUser(r *http.Request) (string, error) {
	identity, err := mcpauth.RequireOwnSession(r)
	if err != nil {
		return "", err
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
