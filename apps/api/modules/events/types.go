package events

import "time"

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

type UpdateEventRequest struct {
	// CalendarID, when set and different from the event's current calendar,
	// moves the event to that calendar (requires write access to it).
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

type EventCreator struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

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
