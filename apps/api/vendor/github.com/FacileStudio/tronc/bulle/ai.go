package bulle

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"strings"
)

// ActingForHeader names, on a DecideAI call, the Bulle user the service
// acts for.
const ActingForHeader = "X-Bulle-Acting-For"

// ErrNoActingFor is returned before any call when a DecideRequest names no
// user.
var ErrNoActingFor = stderrors.New("bulle: DecideAI needs the email of the user it acts for")

// CompleteRequest is one completion for the signed-in user. Input is
// untrusted data the prompt works on, which Bulle fences off from the
// instructions; Schema, a JSON Schema, makes the answer a validated JSON
// value instead of text. SpaceID is the Registre space the request is for,
// whose plan may grant AI when the user's own does not; empty means the
// personal scope. Bulle caps System at 4 KiB, Prompt at 8 KiB, Input at
// 64 KiB and MaxTokens at 4096 (default 1024).
type CompleteRequest struct {
	System    string          `json:"system,omitempty"`
	Prompt    string          `json:"prompt"`
	Input     string          `json:"input,omitempty"`
	Schema    json.RawMessage `json:"schema,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	SpaceID   string          `json:"space_id,omitempty"`
}

// Completion is Bulle's answer: Text, or JSON when the request had a
// schema, and the model that wrote it.
type Completion struct {
	Text  string          `json:"text,omitempty"`
	JSON  json.RawMessage `json:"json,omitempty"`
	Model string          `json:"model"`
}

// Complete asks Bulle's POST /api/ai/complete for one completion, as the
// user the TokenSource speaks for. A user without AI access gets
// *PlanRequired; past the daily cap a 429 *StatusError.
func (c *Client) Complete(ctx context.Context, req CompleteRequest) (Completion, error) {
	var out Completion
	err := c.Call(ctx, Request{Method: http.MethodPost, Segments: []string{"ai", "complete"}, Body: req}, &out)
	return out, err
}

// Question is one typed question of a Decide call, in Bulle's decision
// contract: Type "choice" picks one of Options following Instructions.
type Question struct {
	Type         string   `json:"type"`
	Instructions any      `json:"instructions"`
	Criteria     any      `json:"criteria,omitempty"`
	Options      []string `json:"options,omitempty"`
}

// DecideRequest asks Questions about State for the Bulle user whose email
// is ActingFor.
type DecideRequest struct {
	ActingFor string              `json:"-"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// Answer is one choice question's result.
type Answer struct {
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

// Usage is what one DecideAI call billed.
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost,omitempty"`
}

// Decision is every answer, keyed by its question's name, the model that
// served them and the usage.
type Decision struct {
	Answers map[string]Answer `json:"answers"`
	Model   string            `json:"model"`
	Usage   Usage             `json:"usage"`
}

// DecideAI asks Bulle's POST /api/ai/decide. Unlike the other calls it runs
// as the app's service: the TokenSource returns the service key Bulle knows
// the app by, and ActingFor names the user, whose plan Bulle checks.
func (c *Client) DecideAI(ctx context.Context, req DecideRequest) (Decision, error) {
	actingFor := strings.TrimSpace(req.ActingFor)
	if actingFor == "" {
		return Decision{}, ErrNoActingFor
	}
	var out Decision
	call := Request{
		Method: http.MethodPost, Segments: []string{"ai", "decide"}, Body: req,
		Header: http.Header{ActingForHeader: {actingFor}},
	}
	err := c.Call(ctx, call, &out)
	return out, err
}
