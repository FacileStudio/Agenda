package bulle

import (
	"context"
	"net/http"
	"strings"
)

// StartRun starts a background run for app, a suite app label such as agenda, from
// the user's own prompt. Bulle caps it at 30 runs an hour per user and app,
// answers 429 past that and 402 without AI access. Follow the run with
// RunStatus and answer its approvals with Decide.
func (c *Client) StartRun(ctx context.Context, app string, req StartRunRequest) (RunStart, error) {
	body := struct {
		App string `json:"app"`
		StartRunRequest
	}{App: app, StartRunRequest: req}
	var start RunStart
	err := c.Call(ctx, Request{Method: http.MethodPost, Segments: []string{"runs"}, Body: body}, &start)
	return start, err
}

// Profile reads the user's assistant name and avatar. It needs no AI access.
func (c *Client) Profile(ctx context.Context) (Profile, error) {
	var profile Profile
	err := c.Call(ctx, Request{Method: http.MethodGet, Segments: []string{"assistant", "profile"}}, &profile)
	profile.AvatarURL = avatarLink(profile.AvatarURL)
	return profile, err
}

// avatarLink keeps an avatar the user's browser can show in an img: an http
// or https link, or an inline image, which is how Bulle stores a generated
// blob. Anything else is dropped.
func avatarLink(raw string) string {
	if strings.HasPrefix(raw, "data:image/") {
		return raw
	}
	return webLink(raw)
}
