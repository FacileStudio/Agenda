package plans

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

type controller struct {
	service *Service
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := c.service.List(r.Context(), userOf(r), r.URL.Query().Get("state"), limit)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (c *controller) get(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out, err := c.service.Get(r.Context(), userOf(r), id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (c *controller) apply(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	var req ApplyRequest
	if r.ContentLength != 0 {
		if err := httpjson.DecodeJSON(w, r, &req); err != nil {
			httpjson.WriteError(w, err)
			return
		}
	}
	out, err := c.service.Apply(r.Context(), userOf(r), id, req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (c *controller) dismiss(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out, err := c.service.Dismiss(r.Context(), userOf(r), id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func userOf(r *http.Request) int64 {
	identity, _ := kit.FromContext(r.Context())
	return identity.UserID
}

func planID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "planID"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.Invalid("invalid plan id")
	}
	return id, nil
}
