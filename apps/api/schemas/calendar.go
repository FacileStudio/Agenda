package schemas

import "time"

type Calendar struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	OwnerID     int64     `gorm:"column:owner_id;index"`
	Slug        string    `gorm:"column:slug;index"`
	Name        string    `gorm:"column:name"`
	Color       string    `gorm:"column:color"`
	Description string    `gorm:"column:description"`
	IsPersonal  bool      `gorm:"column:is_personal;default:false"`
	SyncToken   string    `gorm:"column:sync_token"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Calendar) TableName() string { return "calendars" }

type CalendarMember struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	CalendarID int64     `gorm:"column:calendar_id;uniqueIndex:idx_calendar_member"`
	UserID     int64     `gorm:"column:user_id;uniqueIndex:idx_calendar_member"`
	Role       string    `gorm:"column:role"` // reader, writer, admin
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (CalendarMember) TableName() string { return "calendar_members" }
