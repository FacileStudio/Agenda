// Package pool mirrors Opus tasks with a due date into Agenda calendars and
// records Sablier sessions and Echo calls as activities, fed by the Antenne
// Pool.
package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/calendars"
	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/catalog"

	"gorm.io/gorm"
)

const (
	SourceOpus = "opus"

	channelTaskCreated = string(catalog.TaskCreated)
	channelTaskUpdated = string(catalog.TaskUpdated)
	channelTaskDeleted = string(catalog.TaskDeleted)
)

// TaskChannels are the Opus task channels the mirror listens to.
func TaskChannels() []string {
	return []string{
		channelTaskCreated, channelTaskUpdated, channelTaskDeleted,
		string(catalog.TaskStatusChanged), string(catalog.TaskActorChanged),
	}
}

// Mirror turns Opus task events into read-only all-day events. A task of a
// Registre space lands in that space's calendar; any other task lands in its
// assignee's default personal calendar.
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
	task := evt.payload
	if evt.app != appOpus || task.FacileID == "" || isLegacyID(task) {
		m.logger.Debug("pool: task event ignored", slog.String("channel", channel), slog.String("app", evt.app))
		return nil
	}
	stale, err := isStale(ctx, m.db, SourceOpus, task.FacileID, evt.version)
	if err != nil || stale {
		return err
	}
	if err := m.apply(ctx, channel, task); err != nil {
		return err
	}
	return recordVersion(ctx, m.db, SourceOpus, task.FacileID, evt.version)
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
	calendarID, err := m.targetCalendar(ctx, task)
	if err != nil {
		return err
	}
	if calendarID == 0 {
		m.logger.Info("pool: no Agenda calendar for task, skipping", slog.String("task", task.FacileID), slog.Bool("assigned", task.ActorEmail != ""))
		return m.events.DeleteExternal(ctx, SourceOpus, task.FacileID)
	}
	title := task.Title
	if strings.TrimSpace(title) == "" {
		title = task.Name
	}
	external := events.External{Source: SourceOpus, ID: task.FacileID, URL: linkURL(task.URL)}
	return m.events.UpsertExternal(ctx, calendarID, external, &events.CreateEventRequest{
		Title:    title,
		StartAt:  due,
		EndAt:    due.Add(allDayEndOffset),
		IsAllDay: true,
	})
}

// targetCalendar is the space calendar for a task of a Registre space Agenda
// knows, otherwise the assignee's default personal calendar, or 0 when the
// task has neither.
func (m *Mirror) targetCalendar(ctx context.Context, task taskPayload) (int64, error) {
	if space := strings.TrimSpace(task.RegistreSpaceID); space != "" && !task.Personal {
		id, err := m.spaceCalendar(ctx, space)
		if err != nil || id != 0 {
			return id, err
		}
	}
	email := strings.TrimSpace(task.ActorEmail)
	if email == "" {
		email = task.OwnerEmail
	}
	userID, found, err := findUser(ctx, m.db, email)
	if err != nil || !found {
		return 0, err
	}
	return m.personalCalendar(ctx, userID)
}

func (m *Mirror) spaceCalendar(ctx context.Context, registreSpaceID string) (int64, error) {
	var ids []int64
	err := m.db.WithContext(ctx).Raw(`SELECT c.id FROM calendars c JOIN spaces s ON s.id = c.space_id
		WHERE s.registre_space_id = ? ORDER BY c.id LIMIT 1`, registreSpaceID).Scan(&ids).Error
	if err != nil {
		return 0, fmt.Errorf("look up the space calendar: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return ids[0], nil
}

func (m *Mirror) personalCalendar(ctx context.Context, userID int64) (int64, error) {
	db := m.db.WithContext(ctx)
	id, err := schemas.PersonalDefaultCalendar(db, userID)
	if err != nil || id != 0 {
		return id, err
	}
	if err := m.calendars.EnsurePersonalCalendar(ctx, userID); err != nil {
		return 0, fmt.Errorf("create the personal calendar: %w", err)
	}
	return schemas.PersonalDefaultCalendar(db, userID)
}
