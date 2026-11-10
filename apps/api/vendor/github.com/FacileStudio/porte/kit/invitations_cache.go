package kit

import (
	"context"
	stderrors "errors"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/tronc/apiref"
)

// ErrNoRegistreIdentity is a user who holds no identity filed under the
// configured issuer, such as a local account. Registre is not asked.
var ErrNoRegistreIdentity = stderrors.New("porte/kit: the user has no registre identity")

// InvitationsRoute documents GET InvitationsPath for tronc/apiref. Add it to
// one of the app's documented modules after MountInvitations.
func InvitationsRoute() apiref.Route {
	return apiref.Route{
		Method:  "GET",
		Path:    InvitationsPath,
		Summary: "Pending space invitations for the user menu",
		Description: "Counts the space invitations Registre holds for the signed-in user. href is the one " +
			"invitation's page when count is 1, else the issuer's /spaces page. Never fails: when the count " +
			"cannot be read the answer is {\"count\":0} without href. Cached per user for 60 seconds.",
		Auth:         "session cookie or bearer token",
		ResponseBody: "InvitationsResponse: {count: integer, href?: string}",
		Errors: []apiref.Error{
			{Status: 401, Code: "unauthenticated", Description: "No valid session, from the app's session middleware."},
		},
	}
}

func (i *Invitations) subject(ctx context.Context, userID int64) (string, error) {
	stored, err := i.cfg.Identities.ListByUser(ctx, userID)
	if err != nil {
		return "", err
	}
	issuer := strings.TrimRight(i.cfg.Directory.Issuer, "/")
	var best string
	var bestSynced time.Time
	for _, identity := range stored {
		if strings.TrimRight(identity.Provider, "/") != issuer || identity.PasswordHash != "" || identity.Subject == "" {
			continue
		}
		if best == "" || identity.SyncedAt.After(bestSynced) {
			best, bestSynced = identity.Subject, identity.SyncedAt
		}
	}
	if best == "" {
		return "", ErrNoRegistreIdentity
	}
	return best, nil
}

type cachedAnswer struct {
	answer InvitationsResponse
	at     time.Time
}

type answerCache struct {
	mu      sync.Mutex
	max     int
	entries map[int64]cachedAnswer
}

func newAnswerCache(limit int) *answerCache {
	return &answerCache{max: limit, entries: map[int64]cachedAnswer{}}
}

func (c *answerCache) get(userID int64, now time.Time) (InvitationsResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[userID]
	if !ok || now.Sub(entry.at) >= InvitationsTTL || now.Before(entry.at) {
		return InvitationsResponse{}, false
	}
	return entry.answer, true
}

func (c *answerCache) put(userID int64, answer InvitationsResponse, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[userID]; !ok && len(c.entries) >= c.max {
		c.evict(now)
	}
	c.entries[userID] = cachedAnswer{answer: answer, at: now}
}

func (c *answerCache) evict(now time.Time) {
	var oldest int64
	var oldestAt time.Time
	for userID, entry := range c.entries {
		if now.Sub(entry.at) >= InvitationsTTL {
			delete(c.entries, userID)
			continue
		}
		if oldestAt.IsZero() || entry.at.Before(oldestAt) {
			oldest, oldestAt = userID, entry.at
		}
	}
	if len(c.entries) >= c.max {
		delete(c.entries, oldest)
	}
}
