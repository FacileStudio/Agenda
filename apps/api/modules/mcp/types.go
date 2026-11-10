package mcp

// calendarsInput is the input of agenda_list_calendars.
type calendarsInput struct{}

// calendarSummary is one calendar the user can open.
type calendarSummary struct {
	CalendarID  int64  `json:"calendar_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Role        string `json:"role"`
	Personal    bool   `json:"personal"`
	InSpace     bool   `json:"in_space"`
	Description string `json:"description,omitempty"`
}

// calendarsOutput is the output of agenda_list_calendars.
type calendarsOutput struct {
	Calendars []calendarSummary `json:"calendars"`
}

// listInput is the input of agenda_list_events.
type listInput struct {
	From           string  `json:"from" jsonschema:"start of the range: RFC 3339 such as 2026-11-10T09:00:00+01:00, or a date such as 2026-11-10"`
	To             string  `json:"to" jsonschema:"end of the range, same formats; a date means the end of that day. At most 366 days after from"`
	CalendarIDs    []int64 `json:"calendar_ids,omitempty" jsonschema:"calendars to read, from agenda_list_calendars. Omit for every calendar"`
	Query          string  `json:"query,omitempty" jsonschema:"words that must all appear in the title, description or location. Omit to list everything in the range"`
	Timezone       string  `json:"timezone,omitempty" jsonschema:"IANA zone for dates and recurring events, such as Europe/Paris. Default UTC"`
	Limit          int     `json:"limit,omitempty" jsonschema:"events per page, 1 to 100. Default 50"`
	Cursor         string  `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	ResponseFormat string  `json:"response_format,omitempty" jsonschema:"concise (default) or detailed, which adds description, status, attendees and recurrence"`
}

// eventSummary is one event, or one occurrence of a recurring event.
type eventSummary struct {
	EventID         int64  `json:"event_id"`
	CalendarID      int64  `json:"calendar_id"`
	Title           string `json:"title"`
	Start           string `json:"start"`
	End             string `json:"end"`
	AllDay          bool   `json:"all_day"`
	Location        string `json:"location,omitempty"`
	OccurrenceStart string `json:"occurrence_start,omitempty"`
	eventDetail
}

// eventDetail is what the detailed format adds to a summary.
type eventDetail struct {
	Description    string   `json:"description,omitempty"`
	Status         string   `json:"status,omitempty"`
	Timezone       string   `json:"timezone,omitempty"`
	RecurrenceRule string   `json:"recurrence_rule,omitempty"`
	ConferenceURL  string   `json:"conference_url,omitempty"`
	Attendees      []string `json:"attendees,omitempty"`
}

// listOutput is the output of agenda_list_events.
type listOutput struct {
	Events     []eventSummary `json:"events"`
	Total      int            `json:"total"`
	NextCursor string         `json:"next_cursor,omitempty"`
	Note       string         `json:"note,omitempty"`
}

// eventRef names one event.
type eventRef struct {
	EventID int64 `json:"event_id" jsonschema:"the event, from agenda_list_events"`
}

// attendee is one invitee of an event.
type attendee struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Response string `json:"response,omitempty"`
}

// eventOutput is the output of agenda_read_event, and of the writes.
type eventOutput struct {
	EventID        int64      `json:"event_id"`
	CalendarID     int64      `json:"calendar_id"`
	Title          string     `json:"title"`
	Start          string     `json:"start"`
	End            string     `json:"end"`
	AllDay         bool       `json:"all_day"`
	Timezone       string     `json:"timezone"`
	Status         string     `json:"status"`
	Location       string     `json:"location,omitempty"`
	Description    string     `json:"description,omitempty"`
	Truncated      bool       `json:"description_truncated,omitempty"`
	RecurrenceRule string     `json:"recurrence_rule,omitempty"`
	ConferenceURL  string     `json:"conference_url,omitempty"`
	Attendees      []attendee `json:"attendees,omitempty"`
	Reminders      []int      `json:"reminder_minutes,omitempty"`
	Exceptions     int        `json:"changed_occurrences,omitempty"`
	ReadOnly       bool       `json:"read_only,omitempty"`
}

