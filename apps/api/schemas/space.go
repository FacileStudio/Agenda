package schemas

import "time"

type Space struct {
	ID          int64         `gorm:"column:id;primaryKey"`
	Name        string        `gorm:"column:name;not null"`
	Description string        `gorm:"column:description"`
	CreatedAt   time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time     `gorm:"column:updated_at;autoUpdateTime"`
	Members     []SpaceMember `gorm:"foreignKey:SpaceID"`
}

func (Space) TableName() string { return "spaces" }

type SpaceMember struct {
	ID       int64     `gorm:"column:id;primaryKey"`
	SpaceID  int64     `gorm:"column:space_id;not null;index;uniqueIndex:idx_space_member"`
	UserID   int64     `gorm:"column:user_id;not null;index;uniqueIndex:idx_space_member"`
	Role     string    `gorm:"column:role;not null;default:'member'"`
	JoinedAt time.Time `gorm:"column:joined_at;autoCreateTime"`
	Space    Space     `gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE"`
	User     User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (SpaceMember) TableName() string { return "space_members" }
