package schemas

import "time"

// Activity is a stretch of time another app recorded for one user: a Sablier
// time session or an Echo call. It is shown only to that user, beside the
// calendar, and is never an event: CalDAV and space calendars do not see it.
// EndAt is nil while the session or call is still running.
type Activity struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	UserID     int64      `gorm:"column:user_id;not null"`
	Source     string     `gorm:"column:source;not null"`
	ExternalID string     `gorm:"column:external_id;not null"`
	Title      string     `gorm:"column:title;not null;default:''"`
	StartAt    time.Time  `gorm:"column:start_at;not null"`
	EndAt      *time.Time `gorm:"column:end_at"`
	URL        string     `gorm:"column:url;not null;default:''"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (Activity) TableName() string { return "activities" }
