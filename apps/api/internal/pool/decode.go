package pool

import (
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/FacileStudio/antenne-client/go/envelope"
)

const appOpus = string(envelope.AppOpus)

type taskPayload struct {
	ID              int64  `json:"id"`
	FacileID        string `json:"facile_id"`
	Title           string `json:"title"`
	Name            string `json:"name"`
	DueDate         string `json:"due_date"`
	ActorEmail      string `json:"actor_email"`
	OwnerEmail      string `json:"owner_email"`
	RegistreSpaceID string `json:"registre_space_id"`
	Personal        bool   `json:"personal"`
	URL             string `json:"url"`
}

type decoded[T any] struct {
	payload   T
	app       string
	facileID  string
	version   int64
	timestamp string
}

func decodeEnvelope[T any](raw json.RawMessage) (decoded[T], error) {
	var env envelope.Event[*T]
	if err := json.Unmarshal(raw, &env); err != nil {
		return decoded[T]{}, err
	}
	out := decoded[T]{app: string(env.App), facileID: env.FacileID, timestamp: env.Timestamp,
		version: eventVersion(env.IdempotencyKey, env.Timestamp)}
	if env.Payload != nil {
		out.payload = *env.Payload
	}
	return out, nil
}

func decodeTask(raw json.RawMessage) (decoded[taskPayload], error) {
	out, err := decodeEnvelope[taskPayload](raw)
	if out.payload.FacileID == "" {
		out.payload.FacileID = out.facileID
	}
	return out, err
}

// eventVersion orders events of one object: the Unix-millisecond suffix of the
// idempotency key when there is one, else the envelope timestamp.
func eventVersion(idempotencyKey, timestamp string) int64 {
	if i := strings.LastIndex(idempotencyKey, ":"); i >= 0 {
		if ms, err := strconv.ParseInt(idempotencyKey[i+1:], 10, 64); err == nil && ms > 0 {
			return ms
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
		return t.UnixMilli()
	}
	return 0
}

func isLegacyID(task taskPayload) bool {
	return task.ID != 0 && task.FacileID == strconv.FormatInt(task.ID, 10)
}

// linkURL keeps an http or https link from another app and drops anything
// else, so a payload cannot plant a javascript: href in the client.
func linkURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return ""
	}
	return parsed.String()
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
