package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Session{},
		&AppSetting{},
		&ApiToken{},
		&Space{},
		&SpaceMember{},
		&Calendar{},
		&CalendarMember{},
		&Event{},
		&EventAttendee{},
	)
}
