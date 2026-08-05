package events

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/Agenda/apps/api/internal/httpjson"

	"github.com/go-chi/chi/v5"
)

type controller struct {
	service *Service
}

func newController(service *Service) *controller {
	return &controller{service: service}
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)

	var from, to *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			from = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			to = &t
		}
	}

	evts, err := c.service.ListEvents(r.Context(), userID, calID, from, to)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, evts)
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	var req CreateEventRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	evt, err := c.service.CreateEvent(r.Context(), userID, calID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, evt)
}

func (c *controller) get(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	evtID := mustEventID(r)
	evt, err := c.service.GetEvent(r.Context(), userID, evtID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, evt)
}

func (c *controller) update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	evtID := mustEventID(r)
	var req UpdateEventRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	evt, err := c.service.UpdateEvent(r.Context(), userID, evtID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, evt)
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	evtID := mustEventID(r)
	if err := c.service.DeleteEvent(r.Context(), userID, evtID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func mustUserID(r *http.Request) int64 {
	identity := authcontext.MustIdentity(r.Context())
	id, _ := strconv.ParseInt(identity.UserID, 10, 64)
	return id
}

func mustCalendarID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, "calendarID"), 10, 64)
	return id
}

func mustEventID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, "eventID"), 10, 64)
	return id
}
