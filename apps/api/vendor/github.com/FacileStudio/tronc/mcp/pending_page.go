package mcp

import (
	"context"
	"net/http"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// ErrDelegated is the refusal for a confirm page request that does not come
// from the person's own session: an agent never confirms its own request.
var ErrDelegated = errors.Forbidden("only the person's own session can confirm actions")

// errNoUser is the refusal for a confirm page request with no person.
var errNoUser = errors.Unauthorized("missing auth")

// PendingItem is what every confirm page entry shows: the action, the client
// that asked and whether the target still matches the question. An app
// embeds it in its own item type, next to what the action targets.
type PendingItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	ClientID  string    `json:"client_id"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	State     string    `json:"state"`
}

// PendingList is the body of GET /mcp/pending: the person's open actions,
// newest first.
type PendingList[T any] struct {
	Actions []T `json:"actions"`

	// Truncated is true when the person has more than OpenLimit waiting
	// actions; the oldest are left out.
	Truncated bool `json:"truncated,omitempty"`
}

// PendingAPIPath is where Mount serves the confirm page's API, on the
// app's /api router.
const PendingAPIPath = "/mcp/pending"

// PendingResult is the body answered when an action is confirmed or
// dismissed. URL is set when the action made something to open, such as a
// public link.
type PendingResult struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

// EmptyRequest documents a route that takes no body.
type EmptyRequest struct{}

// PendingPage serves the API of an app's confirm page: the list of open
// actions, confirm and dismiss. T is the app's item type, which embeds
// PendingItem.
type PendingPage[T any] struct {
	// Confirmations holds the actions.
	Confirmations *Confirmations

	// Actor marks, in the audit, a confirmation made on the page, such as
	// "agenda-web".
	Actor string

	// User returns the signed-in person's user id. It must return
	// ErrDelegated, or another error, for anything but the person's own
	// session.
	User func(r *http.Request) (string, error)

	// Describe adds what the action targets, as it is now, to item.
	Describe func(ctx context.Context, item PendingItem, a Action) T

	// Done is the body answered after a confirmed run. Nil answers
	// StatusDone.
	Done func(ctx context.Context, o Outcome) PendingResult
}

// Routes mounts GET /, POST /{id}/confirm and POST /{id}/dismiss on r, which
// the app mounts at /mcp/pending behind its own authentication. A nil page
// still mounts them, for route documentation tests.
func (p *PendingPage[T]) Routes(r chi.Router) {
	r.Get("/", p.list)
	r.Post("/{id}/confirm", p.confirm)
	r.Post("/{id}/dismiss", p.dismiss)
}

// Mount mounts Routes at PendingAPIPath on router, behind auth, the app's
// authentication middleware. A nil page still mounts them, for route
// documentation tests.
func (p *PendingPage[T]) Mount(router chi.Router, auth ...func(http.Handler) http.Handler) {
	router.Route(PendingAPIPath, func(r chi.Router) {
		r.Use(auth...)
		p.Routes(r)
	})
}

// List renders userID's open actions with their targets as they are now,
// at most OpenLimit of them.
func (p *PendingPage[T]) List(ctx context.Context, userID string) (PendingList[T], error) {
	actions, truncated, err := p.Confirmations.List(ctx, userID)
	if err != nil {
		return PendingList[T]{}, err
	}
	list := PendingList[T]{Actions: make([]T, 0, len(actions)), Truncated: truncated}
	for _, a := range actions {
		item := PendingItem{
			ID: a.ID, Kind: a.Kind, ClientID: a.ClientID, Summary: a.Summary,
			CreatedAt: a.CreatedAt, ExpiresAt: a.ExpiresAt, State: p.Confirmations.State(ctx, a),
		}
		list.Actions = append(list.Actions, p.Describe(ctx, item, a))
	}
	return list, nil
}

// caller is the person of a page request.
func (p *PendingPage[T]) caller(r *http.Request) (Caller, error) {
	userID, err := p.User(r)
	switch {
	case err != nil:
		return Caller{}, err
	case userID == "":
		return Caller{}, errNoUser
	}
	return Caller{UserID: userID, Actor: p.Actor}, nil
}

// list answers the person's open actions.
func (p *PendingPage[T]) list(w http.ResponseWriter, r *http.Request) {
	who, err := p.caller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	list, err := p.List(r.Context(), who.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, list)
}

// confirm runs one action.
func (p *PendingPage[T]) confirm(w http.ResponseWriter, r *http.Request) {
	who, err := p.caller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	outcome, err := p.Confirmations.Confirm(r.Context(), who, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	result := PendingResult{Status: StatusDone}
	if p.Done != nil {
		result = p.Done(r.Context(), outcome)
	}
	httpjson.WriteJSON(w, http.StatusOK, result)
}

// dismiss closes one action without running it.
func (p *PendingPage[T]) dismiss(w http.ResponseWriter, r *http.Request) {
	who, err := p.caller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := p.Confirmations.Dismiss(r.Context(), who, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, PendingResult{Status: OutcomeDismissed})
}
