package kit

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
)

// InvitationsPath is where MountInvitations registers the handler, relative
// to the router it is given: /api/me/invitations under an app's /api router.
const InvitationsPath = "/me/invitations"

// InvitationsTTL is how long one user's answer is served from memory. Every
// call costs Registre an argon2 verification, so a page load must not.
const InvitationsTTL = 60 * time.Second

// DefaultInvitationsEntries bounds the answer cache when
// InvitationsConfig.MaxEntries is zero.
const DefaultInvitationsEntries = 10000

// InvitationsResponse is the body of GET /me/invitations. Href is the one
// invitation's page when Count is 1, the issuer's /spaces page otherwise, and
// absent when the count could not be read.
type InvitationsResponse struct {
	Count int    `json:"count"`
	Href  string `json:"href,omitempty"`
}

// IdentityLister is the slice of porte.IdentityStore the invitations handler
// reads. pg.IdentityStore satisfies it.
type IdentityLister interface {
	ListByUser(ctx context.Context, userID int64) ([]porte.StoredIdentity, error)
}

// InvitationsConfig wires the invitations handler. Directory is the app's
// directory client, the one its Syncer and Deletions already use; its Issuer
// is also the provider the user's Registre identity is filed under.
type InvitationsConfig struct {
	Directory  directory.Client
	Identities IdentityLister
	Logger     *slog.Logger
	MaxEntries int
	Now        func() time.Time
}

// ErrInvitationsConfig is what NewInvitations returns without an issuer,
// client credentials or an identity store.
var ErrInvitationsConfig = stderrors.New("porte/kit: invitations need Directory.Issuer, ClientID, ClientSecret and Identities")

// Invitations answers the signed-in user's pending space invitations for the
// app's user menu. It never fails: any error answers 200 {"count":0} and is
// logged at warn. Answers, failures included, are cached per user for
// InvitationsTTL, and concurrent misses for one user share one Registre call.
type Invitations struct {
	cfg   InvitationsConfig
	cache *answerCache
	group singleflight.Group
}

// NewInvitations checks cfg and builds the handler. It makes no network call.
func NewInvitations(cfg InvitationsConfig) (*Invitations, error) {
	d := cfg.Directory
	if d.Issuer == "" || d.ClientID == "" || d.ClientSecret == "" || cfg.Identities == nil {
		return nil, ErrInvitationsConfig
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = DefaultInvitationsEntries
	}
	return &Invitations{cfg: cfg, cache: newAnswerCache(cfg.MaxEntries)}, nil
}

// MountInvitations registers GET InvitationsPath on router. Mount it inside
// the group that runs the app's session middleware. It is not part of Mount
// and changes no other route.
func MountInvitations(router chi.Router, cfg InvitationsConfig) error {
	handler, err := NewInvitations(cfg)
	if err != nil {
		return err
	}
	router.Method(http.MethodGet, InvitationsPath, handler)
	return nil
}

// ServeHTTP answers the caller's invitations from the cache or from Registre.
func (i *Invitations) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	userID := callerID(r.Context())
	if userID == 0 {
		i.cfg.Logger.Warn("porte/kit: invitations asked without a signed-in user")
		httpjson.WriteJSON(w, http.StatusOK, InvitationsResponse{})
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, i.answer(r.Context(), userID))
}

func (i *Invitations) answer(ctx context.Context, userID int64) InvitationsResponse {
	if cached, ok := i.cache.get(userID, i.cfg.Now()); ok {
		return cached
	}
	result := i.group.DoChan(strconv.FormatInt(userID, 10), func() (any, error) {
		bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), directory.InvitationsTimeout)
		defer cancel()
		answer := i.fetch(bounded, userID)
		i.cache.put(userID, answer, i.cfg.Now())
		return answer, nil
	})
	select {
	case done := <-result:
		answer, _ := done.Val.(InvitationsResponse)
		return answer
	case <-ctx.Done():
		return InvitationsResponse{}
	}
}

func (i *Invitations) fetch(ctx context.Context, userID int64) InvitationsResponse {
	subject, err := i.subject(ctx, userID)
	if err != nil {
		i.cfg.Logger.Warn("porte/kit: no registre subject for invitations",
			slog.Int64("user_id", userID), slog.Any("error", err))
		return InvitationsResponse{}
	}
	found, err := i.cfg.Directory.Invitations(ctx, subject)
	if err != nil {
		i.cfg.Logger.Warn("porte/kit: reading invitations failed, answering none",
			slog.Int64("user_id", userID), slog.Any("error", err))
		return InvitationsResponse{}
	}
	if len(found) == 1 {
		return InvitationsResponse{Count: 1, Href: found[0].URL}
	}
	return InvitationsResponse{Count: len(found), Href: strings.TrimRight(i.cfg.Directory.Issuer, "/") + "/spaces"}
}

func callerID(ctx context.Context) int64 {
	if authenticated, ok := porte.From(ctx); ok && authenticated.UserID != 0 {
		return authenticated.UserID
	}
	identity, _ := FromContext(ctx)
	return identity.UserID
}
