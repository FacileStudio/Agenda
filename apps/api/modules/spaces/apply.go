package spaces

import (
	"context"
	"log/slog"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/internal/oidcavatar"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

// Apply makes Agenda's spaces match a Registre snapshot, in one transaction.
//
// It is idempotent: a second run over the same snapshot writes nothing. It
// only ever touches spaces the snapshot names and members who sign in through
// Registre, so a local password account added to a space by hand stays put.
// A linked space Registre no longer lists is logged and left alone, because
// Registre has no space deletion and its absence is more likely a fault.
// Every space then owns its calendar, renamed when the space was.
func (d *Directory) Apply(ctx context.Context, snapshot directory.Snapshot) error {
	return d.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		linked := []string{""}
		for _, space := range snapshot.Spaces {
			if err := d.applySpace(tx, space); err != nil {
				return err
			}
			linked = append(linked, space.ID)
		}
		if err := schemas.SyncSpaceCalendars(tx); err != nil {
			return err
		}
		return d.reportUnlisted(tx, linked)
	})
}

// applySpace links one Registre space to its Agenda row, then sets its
// members. keep starts with 0, which no user id is, so a space Registre left
// empty still renders NOT IN (0) and not GORM's NOT IN (NULL), which would
// match nothing and remove nobody.
func (d *Directory) applySpace(tx *gorm.DB, space directory.Space) error {
	spaceID, err := linkSpace(tx, space)
	if err != nil {
		return err
	}
	keep := []int64{0}
	for _, member := range space.Members {
		userID, err := d.resolveMember(tx, member)
		if err != nil {
			return err
		}
		if userID == 0 {
			continue
		}
		if err := tx.Exec(`
			INSERT INTO space_members (space_id, user_id, role, joined_at) VALUES (?, ?, ?, now())
			ON CONFLICT (space_id, user_id) DO UPDATE SET role = EXCLUDED.role
			WHERE space_members.role <> EXCLUDED.role
		`, spaceID, userID, agendaRole(member.Role)).Error; err != nil {
			return err
		}
		keep = append(keep, userID)
	}
	return tx.Exec(`
		DELETE FROM space_members sm
		WHERE sm.space_id = ? AND sm.user_id NOT IN ?
		  AND EXISTS (SELECT 1 FROM porte_identities i WHERE i.user_id = sm.user_id AND i.provider = ?)
	`, spaceID, keep, d.issuer).Error
}

// linkSpace finds the Agenda space for a Registre one: by link first, then
// an unlinked space of the same name ignoring case, and otherwise a new one.
// Registre's name wins; the row is only written when something changed.
func linkSpace(tx *gorm.DB, space directory.Space) (int64, error) {
	var id int64
	err := tx.Raw(`
		SELECT id FROM spaces
		WHERE registre_space_id = ? OR (registre_space_id IS NULL AND lower(name) = lower(?))
		ORDER BY registre_space_id IS NULL, id LIMIT 1
	`, space.ID, space.Name).Scan(&id).Error
	if err != nil {
		return 0, err
	}
	if id == 0 {
		row := schemas.Space{Name: space.Name, RegistreSpaceID: &space.ID}
		err := tx.Create(&row).Error
		return row.ID, err
	}
	return id, tx.Exec(`
		UPDATE spaces SET name = ?, registre_space_id = ?, updated_at = now()
		WHERE id = ? AND (name <> ? OR registre_space_id IS DISTINCT FROM ?)
	`, space.Name, space.ID, id, space.Name, space.ID).Error
}

// resolveMember returns the Agenda user for a Registre member, by identity,
// then by email, and otherwise by provisioning one, so their first sign-in
// lands on the account their spaces already point at. Either of the last two
// writes the identity row porte matches on. A member with no subject or no
// email cannot be matched safely and resolves to 0, which skips them.
func (d *Directory) resolveMember(tx *gorm.DB, member directory.Member) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(member.Email))
	if member.Subject == "" || email == "" {
		return 0, nil
	}
	var userID int64
	err := tx.Raw(`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
		d.issuer, member.Subject).Scan(&userID).Error
	if err != nil || userID != 0 {
		return userID, err
	}
	if err := tx.Raw(`SELECT id FROM users WHERE lower(email) = ?`, email).Scan(&userID).Error; err != nil {
		return 0, err
	}
	if userID == 0 {
		user := schemas.User{Email: email, Name: member.Name, OIDCPictureURL: oidcavatar.PhotoURL(member.Picture)}
		if err := tx.Create(&user).Error; err != nil {
			return 0, err
		}
		userID = user.ID
	}
	return userID, tx.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, d.issuer, member.Subject).Error
}

// reportUnlisted logs linked spaces the snapshot did not carry. linked
// always holds "" for the same NOT IN reason applySpace's keep holds 0.
func (d *Directory) reportUnlisted(tx *gorm.DB, linked []string) error {
	var names []string
	err := tx.Raw(`SELECT name FROM spaces WHERE registre_space_id IS NOT NULL AND registre_space_id NOT IN ?`,
		linked).Scan(&names).Error
	if err != nil || len(names) == 0 {
		return err
	}
	d.logger.Warn("directory: linked spaces missing from the snapshot, left as they are", slog.Any("spaces", names))
	return nil
}
