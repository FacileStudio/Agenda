package settings

import (
	"context"
	stderrors "errors"
	"strings"

	"api/internal/errors"
	"api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const appSettingID = 1

type Service struct {
	orm        *gorm.DB
	controller *Controller
}

func NewService(orm *gorm.DB) *Service {
	service := &Service{orm: orm}
	service.controller = newController(service)
	return service
}

func (service *Service) getSettings(ctx context.Context) (*Settings, error) {
	var record schemas.AppSetting
	err := service.orm.WithContext(ctx).Where("id = ?", appSettingID).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, errors.Internal("failed to get settings", err)
	}
	return &Settings{
		EncryptionKeySet: record.EncryptionKey != "",
	}, nil
}

func (service *Service) updateSettings(ctx context.Context, req *UpdateRequest) (*Settings, error) {
	updates := map[string]any{}
	if req.EncryptionKey != nil {
		updates["encryption_key"] = strings.TrimSpace(*req.EncryptionKey)
	}

	if len(updates) == 0 {
		return service.getSettings(ctx)
	}

	record := schemas.AppSetting{ID: appSettingID}
	if req.EncryptionKey != nil {
		record.EncryptionKey = strings.TrimSpace(*req.EncryptionKey)
	}

	if err := service.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(keysOf(updates)),
	}).Create(&record).Error; err != nil {
		return nil, errors.Internal("failed to update settings", err)
	}

	return &Settings{
		EncryptionKeySet: record.EncryptionKey != "",
	}, nil
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
