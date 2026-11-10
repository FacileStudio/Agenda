package mcp

import (
	"errors"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool names. Every name carries the app prefix, so a client holding many
// servers can tell them apart.
const (
	toolListCalendars = "agenda_list_calendars"
	toolListEvents    = "agenda_list_events"
	toolReadEvent     = "agenda_read_event"
	toolFreeSlots     = "agenda_find_free_slots"
	toolCreateEvent   = "agenda_create_event"
	toolUpdateEvent   = "agenda_update_event"
	toolDeleteEvent   = "agenda_delete_event"
)

// Hints appended to a not_found error, naming the tool that lists valid ids.
const (
	hintCalendar = "Call agenda_list_calendars for the calendar ids you can use."
	hintEvent    = "Call agenda_list_events for the event ids in a date range."
)

// errNoCaller is the refusal for a token that resolves to no Agenda user.
var errNoCaller = errors.New("this request carries no valid access token. Reconnect Agenda")

// toolset holds what the tool handlers share.
type toolset struct {
	calendars *calendars.Service
	events    *events.Service
	confirm   *mcpkit.Confirmations
	appURL    string
}

// userOf reads the Agenda user id from the verified token of the request.
func userOf(req *sdk.CallToolRequest) (int64, error) {
	identity, ok := mcpauth.Identity(req.Extra.TokenInfo)
	if !ok {
		return 0, errNoCaller
	}
	return identity.UserID, nil
}
