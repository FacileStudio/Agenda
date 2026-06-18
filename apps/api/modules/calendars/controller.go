package calendars

import (
	"net/http"
	"strconv"

	"api/internal/authcontext"
	"api/internal/httpjson"

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
	var spaceID *int64
	if v := r.URL.Query().Get("space_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			spaceID = &id
		}
	}
	cals, err := c.service.ListCalendars(r.Context(), userID, spaceID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, cals)
}

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	var req CreateCalendarRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	cal, err := c.service.CreateCalendar(r.Context(), userID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, cal)
}

func (c *controller) get(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	cal, err := c.service.GetCalendar(r.Context(), userID, calID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, cal)
}

func (c *controller) update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	var req UpdateCalendarRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	cal, err := c.service.UpdateCalendar(r.Context(), userID, calID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, cal)
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	if err := c.service.DeleteCalendar(r.Context(), userID, calID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) share(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	var req ShareCalendarRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := c.service.ShareCalendar(r.Context(), userID, calID, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) removeMember(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	memberID, _ := strconv.ParseInt(chi.URLParam(r, "memberID"), 10, 64)
	if err := c.service.RemoveMember(r.Context(), userID, calID, memberID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) listMembers(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	calID := mustCalendarID(r)
	members, err := c.service.ListMembers(r.Context(), userID, calID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, members)
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
