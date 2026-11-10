package mcp

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/go-chi/chi/v5"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "agenda-web"

// pendingUser is the signed-in person of a confirm page request. A delegated
// token is refused: an agent must never confirm its own request.
func pendingUser(r *http.Request) (mcpkit.Caller, error) {
	if identity, ok := porte.From(r.Context()); ok && identity.Actor != "" {
		return mcpkit.Caller{}, errors.Forbidden("a delegated token cannot confirm actions")
	}
	return mcpkit.Caller{UserID: authcontext.MustIdentity(r.Context()).UserID, Actor: webActor}, nil
}

// handleList answers the person's open actions with their events.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	list, err := s.tools.pendingList(r.Context(), who.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, list)
}

// handleConfirm runs one action.
func (s *Service) handleConfirm(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if _, err := s.tools.confirm.Confirm(r.Context(), who, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, PendingResult{Status: "deleted"})
}

// handleDismiss closes one action without running it. The event stays.
func (s *Service) handleDismiss(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := s.tools.confirm.Dismiss(r.Context(), who, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, PendingResult{Status: mcpkit.OutcomeDismissed})
}

// pendingList renders the person's open actions with their events as they
// are now.
func (t *toolset) pendingList(ctx context.Context, userID string) (PendingList, error) {
	actions, err := t.confirm.Open(ctx, userID)
	if err != nil {
		return PendingList{}, err
	}
	list := PendingList{Actions: make([]PendingItem, 0, len(actions))}
	for _, action := range actions {
		item := PendingItem{
			ID: action.ID, Kind: action.Kind, ClientID: action.ClientID, Summary: action.Summary,
			CreatedAt: action.CreatedAt, ExpiresAt: action.ExpiresAt, State: t.confirm.State(ctx, action),
		}
		if e, err := t.loadEvent(ctx, action); err == nil {
			item.Event = &PendingEvent{
				ID: e.ID, CalendarID: e.CalendarID, Title: e.Title, StartAt: e.StartAt, EndAt: e.EndAt,
				IsAllDay: e.IsAllDay, Location: e.Location, Recurring: e.RecurrenceRule != "",
			}
		}
		list.Actions = append(list.Actions, item)
	}
	return list, nil
}
