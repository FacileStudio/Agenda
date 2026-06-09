package schemas

import "time"

type User struct {
	ID               int64     `gorm:"column:id;primaryKey"`
	Email            string    `gorm:"column:email;uniqueIndex"`
	Name             string    `gorm:"column:name"`
	AvatarURL        string    `gorm:"column:avatar_url"`
	AvatarSource     string    `gorm:"column:avatar_source"`
	OIDCPictureURL   string    `gorm:"column:oidc_picture_url"`
	OIDCAccessToken  string    `gorm:"column:oidc_access_token"`
	OIDCRefreshToken string    `gorm:"column:oidc_refresh_token"`
	OIDCTokenExpiry  time.Time `gorm:"column:oidc_token_expiry"`
	ProfileSyncedAt  time.Time `gorm:"column:profile_synced_at"`
	PasswordHash     string    `gorm:"column:password_hash"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
