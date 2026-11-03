package spaces

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/httpjson"

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
	spaces, err := c.service.ListSpaces(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, spaces)
}

func (c *controller) get(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	space, err := c.service.GetSpace(r.Context(), userID, spaceID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, space)
}

func (c *controller) listMembers(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	members, err := c.service.ListMembers(r.Context(), userID, spaceID)
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

func mustSpaceID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, "spaceID"), 10, 64)
	return id
}
