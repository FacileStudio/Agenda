package events

import "time"

// CreateEventRequest is the payload for creating an event in a calendar.
type CreateEventRequest struct {
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Location           string    `json:"location"`
	StartAt            time.Time `json:"start_at"`
	EndAt              time.Time `json:"end_at"`
	IsAllDay           bool      `json:"is_all_day"`
	RecurrenceRule     string    `json:"recurrence_rule"`
	Status             string    `json:"status"`
	ConferenceURL      string    `json:"conference_url"`
	ConferenceProvider string    `json:"conference_provider"`
}

// UpdateEventRequest is the payload for editing an event. CalendarID, when set
// and different from the event's current calendar, moves the event to that
// calendar (requires write access to it).
type UpdateEventRequest struct {
	CalendarID         int64     `json:"calendar_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Location           string    `json:"location"`
	StartAt            time.Time `json:"start_at"`
	EndAt              time.Time `json:"end_at"`
	IsAllDay           bool      `json:"is_all_day"`
	RecurrenceRule     string    `json:"recurrence_rule"`
	Status             string    `json:"status"`
	ConferenceURL      string    `json:"conference_url"`
	ConferenceProvider string    `json:"conference_provider"`
}

// EventCreator describes the user who created an event.
type EventCreator struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// EventResponse describes an event as returned to its calendar's members.
type EventResponse struct {
	ID                 int64         `json:"id"`
	CalendarID         int64         `json:"calendar_id"`
	UID                string        `json:"uid"`
	ETag               string        `json:"etag"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	Location           string        `json:"location"`
	StartAt            time.Time     `json:"start_at"`
	EndAt              time.Time     `json:"end_at"`
	IsAllDay           bool          `json:"is_all_day"`
	RecurrenceRule     string        `json:"recurrence_rule"`
	Status             string        `json:"status"`
	ConferenceURL      string        `json:"conference_url"`
	ConferenceProvider string        `json:"conference_provider"`
	CreatedBy          *EventCreator `json:"created_by"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}
