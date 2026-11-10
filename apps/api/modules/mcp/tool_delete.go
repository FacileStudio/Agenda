package mcp

import (
	"context"
	"errors"
	"strconv"

	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Refusals checked before the person is asked, so they are never asked to
// confirm what would fail.
var (
	errFromOpus = errors.New("this event comes from an Opus task and is read-only: edit or delete the task in Opus")
	errReader   = errors.New("the user can only read this calendar, so the event cannot be deleted")
)

// waitingDelete tells the model that nothing happened yet, and what to do.
const waitingDelete = "Nothing was deleted: your client cannot ask the user to confirm. Give the user confirm_url " +
	"and ask them to confirm there. Do not call this tool again for the same event; the request stays open for 7 days."

// deleteEvent is agenda_delete_event.
func (t *toolset) deleteEvent(ctx context.Context, req *sdk.CallToolRequest, in eventRef) (*sdk.CallToolResult, actionOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resumeDelete(ctx, req)
	}
	userID, err := userOf(req)
	if err != nil {
		return nil, actionOutput{}, err
	}
	e, err := t.events.GetEvent(ctx, userID, in.EventID)
	if err != nil {
		return nil, actionOutput{}, mcpkit.Failure(err, hintEvent)
	}
	if e.Source != "" {
		return nil, actionOutput{}, errFromOpus
	}
	if err := t.canWrite(ctx, userID, e.CalendarID); err != nil {
		return nil, actionOutput{}, err
	}
	result, outcome, err := t.confirm.Request(ctx, req, kindDeleteEvent, strconv.FormatInt(in.EventID, 10), question(*e))
	if err != nil {
		return nil, actionOutput{}, mcpkit.Failure(err, hintEvent)
	}
	if result != nil {
		return result, actionOutput{}, nil
	}
	return nil, actionOutput{Status: outcome.Status, EventID: in.EventID, ConfirmURL: outcome.ConfirmURL, Message: waitingDelete}, nil
}

// canWrite refuses a calendar the user can only read.
func (t *toolset) canWrite(ctx context.Context, userID, calendarID int64) error {
	c, err := t.calendarOf(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if c.Role == "reader" || c.Role == "" {
		return errReader
	}
	return nil
}

// resumeDelete handles the retry that carries the person's answer.
func (t *toolset) resumeDelete(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, actionOutput, error) {
	outcome, err := t.confirm.Resume(ctx, req, kindDeleteEvent)
	if err != nil {
		return nil, actionOutput{}, mcpkit.Failure(err, "Call agenda_delete_event again to ask the user afresh.")
	}
	if outcome.Status == mcpkit.StatusDeclined {
		return nil, actionOutput{Status: mcpkit.StatusDeclined, Message: "The user declined. Nothing happened, and the event is kept."}, nil
	}
	_, eventID, _ := refOf(outcome.Action)
	return nil, actionOutput{Status: "deleted", EventID: eventID, Message: "The user confirmed and the event was deleted."}, nil
}
