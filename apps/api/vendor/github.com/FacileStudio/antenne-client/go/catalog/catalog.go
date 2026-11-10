// Package catalog lists every channel the suite emits or listens to on the
// Antenne bus, so a typo in a channel name fails to compile instead of
// silently subscribing to nothing. The TypeScript half is ts/src/catalog.ts.
package catalog

// Channel is the name of one Antenne bus channel, as in "task.created".
type Channel string

const (
	// TaskCreated is emitted by Opus and Sablier with an envelope.Event[envelope.Task].
	TaskCreated Channel = "task.created"
	// TaskUpdated is emitted by Opus and Sablier with an envelope.Event[envelope.Task].
	TaskUpdated Channel = "task.updated"
	// TaskDeleted is emitted by Opus and Sablier with an envelope.Event[envelope.Task].
	TaskDeleted Channel = "task.deleted"
	// TaskStatusChanged is emitted by Opus with an envelope.Event[envelope.Task] on a workflow transition.
	TaskStatusChanged Channel = "task.status_changed"
	// TaskActorChanged is emitted by Opus with an envelope.Event[envelope.Task] on an assignee change.
	TaskActorChanged Channel = "task.actor_changed"

	// ProjectCreated is emitted by Opus and Sablier with an envelope.Event[envelope.Project].
	ProjectCreated Channel = "project.created"
	// ProjectUpdated is emitted by Opus and Sablier with an envelope.Event[envelope.Project].
	ProjectUpdated Channel = "project.updated"
	// ProjectDeleted is emitted by Opus and Sablier with an envelope.Event[envelope.Project].
	ProjectDeleted Channel = "project.deleted"

	// TimeEntryCreated is emitted by Sablier with an envelope.Event[envelope.TimeEntry].
	TimeEntryCreated Channel = "time_entry.created"
	// TimeEntryUpdated is emitted by Sablier with an envelope.Event[envelope.TimeEntry].
	TimeEntryUpdated Channel = "time_entry.updated"
	// TimeEntryDeleted is emitted by Sablier with an envelope.Event[envelope.TimeEntry].
	TimeEntryDeleted Channel = "time_entry.deleted"

	// CommentCreated is emitted by Opus with an envelope.Event of Opus's own comment payload (object "comment").
	CommentCreated Channel = "comment.created"
	// WorkspaceCreated is emitted by Opus with an envelope.Event of Opus's own workspace payload (object "workspace").
	WorkspaceCreated Channel = "workspace.created"

	// AgentSessionCreated is emitted by Mycelium with an envelope.Event[envelope.AgentSession].
	AgentSessionCreated Channel = "agent_session.created"
	// AgentSessionUpdated is listened to by Sablier with an envelope.Event[envelope.AgentSession]; no app emits it today.
	AgentSessionUpdated Channel = "agent_session.updated"
	// UsageAlertCreated is emitted by Mycelium with an envelope.Event of Mycelium's own usage alert payload.
	UsageAlertCreated Channel = "usage_alert.created"

	// AlertFired is emitted by Journal with an envelope.Event of a map carrying the alert rule, count and fired_at.
	AlertFired Channel = "alert.fired"

	// MonitorDown is emitted by Sonde with an envelope.Event[envelope.Monitor] when a monitor goes down.
	MonitorDown Channel = "monitor.down"
	// MonitorUp is emitted by Sonde with an envelope.Event[envelope.Monitor] when a monitor recovers.
	MonitorUp Channel = "monitor.up"

	// GitPush is emitted by Antenne's git webhook provider with Antenne's normalized forge payload.
	GitPush Channel = "git.push"
	// GitPullRequest is emitted by Antenne's git webhook provider with Antenne's normalized forge payload.
	GitPullRequest Channel = "git.pull_request"
	// GitIssue is emitted by Antenne's git webhook provider with Antenne's normalized forge payload.
	GitIssue Channel = "git.issue"
	// GitEvent is emitted by Antenne's git webhook provider for any other forge event, with the same payload.
	GitEvent Channel = "git.event"

	// FiletCheckFailed is emitted by Antenne's filet webhook provider with Antenne's normalized filet run payload.
	FiletCheckFailed Channel = "filet.check.failed"
	// FiletCheckPassed is emitted by Antenne's filet webhook provider with Antenne's normalized filet run payload.
	FiletCheckPassed Channel = "filet.check.passed"
)

// All returns every channel in the catalog, in declaration order.
func All() []Channel {
	return []Channel{
		TaskCreated, TaskUpdated, TaskDeleted, TaskStatusChanged, TaskActorChanged,
		ProjectCreated, ProjectUpdated, ProjectDeleted,
		TimeEntryCreated, TimeEntryUpdated, TimeEntryDeleted,
		CommentCreated, WorkspaceCreated,
		AgentSessionCreated, AgentSessionUpdated, UsageAlertCreated,
		AlertFired,
		MonitorDown, MonitorUp,
		GitPush, GitPullRequest, GitIssue, GitEvent,
		FiletCheckFailed, FiletCheckPassed,
	}
}
