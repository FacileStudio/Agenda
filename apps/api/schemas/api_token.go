package schemas

import "time"

type ApiToken struct {
	Token     string    `gorm:"column:token;primaryKey"`
	UserID    int64     `gorm:"column:user_id;index"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ApiToken) TableName() string { return "api_tokens" }
