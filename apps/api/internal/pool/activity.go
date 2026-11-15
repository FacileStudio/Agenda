package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/envelope"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SourceSablier = "sablier"
	SourceEcho    = "echo"

	appSablier = string(envelope.AppSablier)
	appEcho    = string(envelope.AppEcho)
)

// ActivityChannels are the Sablier session and Echo call channels recorded
// as activities.
func ActivityChannels() []string {
	return []string{
		string(catalog.TimeEntryCreated), string(catalog.TimeEntryUpdated), string(catalog.TimeEntryDeleted),
		string(catalog.CallStarted), string(catalog.CallUpdated), string(catalog.CallEnded),
	}
}

type sessionPayload struct {
	FacileID        string  `json:"facile_id"`
	TaskFacileID    string  `json:"task_facile_id"`
	ProjectFacileID string  `json:"project_facile_id"`
	UserEmail       string  `json:"user_email"`
	StartedAt       string  `json:"started_at"`
	StoppedAt       *string `json:"stopped_at"`
	Description     string  `json:"description"`
	UpdatedAt       string  `json:"updated_at"`
	URL             string  `json:"url"`
}

type callPayload struct {
	FacileID          string   `json:"facile_id"`
	RoomSlug          string   `json:"room_slug"`
	RoomName          string   `json:"room_name"`
	StartedAt         string   `json:"started_at"`
	EndedAt           *string  `json:"ended_at"`
	ParticipantEmails []string `json:"participant_emails"`
	URL               string   `json:"url"`
}

// activity is one decoded session or call, ready to store for its users.
type activity struct {
	source, id, title, url string
	start                  time.Time
	end                    *time.Time
	emails                 []string
	version                int64
	gone                   bool
}

// Activities records Sablier time sessions and Echo calls for the Agenda
// users they belong to. A replayed or out-of-order event older than the
// newest one applied to the same session or call is dropped.
type Activities struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewActivities builds the activity recorder over db.
func NewActivities(db *gorm.DB, logger *slog.Logger) *Activities {
	return &Activities{db: db, logger: logger}
}

// Apply handles one session or call event. Echo keys its lifecycle on the
// channel, not the envelope action, and so does this.
func (a *Activities) Apply(ctx context.Context, channel string, raw json.RawMessage) error {
	var (
		act activity
		ok  bool
		err error
	)
	switch channel {
	case string(catalog.TimeEntryCreated), string(catalog.TimeEntryUpdated), string(catalog.TimeEntryDeleted):
		act, ok, err = decodeSession(raw, channel == string(catalog.TimeEntryDeleted))
	case string(catalog.CallStarted), string(catalog.CallUpdated), string(catalog.CallEnded):
		act, ok, err = decodeCall(raw)
	}
	if err != nil {
		return fmt.Errorf("decode %s: %w", channel, err)
	}
	if !ok {
		a.logger.Debug("pool: activity event ignored", slog.String("channel", channel))
		return nil
	}
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return store(ctx, tx, act) })
}

func store(ctx context.Context, tx *gorm.DB, act activity) error {
	stale, err := isStale(ctx, tx, act.source, act.id, act.version)
	if err != nil || stale {
		return err
	}
	if act.end, err = keptEnd(tx, act); err != nil {
		return err
	}
	users, err := resolveUsers(ctx, tx, act)
	if err != nil {
		return err
	}
	for _, userID := range users {
		if err := upsertActivity(tx, act, userID); err != nil {
			return err
		}
	}
	if err := dropOthers(tx, act, users); err != nil {
		return err
	}
	return recordVersion(ctx, tx, act.source, act.id, act.version)
}

// keptEnd is the end to store. Echo can send a call.updated after call.ended
// (a late join) without ended_at, and an ended call never reopens, so a
// running Echo call takes the end a stored row already has.
func keptEnd(tx *gorm.DB, act activity) (*time.Time, error) {
	if act.source != SourceEcho || act.end != nil {
		return act.end, nil
	}
	var ends []time.Time
	err := tx.Model(&schemas.Activity{}).Where("source = ? AND external_id = ? AND end_at IS NOT NULL", act.source, act.id).
		Order("end_at DESC").Limit(1).Pluck("end_at", &ends).Error
	if err != nil || len(ends) == 0 {
		return nil, err
	}
	return &ends[0], nil
}

// resolveUsers is every Agenda user the activity belongs to, none once it is
// gone.
func resolveUsers(ctx context.Context, tx *gorm.DB, act activity) ([]int64, error) {
	seen := map[int64]bool{}
	users := []int64{}
	if act.gone {
		return users, nil
	}
	for _, email := range act.emails {
		id, found, err := findUser(ctx, tx, email)
		if err != nil {
			return nil, err
		}
		if found && !seen[id] {
			seen[id] = true
			users = append(users, id)
		}
	}
	return users, nil
}

// dropOthers removes the rows of the activity for anyone not in users.
func dropOthers(tx *gorm.DB, act activity, users []int64) error {
	drop := tx.Where("source = ? AND external_id = ?", act.source, act.id)
	if len(users) > 0 {
		drop = drop.Where("user_id NOT IN ?", users)
	}
	if err := drop.Delete(&schemas.Activity{}).Error; err != nil {
		return fmt.Errorf("remove activity rows: %w", err)
	}
	return nil
}

func upsertActivity(tx *gorm.DB, act activity, userID int64) error {
	row := schemas.Activity{UserID: userID, Source: act.source, ExternalID: act.id, Title: act.title,
		StartAt: act.start, EndAt: act.end, URL: act.url}
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "external_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "start_at", "end_at", "url", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("store activity: %w", err)
	}
	return nil
}
