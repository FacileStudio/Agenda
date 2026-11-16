package bulle

import (
	"encoding/json"
	"time"
)

// Source is what an idea is based on: a label and Bulle's reference, which
// is a fingerprint such as courrier:5/email:812 or a URL.
type Source struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
}

// Idea is one idea card.
type Idea struct {
	ID        string    `json:"id"`
	SpaceID   string    `json:"space_id"`
	App       string    `json:"app"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Why       string    `json:"why"`
	Draft     string    `json:"draft"`
	Sources   []Source  `json:"sources"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IdeaList is an app's open ideas in one space.
type IdeaList struct {
	Ideas       []Idea     `json:"ideas"`
	Generating  bool       `json:"generating"`
	LastBatchAt *time.Time `json:"last_batch_at"`
}

// Batch names the batch a generate started.
type Batch struct {
	BatchID string `json:"batch_id"`
}

// RunStart names the background run doing an idea.
type RunStart struct {
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
	IdeaID    string `json:"idea_id,omitempty"`
}

// Output is one call of a run that changed the user's data. URL is empty
// unless it is an http or https link.
type Output struct {
	CallID  string `json:"call_id"`
	Tool    string `json:"tool"`
	App     string `json:"app"`
	ID      string `json:"id"`
	URL     string `json:"url"`
	Preview string `json:"preview"`
}

// Approval is a call waiting for the user. Answer it with [Client.Decide].
type Approval struct {
	CallID    string          `json:"call_id"`
	Tool      string          `json:"tool"`
	Args      json.RawMessage `json:"args"`
	Grantable bool            `json:"grantable"`
}

// Run states, as the app's page shows them.
const (
	StateRunning       = "running"
	StateNeedsApproval = "needs_approval"
	StateDone          = "done"
	StateFailed        = "failed"
)

// Run is a background run. Status is Bulle's own; State is one of the State
// constants, derived from it; ContinueURL opens the run's chat in Bulle.
type Run struct {
	ID               string     `json:"id"`
	SessionID        string     `json:"session_id"`
	Status           string     `json:"status"`
	State            string     `json:"state"`
	Summary          string     `json:"summary"`
	Outputs          []Output   `json:"outputs"`
	PendingApprovals []Approval `json:"pending_approvals"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at"`
	ContinueURL      string     `json:"continue_url"`
}

// runState maps Bulle's run status: waiting, or running with a call held, is
// an approval; finished is done; error, stopped and interrupted are failures.
func runState(status string, pending int) string {
	switch {
	case status == "waiting" || (status == "running" && pending > 0):
		return StateNeedsApproval
	case status == "finished":
		return StateDone
	case status == "error" || status == "stopped" || status == "interrupted":
		return StateFailed
	}
	return StateRunning
}

// PageContext is the app page a run starts from, such as Agenda's calendar
// view. App must be the app that starts the run; Bulle checks the rest and
// answers 400 for a page without an id or an https URL.
type PageContext struct {
	App   string `json:"app"`
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// StartRunRequest starts a run for the signed-in user. Prompt is what the
// user typed or the app's fixed template, never text from a third party:
// Bulle treats it as the owner's own message. Bulle caps it at 2000
// characters and Title at 120.
type StartRunRequest struct {
	SpaceID string       `json:"space_id,omitempty"`
	Title   string       `json:"title,omitempty"`
	Prompt  string       `json:"prompt"`
	Page    *PageContext `json:"page,omitempty"`
}

// Profile is the user's assistant as an app's Bulle button shows it. Both
// fields are empty until the user sets them.
type Profile struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
