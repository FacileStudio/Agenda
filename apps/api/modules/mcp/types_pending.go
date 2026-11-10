package mcp

import "time"

// PendingItem is one action waiting on the confirm page.
type PendingItem struct {
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	ClientID  string        `json:"client_id"`
	Summary   string        `json:"summary"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	State     string        `json:"state"`
	Event     *PendingEvent `json:"event,omitempty"`
}

// PendingEvent is the event an action would delete, as it is now.
type PendingEvent struct {
	ID         int64     `json:"id"`
	CalendarID int64     `json:"calendar_id"`
	Title      string    `json:"title"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	IsAllDay   bool      `json:"is_all_day"`
	Location   string    `json:"location"`
	Recurring  bool      `json:"recurring"`
}

// PendingList is the body of GET /api/mcp/pending.
type PendingList struct {
	Actions []PendingItem `json:"actions"`
}

// PendingResult is the body answered when an action is confirmed or dismissed.
type PendingResult struct {
	Status string `json:"status"`
}

// EmptyRequest documents a route that takes no body.
type EmptyRequest struct{}
