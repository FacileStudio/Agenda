package pool

import (
	"encoding/json"
	"strings"
	"time"
)

func decodeSession(raw json.RawMessage, deleted bool) (activity, bool, error) {
	evt, err := decodeEnvelope[sessionPayload](raw)
	s := evt.payload
	if s.FacileID == "" {
		s.FacileID = evt.facileID
	}
	if err != nil || evt.app != appSablier || s.FacileID == "" {
		return activity{}, false, err
	}
	act := activity{source: SourceSablier, id: s.FacileID, title: strings.TrimSpace(s.Description),
		url: linkURL(s.URL), emails: []string{s.UserEmail}, version: evt.version, gone: deleted}
	if act.version == 0 {
		act.version = eventVersion("", s.UpdatedAt)
	}
	return withTimes(act, s.StartedAt, s.StoppedAt), true, nil
}

// decodeCall orders calls by the envelope timestamp alone: Echo's keys end in
// started, ended or joined:<user id>, never in a time.
func decodeCall(raw json.RawMessage) (activity, bool, error) {
	evt, err := decodeEnvelope[callPayload](raw)
	c := evt.payload
	if c.FacileID == "" {
		c.FacileID = evt.facileID
	}
	if err != nil || evt.app != appEcho || c.FacileID == "" {
		return activity{}, false, err
	}
	title := strings.TrimSpace(c.RoomName)
	if title == "" {
		title = c.RoomSlug
	}
	act := activity{source: SourceEcho, id: c.FacileID, title: title, url: linkURL(c.URL),
		emails: c.ParticipantEmails, version: eventVersion("", evt.timestamp)}
	return withTimes(act, c.StartedAt, c.EndedAt), true, nil
}

// withTimes parses the start and optional end. An unreadable start removes
// the activity instead of storing a guess.
func withTimes(act activity, started string, stopped *string) activity {
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(started))
	if err != nil {
		act.gone = true
		return act
	}
	act.start = start
	if stopped != nil {
		if end, err := time.Parse(time.RFC3339, strings.TrimSpace(*stopped)); err == nil {
			act.end = &end
		}
	}
	return act
}
