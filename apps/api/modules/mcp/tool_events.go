package mcp

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Limits of agenda_list_events.
const (
	defaultListLimit = 50
	maxListLimit     = 100
	maxListDays      = 366
	listTruncated    = "Descriptions are shortened. Read one event in full with agenda_read_event."
)

// listPosition is the cursor of agenda_list_events: how many events earlier
// pages returned.
type listPosition struct {
	Offset int `json:"o"`
}

// listEvents is agenda_list_events.
func (t *toolset) listEvents(ctx context.Context, req *sdk.CallToolRequest, in listInput) (*sdk.CallToolResult, listOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, listOutput{}, err
	}
	full, err := mcpkit.Detailed(in.ResponseFormat)
	if err != nil {
		return nil, listOutput{}, err
	}
	loc, err := zone(in.Timezone)
	if err != nil {
		return nil, listOutput{}, err
	}
	from, to, err := window(in.From, in.To, loc, maxListDays)
	if err != nil {
		return nil, listOutput{}, err
	}
	var position listPosition
	if err := mcpkit.DecodeCursor(in.Cursor, &position); err != nil || position.Offset < 0 {
		return nil, listOutput{}, mcpkit.ErrCursor
	}
	found, err := t.eventsIn(ctx, userID, in.CalendarIDs, from, to, loc)
	if err != nil {
		return nil, listOutput{}, err
	}
	found = matching(found, in.Query)
	if position.Offset > len(found) {
		return nil, listOutput{}, mcpkit.ErrCursor
	}
	return nil, page(found, position.Offset, limitOf(in.Limit), loc, full), nil
}

// eventsIn loads the events of the calendars overlapping [from, to), with
// recurring events expanded, ordered by start.
func (t *toolset) eventsIn(ctx context.Context, userID int64, asked []int64, from, to time.Time, loc *time.Location) ([]events.EventResponse, error) {
	ids, err := t.calendarIDs(ctx, userID, asked)
	if err != nil {
		return nil, err
	}
	var all []events.EventResponse
	for _, id := range ids {
		list, err := t.events.ListEvents(ctx, userID, id, &from, &to, loc)
		if err != nil {
			return nil, mcpkit.Failure(err, hintCalendar)
		}
		for _, e := range list {
			if e.EndAt.After(from) && e.StartAt.Before(to) {
				all = append(all, e)
			}
		}
	}
	slices.SortStableFunc(all, func(a, b events.EventResponse) int {
		return cmp.Or(a.StartAt.Compare(b.StartAt), cmp.Compare(a.ID, b.ID))
	})
	return all, nil
}

// matching keeps the events whose title, description or location hold every
// word of query, ignoring case.
func matching(list []events.EventResponse, query string) []events.EventResponse {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return list
	}
	return slices.DeleteFunc(list, func(e events.EventResponse) bool {
		text := strings.ToLower(e.Title + "\n" + e.Description + "\n" + e.Location)
		return slices.ContainsFunc(words, func(word string) bool { return !strings.Contains(text, word) })
	})
}

// limitOf clamps the page size.
func limitOf(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	return min(limit, maxListLimit)
}

// page renders the events from offset, cut to the size budget, with the
// cursor of the next page.
func page(found []events.EventResponse, offset, limit int, loc *time.Location, full bool) listOutput {
	rest := found[offset:min(offset+limit, len(found))]
	summaries := make([]eventSummary, len(rest))
	for i, e := range rest {
		summaries[i] = summarize(e, loc, full)
	}
	keep := mcpkit.FitCount(len(summaries), func(n int) int {
		return mcpkit.EncodedSize(listOutput{Events: summaries[:n], Total: len(found), NextCursor: "next-cursor-placeholder"})
	})
	out := listOutput{Events: summaries[:keep], Total: len(found)}
	if offset+keep < len(found) {
		out.NextCursor = mcpkit.EncodeCursor(listPosition{Offset: offset + keep})
	}
	if full {
		out.Note = listTruncated
	}
	return out
}

// readEvent is agenda_read_event.
func (t *toolset) readEvent(ctx context.Context, req *sdk.CallToolRequest, in eventRef) (*sdk.CallToolResult, eventOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, eventOutput{}, err
	}
	e, err := t.events.GetEvent(ctx, userID, in.EventID)
	if err != nil {
		return nil, eventOutput{}, mcpkit.Failure(err, hintEvent)
	}
	return nil, fullEvent(*e), nil
}
