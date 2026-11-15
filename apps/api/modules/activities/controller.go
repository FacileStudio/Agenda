package activities

import (
	"net/http"
	"time"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type controller struct {
	service *Service
}

func (c *controller) list(w http.ResponseWriter, r *http.Request) {
	identity, _ := kit.FromContext(r.Context())
	from, errFrom := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, errTo := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if errFrom != nil || errTo != nil {
		httpjson.WriteError(w, errors.Invalid("from and to must be RFC 3339 instants"))
		return
	}
	out, err := c.service.List(r.Context(), identity.UserID, from, to)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}
