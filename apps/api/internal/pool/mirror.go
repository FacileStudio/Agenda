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
)

const (
	SourceOpus = "opus"

	calendarName  = "Opus tasks"
	calendarColor = "#8b5cf6"

	channelTaskDeleted = "task.deleted"
)

var TaskChannels = []string{
	"task.created",
	"task.updated",
	channelTaskDeleted,
	"task.status_changed",
	"task.actor_changed",
}

type taskPayload struct {
	ID         int64  `json:"id"`
	FacileID   string `json:"facile_id"`
	Title      string `json:"title"`
	Name       string `json:"name"`
	DueDate    string `json:"due_date"`
	ActorEmail string `json:"actor_email"`
}

type envelope struct {
	FacileID string       `json:"facile_id"`
	Payload  *taskPayload `json:"payload"`
}

func decodeTask(raw json.RawMessage) (taskPayload, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return taskPayload{}, err
	}
	var task taskPayload
	if env.Payload != nil {
		task = *env.Payload
		if task.FacileID == "" {
			task.FacileID = env.FacileID
		}
	} else if err := json.Unmarshal(raw, &task); err != nil {
		return taskPayload{}, err
	}
	if task.FacileID == "" && task.ID != 0 {
		task.FacileID = strconv.FormatInt(task.ID, 10)
	}
	return task, nil
}

func parseDueDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			t = t.UTC()
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
		}
	}
	return time.Time{}, false
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

// Apply handles one task event. It is idempotent: replaying the same event
// leaves a single mirrored event.
func (m *Mirror) Apply(ctx context.Context, channel string, raw json.RawMessage) error {
	task, err := decodeTask(raw)
	if err != nil {
		return fmt.Errorf("decode task event: %w", err)
	}
	if task.FacileID == "" {
		m.logger.Debug("pool: task event without id, skipping", slog.String("channel", channel))
		return nil
	}
	if channel == channelTaskDeleted {
		return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
	}

	due, hasDue := parseDueDate(task.DueDate)
	if !hasDue {
		return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
	}

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
		EndAt:    due.AddDate(0, 0, 1),
		IsAllDay: true,
	})
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
