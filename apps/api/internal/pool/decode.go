package pool

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

const appOpus = "Opus"

type taskPayload struct {
	ID         int64  `json:"id"`
	FacileID   string `json:"facile_id"`
	Title      string `json:"title"`
	Name       string `json:"name"`
	DueDate    string `json:"due_date"`
	ActorEmail string `json:"actor_email"`
	OwnerEmail string `json:"owner_email"`
}

type envelope struct {
	App            string       `json:"app"`
	FacileID       string       `json:"facile_id"`
	Timestamp      string       `json:"timestamp"`
	IdempotencyKey string       `json:"idempotency_key"`
	Payload        *taskPayload `json:"payload"`
}

type decoded struct {
	task    taskPayload
	app     string
	version int64
}

func decodeTask(raw json.RawMessage) (decoded, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return decoded{}, err
	}
	out := decoded{app: env.App, version: eventVersion(env)}
	if env.Payload == nil {
		return out, nil
	}
	out.task = *env.Payload
	if out.task.FacileID == "" {
		out.task.FacileID = env.FacileID
	}
	return out, nil
}

func eventVersion(env envelope) int64 {
	if i := strings.LastIndex(env.IdempotencyKey, ":"); i >= 0 {
		if ms, err := strconv.ParseInt(env.IdempotencyKey[i+1:], 10, 64); err == nil && ms > 0 {
			return ms
		}
	}
	if t, err := time.Parse(time.RFC3339, env.Timestamp); err == nil {
		return t.UnixMilli()
	}
	return 0
}

func isLegacyID(task taskPayload) bool {
	return task.ID != 0 && task.FacileID == strconv.FormatInt(task.ID, 10)
}

const allDayEndOffset = 12 * time.Hour

func parseDueDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			if _, offset := t.Zone(); offset == 0 && layout == time.RFC3339 {
				t = t.In(dueLocation)
			}
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
		}
	}
	return time.Time{}, false
}

var dueLocation = loadDueLocation()

func loadDueLocation() *time.Location {
	name := strings.TrimSpace(os.Getenv("DUE_DATE_TIMEZONE"))
	if name == "" {
		name = "Europe/Paris"
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}
