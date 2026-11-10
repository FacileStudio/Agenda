package schemas

import "time"

// ForfaitState holds the plan entitlements Registre last served, as a forfait
// export document in one row, so a restart enforces the last known plans
// instead of failing open again.
type ForfaitState struct {
	ID        int       `gorm:"column:id;primaryKey;autoIncrement:false"`
	Document  string    `gorm:"column:document;type:text;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

// TableName pins the table name.
func (ForfaitState) TableName() string { return "forfait_state" }
