// Package directory is how a Facile app keeps its shared spaces identical to
// Registre's.
//
// Registre owns spaces, their members and their roles. An app never edits
// them; it applies the snapshot Registre serves, so every app shows the same
// spaces with the same people at the same rank. The snapshot is the whole
// truth rather than a stream of changes, which makes applying it idempotent:
// a missed, repeated or reordered notification costs nothing, because the
// next pull simply carries the current state.
//
// An app pulls at boot, on a fixed interval, and within seconds of Registre
// posting a nudge to the app's sync route after any change. The nudge carries
// nothing and is not trusted; it only makes the next pull happen sooner, and
// the pull itself is authenticated with the app's own OIDC client credentials.
package directory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Roles are the three ranks Registre knows, lowest to highest. An app with a
// finer vocabulary maps them onto its own and never sends anything else back.
const (
	RoleMember = "member"
	RoleAdmin  = "admin"
	RoleOwner  = "owner"
)

// SyncPath is the route every app mounts Handler on. Registre's register
// records the full URL per app; the path is fixed so the suite has one shape.
const SyncPath = "/api/registre/sync"

// Member is one person in a space. Subject is their Registre `sub`, the key an
// app's porte identity rows already match on. Picture is the stable URL the
// picture claim carries and answers 404 until one is uploaded.
type Member struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Role    string `json:"role"`
}

// Space is one shared space and everyone in it. ID is Registre's, stable
// across renames; an app stores it on its own space row to link the two.
type Space struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

// Snapshot is every shared space as Registre holds it now. Version rises on
// every change, so an app can skip applying a snapshot it already applied.
type Snapshot struct {
	Version int64   `json:"version"`
	Spaces  []Space `json:"spaces"`
}

// ImportMember is one person an app already had in a space before Registre
// owned spaces. Subject is preferred; Email is the fallback Registre matches
// on when the app never stored the subject.
type ImportMember struct {
	Subject string `json:"sub,omitempty"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role"`
}

// ImportSpace is one shared space an app held before Registre owned spaces.
// Personal spaces are never imported.
type ImportSpace struct {
	Name    string         `json:"name"`
	Members []ImportMember `json:"members"`
}

// Client talks to Registre's directory API with the app's own client
// credentials, the same pair the app already signs users in with.
type Client struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	HTTP         *http.Client
}

// ErrUnavailable wraps every failure to reach Registre or read its answer, so
// a caller can keep serving its last applied state and retry later.
var ErrUnavailable = errors.New("directory: registre unavailable")

// Snapshot fetches every shared space as Registre holds it now.
func (c Client) Snapshot(ctx context.Context) (Snapshot, error) {
	return c.call(ctx, http.MethodGet, "/api/directory/snapshot", nil)
}

// Import hands Registre the shared spaces this app held before Registre owned
// them, once. Registre merges them by name, case-insensitively, keeps the
// highest role anyone held, and remembers the app imported: a second call is
// a no-op that answers the current snapshot, so a member removed in Registre
// is never brought back by an app restarting.
func (c Client) Import(ctx context.Context, spaces []ImportSpace) (Snapshot, error) {
	if spaces == nil {
		spaces = []ImportSpace{}
	}
	body, err := json.Marshal(map[string]any{"spaces": spaces})
	if err != nil {
		return Snapshot{}, fmt.Errorf("directory: encode the import: %w", err)
	}
	return c.call(ctx, http.MethodPost, "/api/directory/import", body)
}

// call sends one authenticated request and decodes the snapshot it answers.
func (c Client) call(ctx context.Context, method, path string, body []byte) (Snapshot, error) {
	request, err := http.NewRequestWithContext(ctx, method,
		strings.TrimSuffix(c.Issuer, "/")+path, bytes.NewReader(body))
	if err != nil {
		return Snapshot{}, fmt.Errorf("directory: build the request: %w", err)
	}
	request.SetBasicAuth(c.ClientID, c.ClientSecret)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient().Do(request)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return Snapshot{}, fmt.Errorf("%w: %s answered %d", ErrUnavailable, path, response.StatusCode)
	}
	var snapshot Snapshot
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode the snapshot: %v", ErrUnavailable, err)
	}
	return snapshot, nil
}

// httpClient is the configured client, or one with a sane timeout.
func (c Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}
