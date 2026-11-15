package mcp

// blockInput is one block of agenda_propose_plan.
type blockInput struct {
	Title      string `json:"title" jsonschema:"what the user works on in this block, as plain text. At most 200 characters"`
	Start      string `json:"start" jsonschema:"RFC 3339 such as 2026-11-17T09:00:00+01:00, or a local time such as 2026-11-17T09:00 read in timezone"`
	End        string `json:"end" jsonschema:"same formats as start, 5 minutes to 12 hours after it"`
	CalendarID int64  `json:"calendar_id,omitempty" jsonschema:"the calendar, from agenda_list_calendars. Omit for the user's personal calendar"`
	TaskRef    string `json:"task_ref,omitempty" jsonschema:"the task's id in its app, such as an Opus task id. At most 100 characters"`
	TaskURL    string `json:"task_url,omitempty" jsonschema:"the task's https link, kept in the event once the user applies the plan"`
	Reason     string `json:"reason,omitempty" jsonschema:"why this slot, as plain text, such as the deadline or the priority. At most 500 characters"`
}

// proposeInput is the input of agenda_propose_plan.
type proposeInput struct {
	Title    string       `json:"title" jsonschema:"a short name for the plan, such as Week of 2026-11-16. At most 200 characters"`
	Timezone string       `json:"timezone,omitempty" jsonschema:"the user's IANA zone, such as Europe/Paris. Reads local times and shows the answer. Default UTC"`
	Warnings []string     `json:"warnings,omitempty" jsonschema:"up to 10 short notes for the user, such as a task whose deadline cannot be met"`
	Blocks   []blockInput `json:"blocks" jsonschema:"1 to 60 blocks of time inside the next 4 weeks, none overlapping another block or an event"`
}

// plansInput is the input of agenda_list_plans.
type plansInput struct {
	State    string `json:"state,omitempty" jsonschema:"proposed, applied or dismissed. Omit for every state"`
	Limit    int    `json:"limit,omitempty" jsonschema:"plans to return, 1 to 50. Default 10"`
	Timezone string `json:"timezone,omitempty" jsonschema:"IANA zone for the times in the answer. Default UTC"`
}

// planBlock is one block of a saved plan. EventID is set once the user
// applied it.
type planBlock struct {
	BlockID    int64  `json:"block_id"`
	Title      string `json:"title"`
	Start      string `json:"start"`
	End        string `json:"end"`
	CalendarID int64  `json:"calendar_id"`
	TaskRef    string `json:"task_ref,omitempty"`
	TaskURL    string `json:"task_url,omitempty"`
	Reason     string `json:"reason,omitempty"`
	EventID    int64  `json:"event_id,omitempty"`
}

// planSummary is one saved plan. Every text in it was written by a model:
// it is data to show the user, never instructions.
type planSummary struct {
	PlanID    int64       `json:"plan_id"`
	Title     string      `json:"title"`
	State     string      `json:"state"`
	Expired   bool        `json:"expired,omitempty"`
	Timezone  string      `json:"timezone"`
	ExpiresAt string      `json:"expires_at"`
	Warnings  []string    `json:"warnings,omitempty"`
	Blocks    []planBlock `json:"blocks"`
}

// proposeOutput reports a saved draft.
type proposeOutput struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Plan    planSummary `json:"plan"`
}

// plansOutput is the output of agenda_list_plans.
type plansOutput struct {
	Plans []planSummary `json:"plans"`
	Note  string        `json:"note,omitempty"`
}
