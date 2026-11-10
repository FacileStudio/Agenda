package mcp

import (
	"context"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// listCalendars is agenda_list_calendars.
func (t *toolset) listCalendars(ctx context.Context, req *sdk.CallToolRequest, _ calendarsInput) (*sdk.CallToolResult, calendarsOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, calendarsOutput{}, err
	}
	list, err := t.calendars.ListCalendars(ctx, userID, nil)
	if err != nil {
		return nil, calendarsOutput{}, mcpkit.Failure(err, "")
	}
	out := calendarsOutput{Calendars: make([]calendarSummary, 0, len(list))}
	for _, c := range list {
		out.Calendars = append(out.Calendars, calendarSummary{
			CalendarID: c.ID, Name: c.Name, Color: c.Color, Role: c.Role, Personal: c.IsPersonal,
			InSpace: c.SpaceID != nil, Description: c.Description,
		})
	}
	return nil, out, nil
}

// calendarIDs is the calendars a read covers: the ones asked for, or every
// calendar the user can open.
func (t *toolset) calendarIDs(ctx context.Context, userID int64, asked []int64) ([]int64, error) {
	if len(asked) > 0 {
		return asked, nil
	}
	list, err := t.calendars.ListCalendars(ctx, userID, nil)
	if err != nil {
		return nil, mcpkit.Failure(err, "")
	}
	ids := make([]int64, len(list))
	for i, c := range list {
		ids[i] = c.ID
	}
	return ids, nil
}

// calendarOf is one calendar the user can open, or the zero value when
// they cannot.
func (t *toolset) calendarOf(ctx context.Context, userID, calendarID int64) (calendars.CalendarResponse, error) {
	list, err := t.calendars.ListCalendars(ctx, userID, nil)
	if err != nil {
		return calendars.CalendarResponse{}, mcpkit.Failure(err, "")
	}
	for _, c := range list {
		if c.ID == calendarID {
			return c, nil
		}
	}
	return calendars.CalendarResponse{}, nil
}
