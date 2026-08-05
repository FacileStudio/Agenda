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

func (c *controller) create(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	var req CreateSpaceRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	space, err := c.service.CreateSpace(r.Context(), userID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, space)
}

func (c *controller) update(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	var req UpdateSpaceRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	space, err := c.service.UpdateSpace(r.Context(), userID, spaceID, &req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, space)
}

func (c *controller) delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	if err := c.service.DeleteSpace(r.Context(), userID, spaceID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
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

func (c *controller) addMember(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	var req AddMemberRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := c.service.AddMember(r.Context(), userID, spaceID, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) removeMember(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err := c.service.RemoveMember(r.Context(), userID, spaceID, targetID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) updateRole(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	targetID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	var req UpdateRoleRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := c.service.UpdateMemberRole(r.Context(), userID, spaceID, targetID, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (c *controller) leave(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	spaceID := mustSpaceID(r)
	if err := c.service.LeaveSpace(r.Context(), userID, spaceID); err != nil {
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

func mustSpaceID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, "spaceID"), 10, 64)
	return id
}
