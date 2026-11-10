package auth

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/porte/kit"
)

const (
	eraseOwnedAttendees = `DELETE FROM event_attendees WHERE event_id IN (
		SELECT e.id FROM events e JOIN calendars c ON c.id = e.calendar_id
		WHERE c.owner_id = $1 AND c.space_id IS NULL)`
	eraseOwnedEvents = `DELETE FROM events WHERE calendar_id IN (
		SELECT id FROM calendars WHERE owner_id = $1 AND space_id IS NULL)`
	eraseOwnedShares = `DELETE FROM calendar_members WHERE calendar_id IN (
		SELECT id FROM calendars WHERE owner_id = $1 AND space_id IS NULL)`
	eraseOwnedCalendars = `DELETE FROM calendars WHERE owner_id = $1 AND space_id IS NULL`
	eraseOwnShares      = `DELETE FROM calendar_members WHERE user_id = $1`
	eraseOwnAttendance  = `DELETE FROM event_attendees WHERE user_id = $1`
	eraseMemberships    = `DELETE FROM space_members WHERE user_id = $1`
	anonymiseAuthorship = `UPDATE events SET created_by_id = NULL WHERE created_by_id = $1`
)

// NewStore is porte's user store over Agenda's users table. Agenda has no
// admin flag, so a new account gets no privileges. The picture lives in
// oidc_picture_url, beside the upload path the kit does not know about.
func NewStore(db *sql.DB) *kit.Store {
	return &kit.Store{
		DB:        db,
		Columns:   kit.Columns{Picture: "oidc_picture_url"},
		EraseData: eraseData,
	}
}

// eraseData deletes what an erased person owns, before the kit deletes the
// users row in the same transaction: personal calendars with their events,
// attendees and shares, the person's own shares, attendances and memberships.
// Events the person wrote in calendars owned by others
// or by a space stay, with no author.
func eraseData(ctx context.Context, tx *sql.Tx, userID int64) error {
	for _, statement := range []string{
		eraseOwnedAttendees, eraseOwnedEvents, eraseOwnedShares, eraseOwnedCalendars,
		eraseOwnShares, eraseOwnAttendance, eraseMemberships, anonymiseAuthorship,
	} {
		if _, err := tx.ExecContext(ctx, statement, userID); err != nil {
			return err
		}
	}
	return nil
}
