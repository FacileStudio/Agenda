package events

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func (c *controller) updateOccurrence(w http.ResponseWriter, r *http.Request) {
	var req OccurrenceRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	evt, err := c.service.UpdateOccurrence(r.Context(), mustUserID(r), mustEventID(r), chi.URLParam(r, "when"), &req)
	writeEvent(w, evt, err)
}

func (c *controller) cancelOccurrence(w http.ResponseWriter, r *http.Request) {
	evt, err := c.service.CancelOccurrence(r.Context(), mustUserID(r), mustEventID(r), chi.URLParam(r, "when"))
	writeEvent(w, evt, err)
}

func (c *controller) restoreOccurrence(w http.ResponseWriter, r *http.Request) {
	evt, err := c.service.RestoreOccurrence(r.Context(), mustUserID(r), mustEventID(r), chi.URLParam(r, "when"))
	writeEvent(w, evt, err)
}

func writeEvent(w http.ResponseWriter, evt *EventResponse, err error) {
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, evt)
}
