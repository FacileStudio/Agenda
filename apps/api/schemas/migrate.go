package schemas

import (
	"context"
	"log/slog"

	"github.com/FacileStudio/Agenda/apps/api/migrations"
	"gorm.io/gorm"
)

// Migrate applies the migrations and then the boot data steps, with no OIDC
// issuer. The tests build their schemas through it, so they run exactly the
// SQL production runs.
func Migrate(db *gorm.DB) error {
	return MigrateWithIssuer(context.Background(), db, "", slog.Default())
}

// MigrateWithIssuer applies the ordered migrations under migrations/ and then
// Reconcile. The issuer is what the identity backfill needs: porte matches an
// account on (provider, subject) and the provider is the issuer, so
// backfilling with a placeholder would leave every existing SSO user
// unmatched and quietly fall through to the email path.
func MigrateWithIssuer(ctx context.Context, db *gorm.DB, issuer string, logger *slog.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := migrations.Up(ctx, migrations.Config(sqlDB, logger)); err != nil {
		return err
	}
	return Reconcile(db, issuer)
}

// Reconcile runs the idempotent data steps that ran at every boot before the
// schema moved to migrations, in the same order. It changes rows, never the
// schema: the avatar columns, one calendar per space, and the credentials
// porte reads. AdoptPorte runs last because it repoints identities at users.
func Reconcile(db *gorm.DB, issuer string) error {
	if err := backfillAvatarSources(db); err != nil {
		return err
	}
	if err := SyncSpaceCalendars(db); err != nil {
		return err
	}
	return AdoptPorte(db, issuer)
}

// backfillAvatarSources moves the two values the avatar is now derived from onto the
// columns that own them.
//
// The uploads move by FILENAME, not by avatar_source. That column was added after the
// upload feature, so the oldest uploaded avatars have it empty, and keying on
// an upload source would quietly drop their picture — it did exactly that on
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
// The final statement normalises NULL upload paths: a NULL would fail to scan
// into the plain string the model declares.
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
	return db.Exec(`UPDATE users SET avatar_upload_path = '' WHERE avatar_upload_path IS NULL`).Error
}
