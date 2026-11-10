// Package envelope is the canonical event envelope every Facile Suite app puts
// on the Antenne bus. The TypeScript half in ts/src/envelope.ts must serialize
// to identical JSON keys; the fixtures under testdata/ prove it from both sides.
package envelope

// App names one application in the suite that may appear on the wire.
type App string

// The apps that may appear on the wire, spelled as they register on the bus.
const (
	AppSablier  App = "Sablier"
	AppOpus     App = "Opus"
	AppArdoise  App = "Ardoise"
	AppPlume    App = "Plume"
	AppGlouton  App = "Glouton"
	AppVision   App = "Vision"
	AppMycelium App = "Mycelium"
	AppSonde    App = "Sonde"
	AppAgenda   App = "Agenda"
	AppJournal  App = "Journal"
	AppCourrier App = "Courrier"
	AppNuage    App = "Nuage"
	AppBulle    App = "Bulle"
	AppAntenne  App = "Antenne"
)

// Action is what happened to an object: created, updated or deleted.
type Action string

// The three actions an event can report.
const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
)

// ObjectType names the kind of object an event carries. Every suite object
// type that crosses the wire has exactly one.
type ObjectType string

// The object types the shared payload structs below describe. An Agenda entry
// is "calendar_event", never "event", which would be ambiguous on an event bus.
const (
	ObjectProject       ObjectType = "project"
	ObjectTask          ObjectType = "task"
	ObjectUser          ObjectType = "user"
	ObjectTimeEntry     ObjectType = "time_entry"
	ObjectInvoice       ObjectType = "invoice"
	ObjectDocument      ObjectType = "document"
	ObjectAgentSession  ObjectType = "agent_session"
	ObjectMonitor       ObjectType = "monitor"
	ObjectCalendar      ObjectType = "calendar"
	ObjectCalendarEvent ObjectType = "calendar_event"
)

// EventVersion pins the envelope version both implementations export.
const EventVersion = 1

// Event wraps a domain object in the canonical envelope every app shares.
type Event[T any] struct {
	Version        int        `json:"version"`
	App            App        `json:"app"`
	Object         ObjectType `json:"object"`
	Action         Action     `json:"action"`
	FacileID       string     `json:"facile_id"`
	Payload        T          `json:"payload"`
	Timestamp      string     `json:"timestamp"`
	IdempotencyKey string     `json:"idempotency_key"`
}

// Project is the task workspace domain shape on the wire.
type Project struct {
	FacileID    string  `json:"facile_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
}

// Task is one unit of tracked work inside a project.
type Task struct {
	FacileID        string `json:"facile_id"`
	ProjectFacileID string `json:"project_facile_id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	ActorEmail      string `json:"actor_email,omitempty"`
}

// User is one human account.
type User struct {
	FacileID string `json:"facile_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

// TimeEntry is a tracked interval of work on a task.
type TimeEntry struct {
	FacileID     string  `json:"facile_id"`
	TaskFacileID string  `json:"task_facile_id"`
	UserFacileID string  `json:"user_facile_id"`
	UserEmail    string  `json:"user_email,omitempty"`
	StartedAt    string  `json:"started_at"`
	StoppedAt    *string `json:"stopped_at"`
}

// AgentSession records one coding-agent run in a project.
type AgentSession struct {
	FacileID  string `json:"facile_id"`
	Project   string `json:"project"`
	Machine   string `json:"machine"`
	Agent     string `json:"agent"`
	Branch    string `json:"branch,omitempty"`
	UserEmail string `json:"user_email"`
	StartedAt string `json:"started_at"`
	StoppedAt string `json:"stopped_at"`
	TokensIn  int64  `json:"tokens_in"`
	TokensOut int64  `json:"tokens_out"`
}

// Invoice is the billing domain shape on the wire.
type Invoice struct {
	FacileID   string  `json:"facile_id"`
	ClientName string  `json:"client_name"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Status     string  `json:"status"`
}

// Document is one signable document's wire shape.
type Document struct {
	FacileID    string `json:"facile_id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	SignerEmail string `json:"signer_email"`
}

// Monitor is one uptime probe's wire shape. Status is "up" or "down";
// Error carries the last failure message and is null when the monitor is up.
type Monitor struct {
	MonitorID string  `json:"monitor_id"`
	FacileID  string  `json:"facile_id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMS int64   `json:"latency_ms"`
	Error     *string `json:"error"`
}

// Calendar is an Agenda calendar's wire shape. It is deliberately minimal,
// because calendar data is personal: no description, color or sharing list.
// OwnerEmail is empty for a calendar owned by a space rather than a person.
type Calendar struct {
	FacileID   string `json:"facile_id"`
	Name       string `json:"name"`
	OwnerEmail string `json:"owner_email,omitempty"`
	ActorEmail string `json:"actor_email,omitempty"`
}

// CalendarEvent is one Agenda calendar entry's wire shape. StartsAt and EndsAt
// are RFC 3339 strings; for an all-day entry they span whole days. It carries
// no description, location or attendee list, because calendar data is personal.
type CalendarEvent struct {
	FacileID         string `json:"facile_id"`
	CalendarFacileID string `json:"calendar_facile_id"`
	Title            string `json:"title"`
	StartsAt         string `json:"starts_at"`
	EndsAt           string `json:"ends_at"`
	AllDay           bool   `json:"all_day"`
	ActorEmail       string `json:"actor_email,omitempty"`
}
