// Package bulle is the client a suite app uses to reach Bulle as the
// signed-in user: today the per-app Ideas API, later any AI call. The token
// comes from a TokenSource, which is porte/exchange's Source in practice
// (audience bulle, scopes bulle.read and bulle.write); this package does not
// import porte.
//
// Failures come back typed: a 402 is a [*PlanRequired], any other 4xx a
// [*StatusError] that keeps its status (429 is the daily cap, 409 a batch
// already running), and a 5xx, a timeout or a network error wraps
// [ErrUnreachable]. Paths are built here from escaped segments; nothing a
// browser sends becomes a path as it is.
package bulle

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 20 * time.Second

// ErrUnreachable is wrapped by every failure to get an answer from Bulle: a
// network error, a timeout, a 5xx or an unreadable body.
var ErrUnreachable = stderrors.New("bulle: unreachable")

// PlanRequired is Bulle's 402: the user's plan has no AI access. Message and
// UpgradeURL are for the user; UpgradeURL is empty unless it is a same-origin
// path or an https URL. Key is the entitlement the answer names, when it
// names one.
type PlanRequired struct {
	Message    string
	UpgradeURL string
	Key        string
}

func (e *PlanRequired) Error() string { return "bulle: plan required: " + e.Message }

// StatusError is any other 4xx answer, with Bulle's error code and message.
type StatusError struct {
	Status  int
	Code    string
	Message string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("bulle: answered %d %s: %s", e.Status, e.Code, e.Message)
}

// TokenSource returns the bearer token for the user the context belongs to.
type TokenSource func(ctx context.Context) (string, error)

// Config wires a Client. URL and TokenSource are required.
type Config struct {
	// URL is Bulle's origin, such as https://bulle.facile.studio.
	URL         string
	TokenSource TokenSource
	// Forget, when set, is called after a 401 before the one retry, so the
	// token source mints a new token instead of returning the cached one.
	Forget func(ctx context.Context)
	// HTTPClient calls Bulle. The default has a 20 s timeout.
	HTTPClient *http.Client
}

// Client calls Bulle's API. It is safe for concurrent use.
type Client struct {
	base   *url.URL
	cfg    Config
	client *http.Client
}

// New checks cfg and builds a Client.
func New(cfg Config) (*Client, error) {
	base, err := url.Parse(strings.TrimRight(cfg.URL, "/"))
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil {
		return nil, stderrors.New("bulle: URL must be an absolute http or https origin")
	}
	if cfg.TokenSource == nil {
		return nil, stderrors.New("bulle: a TokenSource is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{base: base, cfg: cfg, client: client}, nil
}

// ContinueURL is where the user continues a run's conversation in Bulle.
func (c *Client) ContinueURL(sessionID string) string {
	if sessionID == "" {
		return c.base.String()
	}
	return c.base.String() + "/s/" + url.PathEscape(sessionID)
}
