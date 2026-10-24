package events

import (
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"github.com/emersion/go-ical"
	"github.com/teambition/rrule-go"
)

const (
	maxOccurrences = 1000
	maxIterations  = 100000
)

// normalizeRecurrenceRule trims an RRULE value, drops an optional "RRULE:"
// prefix and rejects anything rrule-go cannot parse, so a bad rule never
// reaches the database or a CalDAV client.
func normalizeRecurrenceRule(rule string) (string, error) {
	rule = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(rule)), "RRULE:")
	if rule == "" {
		return "", nil
	}
	if len(rule) > 500 {
		return "", errors.Invalid("recurrence rule must be 500 characters or fewer")
	}
	if _, err := rrule.StrToROption(rule); err != nil {
		return "", errors.Invalid("invalid recurrence rule")
	}
	return rule, nil
}

// requestLocation resolves an IANA zone name sent by a client, falling back to
// UTC when it is empty or unknown.
func requestLocation(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// expandOccurrences returns the occurrences of a recurring event that overlap
// [from, to]. The series is anchored in the TZID of its DTSTART when the raw
// ICS carries one, else in the viewer's zone, so a weekly 09:00 stays at 09:00
// across DST changes. EXDATEs are skipped and RECURRENCE-ID overrides written
// by CalDAV clients replace the instance they target. Every occurrence keeps
// the series ID: edits apply to the whole series.
func expandOccurrences(e *schemas.Event, viewer *time.Location, from, to time.Time) []schemas.Event {
	master, instances := splitComponents(e.RawICS)
	loc := anchorLocation(master, viewer)
	set, err := seriesSet(e, master, loc)
	if err != nil {
		return overlapping(e, from, to)
	}
	overrides := indexOverrides(instances, loc)

	duration := e.EndAt.Sub(e.StartAt)
	next := set.Iterator()
	var out []schemas.Event
	for i := 0; i < maxIterations && len(out) < maxOccurrences; i++ {
		start, ok := next()
		if !ok || start.After(to) {
			break
		}
		occ := *e
		occ.StartAt = start
		occ.EndAt = addWallClock(start, duration)
		if override := overrides[start.Unix()]; override != nil && !applyOverride(&occ, override, loc) {
			continue
		}
		if !occ.EndAt.Before(from) && !occ.StartAt.After(to) {
			out = append(out, occ)
		}
	}
	return out
}

func seriesSet(e *schemas.Event, master *ical.Component, loc *time.Location) (*rrule.Set, error) {
	opt, err := rrule.StrToROptionInLocation(e.RecurrenceRule, loc)
	if err != nil {
		return nil, err
	}
	opt.Dtstart = e.StartAt.In(loc)
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, err
	}
	set := &rrule.Set{}
	set.RRule(rule)
	set.SetExDates(exceptionDates(master, loc))
	return set, nil
}

func exceptionDates(master *ical.Component, loc *time.Location) []time.Time {
	if master == nil {
		return nil
	}
	var out []time.Time
	for _, prop := range master.Props[ical.PropExceptionDates] {
		for v := range strings.SplitSeq(prop.Value, ",") {
			single := prop
			single.Value = v
			if t, err := single.DateTime(loc); err == nil {
				out = append(out, t)
			}
		}
	}
	return out
}

func indexOverrides(instances []*ical.Component, loc *time.Location) map[int64]*ical.Component {
	out := make(map[int64]*ical.Component, len(instances))
	for _, comp := range instances {
		if t, err := comp.Props.DateTime(ical.PropRecurrenceID, loc); err == nil {
			out[t.Unix()] = comp
		}
	}
	return out
}

func overlapping(e *schemas.Event, from, to time.Time) []schemas.Event {
	if e.EndAt.Before(from) || e.StartAt.After(to) {
		return nil
	}
	return []schemas.Event{*e}
}

// addWallClock adds d to start and corrects for a UTC offset change between
// the two instants, so an occurrence after a DST switch keeps its wall-clock
// length instead of ending an hour early or late.
func addWallClock(start time.Time, d time.Duration) time.Time {
	end := start.Add(d)
	_, startOffset := start.Zone()
	_, endOffset := end.Zone()
	return end.Add(time.Duration(startOffset-endOffset) * time.Second)
}

func splitComponents(raw string) (*ical.Component, []*ical.Component) {
	if raw == "" {
		return nil, nil
	}
	cal, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	if err != nil {
		return nil, nil
	}
	var master *ical.Component
	var instances []*ical.Component
	for _, child := range cal.Children {
		if child.Name != ical.CompEvent {
			continue
		}
		if child.Props.Get(ical.PropRecurrenceID) == nil {
			master = child
		} else {
			instances = append(instances, child)
		}
	}
	return master, instances
}

func anchorLocation(comp *ical.Component, fallback *time.Location) *time.Location {
	if comp == nil {
		return fallback
	}
	prop := comp.Props.Get(ical.PropDateTimeStart)
	if prop == nil {
		return fallback
	}
	if tzid := prop.Params.Get(ical.PropTimezoneID); tzid != "" {
		if loc, err := time.LoadLocation(tzid); err == nil {
			return loc
		}
	}
	return fallback
}

// applyOverride copies a RECURRENCE-ID instance onto occ. It reports false
// when the instance was cancelled and must not be shown.
func applyOverride(occ *schemas.Event, comp *ical.Component, loc *time.Location) bool {
	if strings.EqualFold(propValue(comp, ical.PropStatus), "cancelled") {
		return false
	}
	if start, err := comp.Props.DateTime(ical.PropDateTimeStart, loc); err == nil && !start.IsZero() {
		duration := occ.EndAt.Sub(occ.StartAt)
		occ.StartAt = start
		occ.EndAt = start.Add(duration)
	}
	if end, err := comp.Props.DateTime(ical.PropDateTimeEnd, loc); err == nil && end.After(occ.StartAt) {
		occ.EndAt = end
	}
	if v := propValue(comp, ical.PropSummary); v != "" {
		occ.Title = v
	}
	if v := propValue(comp, ical.PropLocation); v != "" {
		occ.Location = v
	}
	if v := propValue(comp, ical.PropDescription); v != "" {
		occ.Description = v
	}
	return true
}

func propValue(comp *ical.Component, name string) string {
	prop := comp.Props.Get(name)
	if prop == nil {
		return ""
	}
	if v, err := prop.Text(); err == nil {
		return v
	}
	return prop.Value
}
