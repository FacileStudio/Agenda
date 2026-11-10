package mcptest

import (
	"strings"
	"testing"

	"github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// status is the part of a sensitive tool's result the checks read.
type status struct {
	Status     string `json:"status"`
	ConfirmURL string `json:"confirm_url"`
}

// accept is an accepting answer to a confirmation.
var accept = sdk.InputResponseMap{mcp.ConfirmInput: &sdk.ElicitResult{Action: "accept", Content: map[string]any{mcp.ConfirmInput: true}}}

// checkConfirm runs the confirmation path of one sensitive tool: a client
// that cannot elicit and a pre-MRTR client get a link and nothing runs, a
// decline runs nothing, an accept runs once and its state cannot be
// replayed, and a link id cannot be passed off as a signed state.
func checkConfirm(t *testing.T, cfg Config, c ConfirmCase) {
	t.Run("no elicitation gets a link", func(t *testing.T) { checkLink(t, cfg, c, ClientOptions{}) })
	t.Run("pre-mrtr client gets a link", func(t *testing.T) {
		asked := 0
		checkLink(t, cfg, c, ClientOptions{Elicit: Answering("accept", &asked), Version: "2025-11-25"})
		if asked != 0 {
			t.Errorf("a stateless server cannot ask a pre-MRTR client, yet it was asked %d times", asked)
		}
	})
	t.Run("decline runs nothing", func(t *testing.T) {
		args, runs := c.Setup(t)
		asked := 0
		var out status
		Decode(t, Call(t, Connect(t, cfg.URL, cfg.Token(t), ClientOptions{Elicit: Answering("decline", &asked)}), c.Tool, args), &out)
		if out.Status != mcp.StatusDeclined || asked != 1 || runs() != 0 {
			t.Errorf("want one question, status declined and no run, got %d questions, %+v, %d runs", asked, out, runs())
		}
	})
	t.Run("accept runs once", func(t *testing.T) { checkAccept(t, cfg, c) })
	t.Run("link id is not a state", func(t *testing.T) {
		args, runs := c.Setup(t)
		var out status
		Decode(t, Call(t, Connect(t, cfg.URL, cfg.Token(t), ClientOptions{}), c.Tool, args), &out)
		id := out.ConfirmURL[strings.LastIndex(out.ConfirmURL, "=")+1:]
		session := Connect(t, cfg.URL, cfg.Token(t), ClientOptions{Manual: true})
		forged, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: c.Tool, Arguments: args, RequestState: id, InputResponses: accept})
		if err != nil || !forged.IsError || runs() != 0 {
			t.Errorf("a forged state must fail and run nothing, got %v, %+v, %d runs", err, forged, runs())
		}
	})
}

// checkLink checks a client that cannot be asked gets a pending status and a
// confirm link, and that nothing ran.
func checkLink(t *testing.T, cfg Config, c ConfirmCase, opts ClientOptions) {
	args, runs := c.Setup(t)
	var out status
	Decode(t, Call(t, Connect(t, cfg.URL, cfg.Token(t), opts), c.Tool, args), &out)
	if out.Status != mcp.StatusPending || !strings.Contains(out.ConfirmURL, "action=") || runs() != 0 {
		t.Errorf("want %s with a confirm_url and no run, got %+v and %d runs", mcp.StatusPending, out, runs())
	}
}

// checkAccept answers a question by hand, checks the action ran once, then
// replays the same answer and checks it did not run again.
func checkAccept(t *testing.T, cfg Config, c ConfirmCase) {
	args, runs := c.Setup(t)
	asked := 0
	session := Connect(t, cfg.URL, cfg.Token(t), ClientOptions{Elicit: Answering("accept", &asked), Manual: true})
	question := Call(t, session, c.Tool, args)
	if !question.NeedsInput() || question.RequestState == "" {
		t.Fatalf("a client that can elicit must get input_required, got %+v", question)
	}
	retry := &sdk.CallToolParams{Name: c.Tool, Arguments: args, RequestState: question.RequestState, InputResponses: accept}
	done, err := session.CallTool(t.Context(), retry)
	if err != nil || done.IsError || runs() != 1 {
		t.Fatalf("an accepted confirmation must run once, got %v, %s, %d runs", err, Text(done), runs())
	}
	replay, err := session.CallTool(t.Context(), retry)
	if err != nil || !replay.IsError || runs() != 1 {
		t.Errorf("a replayed confirmation must fail and run nothing, got %v, %+v, %d runs", err, replay, runs())
	}
}
