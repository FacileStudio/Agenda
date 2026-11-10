package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// kindDeleteEvent is the one sensitive action.
const kindDeleteEvent = "delete_event"

// errTarget is the refusal for a pending action whose target is not an
// event id.
var errTarget = errors.NotFound("this action names no event")

// kinds are the sensitive actions: deleting an event fingerprints it, so an
// edit after the question closes the action instead of deleting the new
// version.
func (t *toolset) kinds() map[string]mcpkit.Kind {
	return map[string]mcpkit.Kind{
		kindDeleteEvent: {Tool: toolDeleteEvent, Fingerprint: t.fingerprint, Run: t.removeEvent},
	}
}

// refOf reads the user and the event an action targets.
func refOf(a mcpkit.Action) (int64, int64, error) {
	userID, errUser := strconv.ParseInt(a.UserID, 10, 64)
	eventID, errEvent := strconv.ParseInt(a.Target, 10, 64)
	if errUser != nil || errEvent != nil {
		return 0, 0, errTarget
	}
	return userID, eventID, nil
}

// loadEvent loads the event an action targets.
func (t *toolset) loadEvent(ctx context.Context, a mcpkit.Action) (*events.EventResponse, error) {
	userID, eventID, err := refOf(a)
	if err != nil {
		return nil, err
	}
	return t.events.GetEvent(ctx, userID, eventID)
}

// fingerprint is the digest of the targeted event as it is now.
func (t *toolset) fingerprint(ctx context.Context, a mcpkit.Action) (string, error) {
	e, err := t.loadEvent(ctx, a)
	if err != nil {
		return "", err
	}
	return digest(*e), nil
}

// removeEvent deletes the targeted event.
func (t *toolset) removeEvent(ctx context.Context, a mcpkit.Action) error {
	userID, eventID, err := refOf(a)
	if err != nil {
		return err
	}
	return t.events.DeleteEvent(ctx, userID, eventID)
}

// digest fingerprints an event: its ETag changes on every write, and the
// fields the person was shown are included for good measure.
func digest(e events.EventResponse) string {
	fields := []string{
		e.ETag, e.UpdatedAt.UTC().Format(time.RFC3339Nano), e.Title, e.StartAt.UTC().Format(time.RFC3339),
		e.EndAt.UTC().Format(time.RFC3339), e.RecurrenceRule, strconv.FormatInt(e.CalendarID, 10),
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\x00")))
	return hex.EncodeToString(sum[:16])
}

// question is what the client asks the person.
func question(e events.EventResponse) string {
	loc := eventZone(e, time.UTC)
	what := fmt.Sprintf("Delete the event %q on %s for good?", e.Title, when(e.StartAt, e.IsAllDay, loc))
	if e.RecurrenceRule != "" {
		what = fmt.Sprintf("Delete the recurring event %q, every occurrence, for good?", e.Title)
	}
	return what
}
