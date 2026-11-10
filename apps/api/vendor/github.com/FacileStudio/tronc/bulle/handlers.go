package bulle

import (
	"net/http"
	"net/url"

	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

func (h *ideas) list(w http.ResponseWriter, r *http.Request) {
	if !h.aiAccess(r) {
		page := IdeasPage{UpgradeURL: h.opts.UpgradeURL, IdeaList: IdeaList{Ideas: []Idea{}}}
		httpjson.WriteJSON(w, http.StatusOK, page)
		return
	}
	list, err := h.client.Ideas(r.Context(), h.opts.App, h.space(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, IdeasPage{AIAccess: true, IdeaList: list})
}

func (h *ideas) generate(w http.ResponseWriter, r *http.Request) {
	if !h.requireAI(w, r) {
		return
	}
	batch, err := h.client.Generate(r.Context(), h.opts.App, h.space(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusAccepted, batch)
}

func (h *ideas) run(w http.ResponseWriter, r *http.Request) {
	if !h.requireAI(w, r) {
		return
	}
	start, err := h.client.Run(r.Context(), param(r, "id"), h.space(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusAccepted, start)
}

func (h *ideas) dismiss(w http.ResponseWriter, r *http.Request) {
	if err := h.client.Dismiss(r.Context(), param(r, "id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ideas) status(w http.ResponseWriter, r *http.Request) {
	run, err := h.client.RunStatus(r.Context(), param(r, "run"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, run)
}

func (h *ideas) decide(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Approve bool `json:"approve"`
	}
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if body.Approve && !h.requireAI(w, r) {
		return
	}
	if err := h.client.Decide(r.Context(), param(r, "run"), param(r, "call"), body.Approve); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// param is a route parameter as the browser meant it: chi hands back the
// escaped form when the path held an escape, and the client escapes again.
func param(r *http.Request, name string) string {
	value, err := url.PathUnescape(chi.URLParam(r, name))
	if err != nil {
		return ""
	}
	return value
}
