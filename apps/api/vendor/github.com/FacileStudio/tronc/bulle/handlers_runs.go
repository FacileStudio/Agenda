package bulle

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"
)

func (h *ideas) profile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.client.Profile(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, profile)
}

func (h *ideas) startRun(w http.ResponseWriter, r *http.Request) {
	if !h.requireAI(w, r) {
		return
	}
	var body StartRunRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	body.SpaceID = h.space(r)
	start, err := h.client.StartRun(r.Context(), h.opts.App, body)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusAccepted, start)
}
