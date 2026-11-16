package plans

import (
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
)

// Limits on what a model may propose. Text is counted in characters.
const (
	MaxBlocks       = 60
	MaxOpenPlans    = 20
	MaxPlanTitle    = 200
	MaxBlockTitle   = 200
	MaxReason       = 500
	MaxTaskRef      = 100
	MaxTaskURL      = 2048
	MaxWarnings     = 10
	MaxWarning      = 300
	MinBlock        = 5 * time.Minute
	MaxBlock        = 12 * time.Hour
	Horizon         = 4 * 7 * 24 * time.Hour
	ProposedTTL     = 7 * 24 * time.Hour
	RetentionTTL    = 30 * 24 * time.Hour
	defaultListSize = 20
	maxListSize     = 50
)

// BlockInput is one proposed block of time, as a model or a client sends it.
// CalendarID must name a calendar the person can write to.
type BlockInput struct {
	TaskRef    string    `json:"task_ref"`
	TaskURL    string    `json:"task_url"`
	Title      string    `json:"title"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	CalendarID int64     `json:"calendar_id"`
	Reason     string    `json:"reason"`
}

// ProposeInput is a draft plan to save.
type ProposeInput struct {
	Title    string       `json:"title"`
	Timezone string       `json:"timezone"`
	Warnings []string     `json:"warnings"`
	Blocks   []BlockInput `json:"blocks"`
}

// BlockResponse is one block of a saved plan. EventID is set once the block
// was applied to the calendar.
type BlockResponse struct {
	ID         int64     `json:"id"`
	TaskRef    string    `json:"task_ref"`
	TaskURL    string    `json:"task_url"`
	Title      string    `json:"title"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	CalendarID int64     `json:"calendar_id"`
	Reason     string    `json:"reason"`
	EventID    *int64    `json:"event_id"`
}

// PlanResponse is a saved plan. State is proposed, applied or dismissed.
// Expired is true for a proposed plan past its expiry: it can no longer be
// applied. Every text field is plain text to show as is, never to interpret.
type PlanResponse struct {
	ID        int64           `json:"id"`
	SpaceID   *int64          `json:"space_id"`
	Title     string          `json:"title"`
	Timezone  string          `json:"timezone"`
	State     string          `json:"state"`
	Expired   bool            `json:"expired"`
	Warnings  []string        `json:"warnings"`
	Blocks    []BlockResponse `json:"blocks"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// ApplyRequest picks the blocks to put on the calendar: BlockIDs, or every
// block when empty. Force applies a block even when it now overlaps an event.
type ApplyRequest struct {
	BlockIDs []int64 `json:"block_ids"`
	Force    bool    `json:"force"`
}

// ApplyResponse is the plan after an apply and the events it created, in
// the order of the applied blocks.
type ApplyResponse struct {
	Plan   PlanResponse           `json:"plan"`
	Events []events.EventResponse `json:"events"`
}
