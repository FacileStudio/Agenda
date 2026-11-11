package mcp

import (
	"time"

	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// PendingItem is one action waiting on the confirm page.
type PendingItem struct {
	mcpkit.PendingItem
	Event *PendingEvent `json:"event,omitempty"`
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
