package pool

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// isStale reports whether a newer event than version was already applied to
// the object (source, externalID).
func isStale(ctx context.Context, db *gorm.DB, source, externalID string, version int64) (bool, error) {
	var stored schemas.ExternalVersion
	err := db.WithContext(ctx).Where("source = ? AND external_id = ?", source, externalID).Limit(1).Find(&stored).Error
	if err != nil {
		return false, fmt.Errorf("look up event version: %w", err)
	}
	return stored.Version > version, nil
}

// recordVersion keeps the newest version applied to (source, externalID). The
// row outlives the object, which is what makes a delete stick.
func recordVersion(ctx context.Context, db *gorm.DB, source, externalID string, version int64) error {
	row := schemas.ExternalVersion{Source: source, ExternalID: externalID, Version: version}
	err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "external_id"}},
		DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("GREATEST(external_versions.version, EXCLUDED.version)")}),
	}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("record event version: %w", err)
	}
	return nil
}

// findUser returns the id of the Agenda user with that email, any case.
func findUser(ctx context.Context, db *gorm.DB, email string) (int64, bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return 0, false, nil
	}
	var user schemas.User
	err := db.WithContext(ctx).Select("id").Where("lower(email) = lower(?)", email).Limit(1).Find(&user).Error
	if err != nil {
		return 0, false, fmt.Errorf("look up assignee: %w", err)
	}
	return user.ID, user.ID != 0, nil
}
