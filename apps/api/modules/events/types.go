package events

import "time"

// CreateEventRequest is the payload for creating an event in a calendar.
// RecurrenceRule is an RFC 5545 RRULE value such as "FREQ=WEEKLY;BYDAY=MO".
// Timezone is the creator's IANA zone; a recurring event is anchored in it so
// its wall-clock time survives DST changes. Status must be confirmed,
// tentative or cancelled and Timezone an IANA name, or the call fails with
// invalid_argument. Color is "#RRGGBB". Attendees and Reminders default to none.
type CreateEventRequest struct {
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Location           string     `json:"location"`
	StartAt            time.Time  `json:"start_at"`
	EndAt              time.Time  `json:"end_at"`
	IsAllDay           bool       `json:"is_all_day"`
	RecurrenceRule     string     `json:"recurrence_rule"`
	Status             string     `json:"status"`
	ConferenceURL      string     `json:"conference_url"`
	ConferenceProvider string     `json:"conference_provider"`
	Timezone           string     `json:"timezone"`
	Color              *string    `json:"color"`
	Attendees          []Attendee `json:"attendees"`
	Reminders          []Reminder `json:"reminders"`
}

// UpdateEventRequest is the payload for editing an event; RecurrenceRule and
// Timezone behave as in CreateEventRequest. An empty Timezone keeps the zone
// the event is already anchored in; an omitted Color, Attendees or Reminders
// keeps the stored value, and an empty one clears it. Editing the schedule of a
// recurring event (start, rule, zone, all-day) discards its occurrence
// exceptions. CalendarID, when set
// and different from the event's current calendar, moves the event to that
// calendar (requires write access to it).
type UpdateEventRequest struct {
	CalendarID         int64      `json:"calendar_id"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Location           string     `json:"location"`
	StartAt            time.Time  `json:"start_at"`
	EndAt              time.Time  `json:"end_at"`
	IsAllDay           bool       `json:"is_all_day"`
	RecurrenceRule     string     `json:"recurrence_rule"`
	Status             string     `json:"status"`
	ConferenceURL      string     `json:"conference_url"`
	ConferenceProvider string     `json:"conference_provider"`
	Timezone           string     `json:"timezone"`
	Color              *string    `json:"color"`
	Attendees          []Attendee `json:"attendees"`
	Reminders          []Reminder `json:"reminders"`
}

// EventCreator describes the user who created an event.
type EventCreator struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// EventResponse describes an event as returned to its calendar's members.
// Timezone is the zone the event is anchored in. Exceptions lists the
// cancelled and overridden occurrences of a recurring event. In an expanded
// list, OccurrenceStart is the original start of the occurrence an entry
// stands for, the value to pass to the occurrence routes.
type EventResponse struct {
	ID                 int64                 `json:"id"`
	CalendarID         int64                 `json:"calendar_id"`
	UID                string                `json:"uid"`
	ETag               string                `json:"etag"`
	Title              string                `json:"title"`
	Description        string                `json:"description"`
	Location           string                `json:"location"`
	StartAt            time.Time             `json:"start_at"`
	EndAt              time.Time             `json:"end_at"`
	IsAllDay           bool                  `json:"is_all_day"`
	RecurrenceRule     string                `json:"recurrence_rule"`
	Status             string                `json:"status"`
	ConferenceURL      string                `json:"conference_url"`
	ConferenceProvider string                `json:"conference_provider"`
	Source             string                `json:"source"`
	ExternalID         string                `json:"external_id"`
	CreatedBy          *EventCreator         `json:"created_by"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
	Timezone           string                `json:"timezone"`
	Color              string                `json:"color"`
	Attendees          []Attendee            `json:"attendees"`
	Reminders          []Reminder            `json:"reminders"`
	Exceptions         []OccurrenceException `json:"exceptions"`
	OccurrenceStart    *time.Time            `json:"occurrence_start,omitempty"`
}

// OkResponse reports success for mutations returning no data.
type OkResponse struct {
	Ok bool `json:"ok"`
}

// OccurrenceRequest overrides one occurrence of a recurring event. Omitted
// fields keep the series value; an empty Description or Location falls back
// to the series value. Status is confirmed or tentative: cancel an occurrence
// with the DELETE route instead. Moving an occurrence keeps its duration
// unless EndAt is given.
type OccurrenceRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Location    *string    `json:"location"`
	StartAt     *time.Time `json:"start_at"`
	EndAt       *time.Time `json:"end_at"`
	Status      *string    `json:"status"`
}
