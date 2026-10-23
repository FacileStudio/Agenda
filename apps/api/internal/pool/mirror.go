// Package pool mirrors Opus tasks with a due date into Agenda calendars,
// fed by the Antenne Pool.
package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SourceOpus = "opus"

	calendarName  = "Opus tasks"
	calendarColor = "#8b5cf6"

	channelTaskCreated = "task.created"
	channelTaskUpdated = "task.updated"
	channelTaskDeleted = "task.deleted"
)

var TaskChannels = []string{
	channelTaskCreated,
	channelTaskUpdated,
	channelTaskDeleted,
	"task.status_changed",
	"task.actor_changed",
}

// Mirror turns Opus task events into read-only all-day events.
type Mirror struct {
	db        *gorm.DB
	events    *events.Service
	calendars *calendars.Service
	logger    *slog.Logger
}

// NewMirror builds a Mirror over the given services.
func NewMirror(db *gorm.DB, eventService *events.Service, calendarService *calendars.Service, logger *slog.Logger) *Mirror {
	return &Mirror{db: db, events: eventService, calendars: calendarService, logger: logger}
}

// Apply handles one task event. Only events whose envelope app is Opus are
// mirrored. An event older than the newest one already applied to the same
// task is dropped, so replays and out-of-order delivery cannot overwrite or
// resurrect newer state. Replaying the same event is idempotent.
func (m *Mirror) Apply(ctx context.Context, channel string, raw json.RawMessage) error {
	evt, err := decodeTask(raw)
	if err != nil {
		return fmt.Errorf("decode task event: %w", err)
	}
	task := evt.task
	if evt.app != appOpus || task.FacileID == "" || isLegacyID(task) {
		m.logger.Debug("pool: task event ignored", slog.String("channel", channel), slog.String("app", evt.app))
		return nil
	}
	stale, err := m.isStale(ctx, task.FacileID, evt.version)
	if err != nil || stale {
		return err
	}
	if err := m.apply(ctx, channel, task); err != nil {
		return err
	}
	return m.recordVersion(ctx, task.FacileID, evt.version)
}

func (m *Mirror) apply(ctx context.Context, channel string, task taskPayload) error {
	if channel == channelTaskDeleted {
		return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
	}
	due, hasDue := parseDueDate(task.DueDate)
	if !hasDue {
		if channel == channelTaskCreated || channel == channelTaskUpdated {
			return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
		}
		return nil
	}
	return m.mirrorTask(ctx, task, due)
}

func (m *Mirror) mirrorTask(ctx context.Context, task taskPayload, due time.Time) error {
	userID, found, err := m.findUser(ctx, task.ActorEmail)
	if err != nil {
		return err
	}
	if !found {
		m.logger.Debug("pool: task has no known assignee, skipping", slog.String("task", task.FacileID))
		return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
	}
	title := task.Title
	if strings.TrimSpace(title) == "" {
		title = task.Name
	}
	calendarID, err := m.calendars.EnsureCalendar(ctx, userID, "opus-tasks-"+strconv.FormatInt(userID, 10), calendarName, calendarColor)
	if err != nil {
		return err
	}
	return m.events.UpsertExternal(ctx, calendarID, SourceOpus, task.FacileID, &events.CreateEventRequest{
		Title:    title,
		StartAt:  due,
		EndAt:    due.Add(allDayEndOffset),
		IsAllDay: true,
	})
}

func (m *Mirror) isStale(ctx context.Context, externalID string, version int64) (bool, error) {
	var stored schemas.ExternalVersion
	err := m.db.WithContext(ctx).Where("source = ? AND external_id = ?", SourceOpus, externalID).Limit(1).Find(&stored).Error
	if err != nil {
		return false, fmt.Errorf("look up event version: %w", err)
	}
	return stored.Version > version, nil
}

func (m *Mirror) recordVersion(ctx context.Context, externalID string, version int64) error {
	row := schemas.ExternalVersion{Source: SourceOpus, ExternalID: externalID, Version: version}
	err := m.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "external_id"}},
		DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("GREATEST(external_versions.version, EXCLUDED.version)")}),
	}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("record event version: %w", err)
	}
	return nil
}

func (m *Mirror) findUser(ctx context.Context, email string) (int64, bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return 0, false, nil
	}
	var user schemas.User
	err := m.db.WithContext(ctx).Select("id").Where("lower(email) = lower(?)", email).Limit(1).Find(&user).Error
	if err != nil {
		return 0, false, fmt.Errorf("look up assignee: %w", err)
	}
	return user.ID, user.ID != 0, nil
}
