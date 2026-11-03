package schemas

import "gorm.io/gorm"

// SpaceCalendarColor is the color a space's calendar starts with.
const SpaceCalendarColor = "#8b5cf6"

// SyncSpaceCalendars makes every space own exactly one calendar, named after
// it, and every space_id on a calendar point at a real space.
//
// A calendar filed under a missing space, or a second calendar filed under
// the same space, is handed back to its owner as a personal calendar: its
// events and shares stay where they are, and nothing is deleted. The space
// keeps its oldest calendar, which then takes the space's name, and events
// copy their calendar's space. The partial unique index pins the invariant
// for every later write. It is idempotent.
func SyncSpaceCalendars(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range []string{detachOrphans, detachExtras, createMissing, followNames, followEvents, uniquePerSpace} {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

const (
	detachOrphans = `UPDATE calendars SET space_id = NULL
	 WHERE space_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM spaces s WHERE s.id = calendars.space_id)`
	detachExtras = `UPDATE calendars SET space_id = NULL
	 WHERE space_id IS NOT NULL
	   AND id <> (SELECT min(c2.id) FROM calendars c2 WHERE c2.space_id = calendars.space_id)`
	createMissing = `INSERT INTO calendars (owner_id, space_id, cal_dav_path, slug, name, color, description, echo_url,
	                        is_personal, sync_token, created_at, updated_at)
	 SELECT 0, s.id, '', 'space-' || s.id, s.name, '` + SpaceCalendarColor + `', '', '', false,
	        'space-' || s.id || '-' || floor(extract(epoch FROM clock_timestamp()) * 1000000)::bigint, now(), now()
	 FROM spaces s
	 WHERE NOT EXISTS (SELECT 1 FROM calendars c WHERE c.space_id = s.id)`
	followNames = `UPDATE calendars c SET name = s.name, updated_at = now()
	 FROM spaces s WHERE c.space_id = s.id AND c.name <> s.name`
	followEvents = `UPDATE events e SET space_id = c.space_id
	 FROM calendars c WHERE e.calendar_id = c.id AND e.space_id IS DISTINCT FROM c.space_id`
	uniquePerSpace = `CREATE UNIQUE INDEX IF NOT EXISTS idx_calendars_one_per_space ON calendars (space_id)
	 WHERE space_id IS NOT NULL`
)
