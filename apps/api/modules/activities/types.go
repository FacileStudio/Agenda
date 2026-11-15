package activities

import "time"

// ActivityResponse is one session or call. Source is sablier or echo; EndAt
// is null while it is still running. URL opens it in its own app.
type ActivityResponse struct {
	ID      int64      `json:"id"`
	Source  string     `json:"source"`
	Title   string     `json:"title"`
	StartAt time.Time  `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
	URL     string     `json:"url"`
}
