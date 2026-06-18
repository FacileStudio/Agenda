package schemas

import "time"

type Event struct {
	ID                 int64     `gorm:"column:id;primaryKey"`
	CalendarID         int64     `gorm:"column:calendar_id;index;uniqueIndex:idx_uid_calendar"`
	UID                string    `gorm:"column:uid;uniqueIndex:idx_uid_calendar"`
	ETag               string    `gorm:"column:etag"`
	Sequence           int       `gorm:"column:sequence;default:0"`
	Title              string    `gorm:"column:title"`
	Description        string    `gorm:"column:description;type:text"`
	Location           string    `gorm:"column:location"`
	StartAt            time.Time `gorm:"column:start_at;index"`
	EndAt              time.Time `gorm:"column:end_at;index"`
	IsAllDay           bool      `gorm:"column:is_all_day;default:false"`
	RecurrenceRule     string    `gorm:"column:recurrence_rule"`
	Status             string    `gorm:"column:status;default:'confirmed'"` // confirmed, tentative, cancelled
	ConferenceURL      string    `gorm:"column:conference_url"`
	ConferenceProvider string    `gorm:"column:conference_provider"`
	RawICS             string    `gorm:"column:raw_ics;type:text"`
	CreatedByID        *int64    `gorm:"column:created_by_id;index"`
	CreatedBy          *User     `gorm:"foreignKey:CreatedByID"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Event) TableName() string { return "events" }

type EventAttendee struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	EventID   int64     `gorm:"column:event_id;index"`
	UserID    *int64    `gorm:"column:user_id;index"`
	Email     string    `gorm:"column:email"`
	Response  string    `gorm:"column:response;default:'needs-action'"` // needs-action, accepted, declined, tentative
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (EventAttendee) TableName() string { return "event_attendees" }
