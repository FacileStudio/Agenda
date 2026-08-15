package schemas

import "time"

// User is an account row: profile fields, avatar sources, OIDC bindings and
// the porte identity linkage.
type User struct {
	ID               int64     `gorm:"column:id;primaryKey"`
	Email            string    `gorm:"column:email;uniqueIndex"`
	Name             string    `gorm:"column:name"`
	AvatarURL        string    `gorm:"column:avatar_url"`
	AvatarSource     string    `gorm:"column:avatar_source"`
	AvatarUploadPath string    `gorm:"column:avatar_upload_path"`
	OIDCPictureURL   string    `gorm:"column:oidc_picture_url"`
	OIDCSubject      *string   `gorm:"column:oidc_subject;uniqueIndex"`
	OIDCAccessToken  string    `gorm:"column:oidc_access_token"`
	OIDCRefreshToken string    `gorm:"column:oidc_refresh_token"`
	OIDCTokenExpiry  time.Time `gorm:"column:oidc_token_expiry"`
	ProfileSyncedAt  time.Time `gorm:"column:profile_synced_at"`
	PasswordHash     string    `gorm:"column:password_hash"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }

const avatarFilePrefix = "/files/"

// AvatarSelectExpr is Avatar() as SQL, for the joins that read a member's picture without
// loading the row. It has to stay in step with Avatar below — hence both being here, one
// above the other, rather than one in Go and one buried in a Raw() string.
const AvatarSelectExpr = `COALESCE(NULLIF(u.oidc_picture_url, ''), ` +
	`NULLIF('/files/' || COALESCE(u.avatar_upload_path, ''), '/files/'), '')`

// Avatar is the picture to render. It is derived from the two sources rather than stored
// alongside them: a photo set in Porte always wins, an upload shows only when the IdP
// offers none, and because nothing is written there is no third value that can drift out
// of agreement with the two that matter.
func (u User) Avatar() string {
	if u.OIDCPictureURL != "" {
		return u.OIDCPictureURL
	}
	if u.AvatarUploadPath != "" {
		return avatarFilePrefix + u.AvatarUploadPath
	}
	return ""
}

// AvatarOrigin names where Avatar came from, so the client can say *why* uploading is
// unavailable instead of just hiding the control.
func (u User) AvatarOrigin() string {
	switch {
	case u.OIDCPictureURL != "":
		return "oidc"
	case u.AvatarUploadPath != "":
		return "upload"
	default:
		return ""
	}
}
