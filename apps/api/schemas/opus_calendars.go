package schemas

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// OpusTasksSlugPattern matches the per-user "Opus tasks" calendars Agenda
// created before Opus tasks moved into the space and personal calendars.
const OpusTasksSlugPattern = "opus-tasks-%"

const personalDefaultQuery = `SELECT id FROM calendars
	WHERE owner_id = ? AND space_id IS NULL AND slug NOT LIKE ?
	ORDER BY is_personal DESC, id LIMIT 1`

// PersonalDefaultCalendar returns the id of the user's default personal
// calendar: their personal calendar, else the oldest calendar they own
// outside any space, never a legacy Opus tasks one. It returns 0 when they
// own none.
func PersonalDefaultCalendar(db *gorm.DB, userID int64) (int64, error) {
	var ids []int64
	if err := db.Raw(personalDefaultQuery, userID, OpusTasksSlugPattern).Scan(&ids).Error; err != nil {
		return 0, fmt.Errorf("look up the personal calendar: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return ids[0], nil
}

// ensurePersonalDefault returns the user's default personal calendar,
// creating the same "Personal" calendar the calendars service creates when
// they own none.
func ensurePersonalDefault(db *gorm.DB, userID int64) (int64, error) {
	id, err := PersonalDefaultCalendar(db, userID)
	if err != nil || id != 0 {
		return id, err
	}
	cal := Calendar{
		OwnerID: userID, Slug: "personal-" + strconv.FormatInt(userID, 10), Name: "Personal",
		Color: "#3b82f6", IsPersonal: true, SyncToken: strconv.FormatInt(time.Now().UnixNano(), 36),
	}
	if err := db.Create(&cal).Error; err != nil {
		return 0, fmt.Errorf("create the personal calendar: %w", err)
	}
	return cal.ID, nil
}

// moveOpusTaskCalendars empties every legacy "Opus tasks" calendar into its
// owner's default personal calendar and deletes it. The space a task belongs
// to was never stored, so the mirror moves a task to its space calendar the
// next time Opus sends it. Idempotent: once the calendars are gone it does
// nothing.
func moveOpusTaskCalendars(db *gorm.DB) error {
	var legacy []Calendar
	if err := db.Where("slug LIKE ?", OpusTasksSlugPattern).Order("id").Find(&legacy).Error; err != nil {
		return fmt.Errorf("list the Opus tasks calendars: %w", err)
	}
	for _, cal := range legacy {
		if err := db.Transaction(func(tx *gorm.DB) error { return moveOpusTaskCalendar(tx, cal) }); err != nil {
			return err
		}
	}
	return nil
}

func moveOpusTaskCalendar(tx *gorm.DB, cal Calendar) error {
	var count int64
	if err := tx.Model(&Event{}).Where("calendar_id = ?", cal.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("count the events of calendar %d: %w", cal.ID, err)
	}
	if count > 0 {
		target, err := ensurePersonalDefault(tx, cal.OwnerID)
		if err != nil {
			return err
		}
		if err := tx.Model(&Event{}).Where("calendar_id = ?", cal.ID).
			Updates(map[string]any{"calendar_id": target, "space_id": nil}).Error; err != nil {
			return fmt.Errorf("move the events of calendar %d: %w", cal.ID, err)
		}
		token := strconv.FormatInt(time.Now().UnixNano(), 36)
		if err := tx.Model(&Calendar{}).Where("id = ?", target).Update("sync_token", token).Error; err != nil {
			return fmt.Errorf("bump calendar %d: %w", target, err)
		}
	}
	if err := tx.Where("calendar_id = ?", cal.ID).Delete(&CalendarMember{}).Error; err != nil {
		return fmt.Errorf("drop the shares of calendar %d: %w", cal.ID, err)
	}
	return tx.Delete(&Calendar{}, cal.ID).Error
}
