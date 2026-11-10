package bulle

import (
	"context"
	"net/http"
	"net/url"
)

func scope(app, space string) url.Values {
	query := url.Values{}
	if app != "" {
		query.Set("app", app)
	}
	if space != "" {
		query.Set("space_id", space)
	}
	return query
}

// Ideas lists app's open ideas in space: a Bulle space id, the id of the
// Registre shared space it mirrors, or "" for the user's personal space.
func (c *Client) Ideas(ctx context.Context, app, space string) (IdeaList, error) {
	var list IdeaList
	err := c.Call(ctx, Request{Method: http.MethodGet, Segments: []string{"ideas"}, Query: scope(app, space)}, &list)
	if list.Ideas == nil {
		list.Ideas = []Idea{}
	}
	return list, err
}

// Generate starts a batch of app's ideas in space. Bulle caps it at three a
// day per user and app, and answers 429 past that and 409 while one runs.
func (c *Client) Generate(ctx context.Context, app, space string) (Batch, error) {
	var batch Batch
	req := Request{
		Method: http.MethodPost, Segments: []string{"ideas", "generate"}, Query: scope(app, ""),
		Body: map[string]string{"space_id": space},
	}
	err := c.Call(ctx, req, &batch)
	return batch, err
}

// Run starts the idea as a background run. With a space, Bulle refuses an
// idea of another space.
func (c *Client) Run(ctx context.Context, ideaID, space string) (RunStart, error) {
	var start RunStart
	req := Request{Method: http.MethodPost, Segments: []string{"ideas", ideaID, "run"}, Query: scope("", space)}
	err := c.Call(ctx, req, &start)
	return start, err
}

// RunStatus reads a run: its state, summary, outputs and pending approvals.
func (c *Client) RunStatus(ctx context.Context, runID string) (Run, error) {
	var run Run
	if err := c.Call(ctx, Request{Method: http.MethodGet, Segments: []string{"runs", runID}}, &run); err != nil {
		return Run{}, err
	}
	run.State = runState(run.Status, len(run.PendingApprovals))
	run.ContinueURL = c.ContinueURL(run.SessionID)
	if run.Outputs == nil {
		run.Outputs = []Output{}
	}
	if run.PendingApprovals == nil {
		run.PendingApprovals = []Approval{}
	}
	for i := range run.Outputs {
		run.Outputs[i].URL = webLink(run.Outputs[i].URL)
	}
	return run, nil
}

// Decide approves or declines a call a run is waiting on.
func (c *Client) Decide(ctx context.Context, runID, callID string, approve bool) error {
	req := Request{
		Method: http.MethodPost, Segments: []string{"runs", runID, "approvals", callID},
		Body: map[string]bool{"approve": approve},
	}
	return c.Call(ctx, req, nil)
}

// Dismiss hides an idea for good.
func (c *Client) Dismiss(ctx context.Context, ideaID string) error {
	req := Request{Method: http.MethodPatch, Segments: []string{"ideas", ideaID}, Body: map[string]string{"status": "dismissed"}}
	return c.Call(ctx, req, nil)
}

func webLink(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return parsed.String()
}
