package schemas

import "time"

// Calendar is a calendar an owner shares with members. It carries the sync
// token bumped on every event change.
type Calendar struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	OwnerID     int64     `gorm:"column:owner_id;index"`
	SpaceID     *int64    `gorm:"column:space_id;index"`
	CalDAVPath  string    `gorm:"column:cal_dav_path;index;not null;default:''"`
	Slug        string    `gorm:"column:slug;index"`
	Name        string    `gorm:"column:name"`
	Color       string    `gorm:"column:color"`
	Description string    `gorm:"column:description"`
	EchoURL     string    `gorm:"column:echo_url"`
	IsPersonal  bool      `gorm:"column:is_personal;default:false"`
	SyncToken   string    `gorm:"column:sync_token"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Calendar) TableName() string { return "calendars" }

// CalendarMember is a user's grant on a calendar. Role is one of reader,
// writer or admin.
type CalendarMember struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	CalendarID int64     `gorm:"column:calendar_id;uniqueIndex:idx_calendar_member"`
	UserID     int64     `gorm:"column:user_id;uniqueIndex:idx_calendar_member"`
	Role       string    `gorm:"column:role"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (CalendarMember) TableName() string { return "calendar_members" }
