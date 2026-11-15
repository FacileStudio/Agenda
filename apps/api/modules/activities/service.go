package activities

import (
	"context"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// maxRange bounds one request; the widest view, the agenda, spans three months.
const maxRange = 100 * 24 * time.Hour

// Service reads activities.
type Service struct {
	db *gorm.DB
}

// List returns the user's activities that overlap [from, to), oldest first.
// A running one overlaps every range that starts after it began.
func (s *Service) List(ctx context.Context, userID int64, from, to time.Time) ([]ActivityResponse, error) {
	if !to.After(from) || to.Sub(from) > maxRange {
		return nil, errors.Invalid("to must follow from by at most 100 days")
	}
	var rows []schemas.Activity
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND start_at < ? AND (end_at IS NULL OR end_at > ?)", userID, to, from).
		Order("start_at, id").Find(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list activities", err)
	}
	out := make([]ActivityResponse, len(rows))
	for i, row := range rows {
		out[i] = ActivityResponse{ID: row.ID, Source: row.Source, Title: row.Title, StartAt: row.StartAt, EndAt: row.EndAt, URL: row.URL}
	}
	return out, nil
}
