package auth

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/FacileStudio/Agenda/apps/api/internal/bus"
	"github.com/FacileStudio/porte/kit"
	mcpkit "github.com/FacileStudio/tronc/mcp"
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

	ownedEventIDs = `SELECT e.id, e.calendar_id FROM events e JOIN calendars c ON c.id = e.calendar_id
		WHERE c.owner_id = $1 AND c.space_id IS NULL AND e.source = '' ORDER BY e.id`
	ownedCalendarIDs = `SELECT id FROM calendars WHERE owner_id = $1 AND space_id IS NULL ORDER BY id`

	fillAccountColumns = `UPDATE users SET created_at = now(),
		avatar_url = '', avatar_source = '', avatar_upload_path = '', password_hash = '',
		oidc_access_token = '', oidc_refresh_token = '',
		oidc_token_expiry = '0001-01-01 00:00:00+00', profile_synced_at = '0001-01-01 00:00:00+00'
		WHERE id = $1`
)

// NewStore is porte's user store over Agenda's users table, built by
// kit.NewStore so the kit reads the table's columns once and reports what its
// insert leaves NULL through Unwritten. It fails when the table is missing, so
// it runs after the migrations. Agenda has no admin flag, so a new account
// gets no privileges. The picture lives in oidc_picture_url, beside the upload
// path the kit does not know about. An erasure queues the deletion of the
// person's calendars and events on outbox, with ids only; a nil outbox
// publishes nothing.
func NewStore(ctx context.Context, db *sql.DB, outbox *bus.Outbox) (*kit.Store, error) {
	return kit.NewStore(ctx, kit.Store{
		DB:       db,
		Columns:  kit.Columns{Picture: "oidc_picture_url"},
		OnCreate: fillNewAccount,
		EraseData: func(ctx context.Context, tx *sql.Tx, userID int64) error {
			if err := publishErasure(ctx, tx, outbox, userID); err != nil {
				return err
			}
			return eraseData(ctx, tx, userID)
		},
	})
}

// publishErasure queues calendar_event.deleted for every event in the
// person's own calendars and calendar.deleted for each of those calendars,
// before eraseData removes them in the same transaction. The payloads carry
// the ids and nothing personal.
func publishErasure(ctx context.Context, tx *sql.Tx, outbox *bus.Outbox, userID int64) error {
	if outbox == nil {
		return nil
	}
	events, err := idRows(ctx, tx, ownedEventIDs, userID, 2)
	if err != nil {
		return err
	}
	for _, row := range events {
		if err := outbox.ErasedEvent(ctx, tx, row[0], row[1]); err != nil {
			return err
		}
	}
	calendars, err := idRows(ctx, tx, ownedCalendarIDs, userID, 1)
	if err != nil {
		return err
	}
	for _, row := range calendars {
		if err := outbox.ErasedCalendar(ctx, tx, row[0]); err != nil {
			return err
		}
	}
	return nil
}

// idRows reads rows of width integer columns.
func idRows(ctx context.Context, tx *sql.Tx, query string, userID int64, width int) ([][]int64, error) {
	rows, err := tx.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]int64
	for rows.Next() {
		row := make([]int64, width)
		targets := make([]any, width)
		for i := range row {
			targets[i] = &row[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// fillNewAccount writes, in the insert's transaction, the columns the kit does
// not name, with the values GORM wrote before the kit: the sign-up time and
// empty strings and zero times rather than NULL.
func fillNewAccount(ctx context.Context, tx *sql.Tx, userID int64) error {
	_, err := tx.ExecContext(ctx, fillAccountColumns, userID)
	return err
}

// eraseData deletes what an erased person owns, before the kit deletes the
// users row in the same transaction: personal calendars with their events,
// attendees and shares, the person's own shares, attendances, memberships and
// pending MCP actions. Events the person wrote in calendars owned by others
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
	return mcpkit.ErasePending(ctx, tx, strconv.FormatInt(userID, 10))
}
