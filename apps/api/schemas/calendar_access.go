package schemas

import "gorm.io/gorm"

// CalendarRoleExpr is the caller's role on the calendar aliased c, bound to
// the named parameter @user, or NULL when they have no access. A space's
// calendar follows space membership alone: owners and admins administer it,
// members write to it. Any other calendar follows ownership and its shares.
const CalendarRoleExpr = `CASE
	WHEN c.space_id IS NOT NULL THEN (
		SELECT CASE sm.role WHEN 'member' THEN 'writer' ELSE 'admin' END
		FROM space_members sm WHERE sm.space_id = c.space_id AND sm.user_id = @user)
	WHEN c.owner_id = @user THEN 'owner'
	ELSE (SELECT cm.role FROM calendar_members cm WHERE cm.calendar_id = c.id AND cm.user_id = @user)
END`

// CalendarAccess is a calendar together with the caller's role on it.
type CalendarAccess struct {
	Calendar
	Role string `gorm:"column:role"`
}

// AccessibleCalendars returns every calendar the user can open, each with
// their role, oldest first.
func AccessibleCalendars(db *gorm.DB, userID int64) ([]CalendarAccess, error) {
	var rows []CalendarAccess
	err := db.Raw(`SELECT * FROM (SELECT c.*, `+CalendarRoleExpr+` AS role FROM calendars c) x
		WHERE x.role IS NOT NULL ORDER BY x.id`, map[string]any{"user": userID}).Scan(&rows).Error
	return rows, err
}

// LoadCalendarAccess returns one calendar and the user's role on it. The
// role is empty when the user has no access; found is false when no
// calendar has that id.
func LoadCalendarAccess(db *gorm.DB, userID, calendarID int64) (access CalendarAccess, found bool, err error) {
	err = db.Raw(`SELECT c.*, coalesce(`+CalendarRoleExpr+`, '') AS role FROM calendars c WHERE c.id = @id`,
		map[string]any{"user": userID, "id": calendarID}).Scan(&access).Error
	return access, access.ID != 0, err
}
