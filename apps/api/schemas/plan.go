package schemas

import "time"

// Plan states.
const (
	PlanProposed  = "proposed"
	PlanApplied   = "applied"
	PlanDismissed = "dismissed"
)

// Plan is a draft schedule a model proposed for a person: blocks of time for
// tasks, which change no calendar until the person applies them. SpaceID is
// set when every block sits in one space's calendar. Warnings, Title and the
// blocks' text are plain text from a model that may have read untrusted
// content, so they are stored and shown as text only. ExpiresAt is when a
// proposed plan lapses and, once it is applied or dismissed, when the row is
// purged.
type Plan struct {
	ID        int64       `gorm:"column:id;primaryKey"`
	UserID    int64       `gorm:"column:user_id;index"`
	SpaceID   *int64      `gorm:"column:space_id"`
	Title     string      `gorm:"column:title"`
	Timezone  string      `gorm:"column:timezone"`
	Warnings  []string    `gorm:"column:warnings;type:jsonb;serializer:json"`
	State     string      `gorm:"column:state"`
	CreatedAt time.Time   `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time   `gorm:"column:updated_at;autoUpdateTime"`
	ExpiresAt time.Time   `gorm:"column:expires_at"`
	Blocks    []PlanBlock `gorm:"foreignKey:PlanID"`
}

// TableName pins the table name.
func (Plan) TableName() string { return "plans" }

// PlanBlock is one proposed block of time for a task. TaskRef is the task's id
// in its own app (an Opus task id) and TaskURL its link. EventID is set once
// the block was applied.
type PlanBlock struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	PlanID     int64     `gorm:"column:plan_id;index"`
	Position   int       `gorm:"column:position"`
	TaskRef    string    `gorm:"column:task_ref"`
	TaskURL    string    `gorm:"column:task_url"`
	Title      string    `gorm:"column:title"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
	CalendarID int64     `gorm:"column:calendar_id"`
	Reason     string    `gorm:"column:reason"`
	EventID    *int64    `gorm:"column:event_id"`
}

// TableName pins the table name.
func (PlanBlock) TableName() string { return "plan_blocks" }
