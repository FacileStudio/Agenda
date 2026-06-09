package schemas

import "time"

type Event struct {
	ID             int64     `gorm:"column:id;primaryKey"`
	CalendarID     int64     `gorm:"column:calendar_id;index"`
	UID            string    `gorm:"column:uid;uniqueIndex"`
	ETag           string    `gorm:"column:etag"`
	Title          string    `gorm:"column:title"`
	Description    string    `gorm:"column:description;type:text"`
	Location       string    `gorm:"column:location"`
	StartAt        time.Time `gorm:"column:start_at;index"`
	EndAt          time.Time `gorm:"column:end_at;index"`
	IsAllDay       bool      `gorm:"column:is_all_day;default:false"`
	RecurrenceRule string    `gorm:"column:recurrence_rule"`
	Status         string    `gorm:"column:status;default:'confirmed'"` // confirmed, tentative, cancelled
	RawICS         string    `gorm:"column:raw_ics;type:text"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Event) TableName() string { return "events" }

type EventAttendee struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	EventID    int64     `gorm:"column:event_id;index"`
	UserID     *int64    `gorm:"column:user_id;index"`
	Email      string    `gorm:"column:email"`
	Response   string    `gorm:"column:response;default:'needs-action'"` // needs-action, accepted, declined, tentative
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (EventAttendee) TableName() string { return "event_attendees" }
