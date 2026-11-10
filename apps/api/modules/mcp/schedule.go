package mcp

import (
	"errors"
	"time"
)

// lastMinute is where an all-day event ends on its last day, as the web
// client stores it.
const lastMinute = 23*time.Hour + 59*time.Minute

// errEndBeforeStart is the refusal for an end that is not after the start.
var errEndBeforeStart = errors.New("end must be after start")

// dayOf is midnight of t's date in loc.
func dayOf(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// bounds are an event's start, end and all-day flag.
type bounds struct {
	start, end time.Time
	allDay     bool
}

// endOf reads an end: for an all-day event, 23:59 of the date given.
func endOf(value string, allDay bool, loc *time.Location) (time.Time, error) {
	t, _, err := parseWhen(value, loc)
	if allDay {
		t = dayOf(t, loc).Add(lastMinute)
	}
	return t, err
}

// lengthFor is how long an event lasts after a change: its old length,
// whole days when it becomes all-day, an hour when it becomes timed.
func lengthFor(old bounds, allDay bool) time.Duration {
	switch {
	case allDay == old.allDay:
		return old.end.Sub(old.start)
	case allDay:
		return lastMinute
	default:
		return time.Hour
	}
}

// span reads a new event's bounds. An all-day event runs from midnight of
// its first date to 23:59 of its last one; a timed event without an end
// lasts an hour.
func span(start, end string, allDay bool, loc *time.Location) (time.Time, time.Time, error) {
	next, err := reschedule(updateInput{Start: &start, End: optional(end), AllDay: &allDay}, bounds{allDay: allDay}, loc)
	return next.start, next.end, err
}

// optional is nil for an empty value.
func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// reschedule applies an update's start, end and all-day changes to old.
// Without a new end the event keeps its length; switching to all-day spans
// whole days, switching to timed lasts an hour.
func reschedule(in updateInput, old bounds, loc *time.Location) (bounds, error) {
	next := bounds{start: old.start, allDay: or(in.AllDay, old.allDay)}
	if in.Start != nil {
		t, _, err := parseWhen(*in.Start, loc)
		if err != nil {
			return old, err
		}
		next.start = t
	}
	if next.allDay {
		next.start = dayOf(next.start, loc)
	}
	length := lengthFor(old, next.allDay)
	if old.start.IsZero() {
		length = lengthFor(bounds{allDay: !next.allDay}, next.allDay)
	}
	next.end = next.start.Add(length)
	if in.End != nil {
		end, err := endOf(*in.End, next.allDay, loc)
		if err != nil {
			return old, err
		}
		next.end = end
	}
	if !next.end.After(next.start) {
		return old, errEndBeforeStart
	}
	return next, nil
}
