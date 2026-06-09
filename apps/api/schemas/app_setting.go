package schemas

type AppSetting struct {
	ID            int    `gorm:"primaryKey"`
	EncryptionKey string `gorm:"column:encryption_key;not null;default:''"`
}
