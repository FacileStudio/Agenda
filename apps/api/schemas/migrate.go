package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
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
	); err != nil {
		return err
	}
	return backfillAvatarSources(db)
}

// backfillAvatarSources moves the two values the avatar is now derived from onto the
// columns that own them.
//
// The uploads move by FILENAME, not by avatar_source. That column was added after the
// upload feature, so the oldest uploaded avatars have it empty, and keying on
// avatar_source = 'upload' would quietly drop their picture — it did exactly that on
// Sablier's production database, where two rows of four were pre-column uploads.
// persistAvatarFile has always named uploads "user-<id>-<nanos>" and the old OIDC download
// named its copies "oidc-<id>-<nanos>", so anything that is not an oidc- copy is somebody's
// upload and is kept. The oidc- copies have nothing to preserve: oidc_picture_url now holds
// the URL that replaces them. Their files are left on the volume rather than deleted here —
// a migration that removes files has to be right the first time.
//
// The second statement is the other half of the same story: the old sync stored
// profile.Picture verbatim, so users with no photo in Authentik carry a
// "data:image/svg+xml;base64,…" placeholder in oidc_picture_url. Under the new rule that
// column means "the IdP has a photo", so a stale data: URI would suppress the upload
// fallback and render Authentik's initials instead of ours.
//
// avatar_url and avatar_source stay in the table, unread, until a later release drops them.
// Expanding first means a rollback is redeploying the old binary, not restoring a backup.
func backfillAvatarSources(db *gorm.DB) error {
	if db.Migrator().HasColumn(&User{}, "avatar_url") {
		if err := db.Exec(
			`UPDATE users SET avatar_upload_path = replace(avatar_url, '/files/', '')
			 WHERE coalesce(avatar_url, '') <> ''
			   AND avatar_url NOT LIKE '/files/avatars/oidc-%'
			   AND coalesce(avatar_upload_path, '') = ''`).Error; err != nil {
			return err
		}
	}
	if err := db.Exec(
		`UPDATE users SET oidc_picture_url = ''
		 WHERE coalesce(oidc_picture_url, '') <> ''
		   AND oidc_picture_url NOT LIKE 'https://%'`).Error; err != nil {
		return err
	}
	// A NULL here would fail to scan into the plain string the model declares.
	return db.Exec(`UPDATE users SET avatar_upload_path = '' WHERE avatar_upload_path IS NULL`).Error
}