// createInput is the input of agenda_create_event.
type createInput struct {
	CalendarID     int64    `json:"calendar_id,omitempty" jsonschema:"the calendar, from agenda_list_calendars. Omit for the user's personal calendar"`
	Title          string   `json:"title" jsonschema:"the event title"`
	Start          string   `json:"start" jsonschema:"RFC 3339 such as 2026-11-12T14:00:00+01:00, or a date for an all-day event"`
	End            string   `json:"end" jsonschema:"RFC 3339, or the last date of an all-day event. Omit for one hour, or one day when all_day"`
	AllDay         bool     `json:"all_day,omitempty" jsonschema:"an all-day event: start and end are dates in timezone"`
	Timezone       string   `json:"timezone,omitempty" jsonschema:"the user's IANA zone, such as Europe/Paris. Anchors recurring and all-day events. Default UTC"`
	Location       string   `json:"location,omitempty" jsonschema:"where it happens"`
	Description    string   `json:"description,omitempty" jsonschema:"notes, as plain text"`
	RecurrenceRule string   `json:"recurrence_rule,omitempty" jsonschema:"an RFC 5545 RRULE such as FREQ=WEEKLY;BYDAY=MO. Omit for a one-off event"`
	Attendees      []string `json:"attendees,omitempty" jsonschema:"invitees' email addresses. Agenda records them; it sends no mail"`
	ConferenceURL  string   `json:"conference_url,omitempty" jsonschema:"a video call link"`
	Status         string   `json:"status,omitempty" jsonschema:"confirmed (default) or tentative"`
}

// updateInput is the input of agenda_update_event. Omitted fields keep
// their value.
type updateInput struct {
	EventID        int64     `json:"event_id" jsonschema:"the event, from agenda_list_events"`
	CalendarID     *int64    `json:"calendar_id,omitempty" jsonschema:"move the event to this calendar"`
	Title          *string   `json:"title,omitempty" jsonschema:"a new title"`
	Start          *string   `json:"start,omitempty" jsonschema:"a new start; without end the event keeps its length"`
	End            *string   `json:"end,omitempty" jsonschema:"a new end"`
	AllDay         *bool     `json:"all_day,omitempty" jsonschema:"make it all-day or timed"`
	Timezone       *string   `json:"timezone,omitempty" jsonschema:"the user's IANA zone"`
	Location       *string   `json:"location,omitempty" jsonschema:"a new location; empty clears it"`
	Description    *string   `json:"description,omitempty" jsonschema:"new notes; empty clears them"`
	RecurrenceRule *string   `json:"recurrence_rule,omitempty" jsonschema:"a new RRULE; empty makes it a one-off event"`
	Attendees      *[]string `json:"attendees,omitempty" jsonschema:"the full new list of invitees' emails; empty clears it"`
	ConferenceURL  *string   `json:"conference_url,omitempty" jsonschema:"a new video call link; empty clears it"`
	Status         *string   `json:"status,omitempty" jsonschema:"confirmed or tentative"`
}

// writeOutput reports a created or updated event.
type writeOutput struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Event   eventOutput `json:"event"`
}

// actionOutput reports a delete, or why it did not happen yet.
type actionOutput struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	EventID    int64  `json:"event_id,omitempty"`
	ConfirmURL string `json:"confirm_url,omitempty"`
}

// slotsInput is the input of agenda_find_free_slots.
type slotsInput struct {
	From            string  `json:"from" jsonschema:"start of the search: RFC 3339 or a date"`
	To              string  `json:"to" jsonschema:"end of the search; a date means the end of that day. At most 31 days after from"`
	DurationMinutes int     `json:"duration_minutes" jsonschema:"the meeting length, 5 to 720 minutes"`
	CalendarIDs     []int64 `json:"calendar_ids,omitempty" jsonschema:"calendars whose events count as busy. Omit for every calendar"`
	Timezone        string  `json:"timezone,omitempty" jsonschema:"the user's IANA zone for working hours, such as Europe/Paris. Default UTC"`
	DayStart        string  `json:"day_start,omitempty" jsonschema:"start of the working day, HH:MM. Default 09:00"`
	DayEnd          string  `json:"day_end,omitempty" jsonschema:"end of the working day, HH:MM. Default 18:00"`
	IncludeWeekends bool    `json:"include_weekends,omitempty" jsonschema:"also search Saturdays and Sundays"`
}

// freeSlot is one free window.
type freeSlot struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Minutes int    `json:"minutes"`
}

// slotsOutput is the output of agenda_find_free_slots.
type slotsOutput struct {
	Slots []freeSlot `json:"slots"`
	Note  string     `json:"note,omitempty"`
}
