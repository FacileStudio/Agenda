package mcp

import (
	"fmt"
	"strings"
	"time"
)

// dateLayout is a calendar date.
const dateLayout = "2006-01-02"

// localLayouts are the zone-less times a model may send, read in the
// request's timezone.
var localLayouts = []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"}

// zone loads an IANA zone; empty means UTC.
func zone(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil || strings.EqualFold(name, "local") {
		return nil, fmt.Errorf("timezone %q is not an IANA name such as Europe/Paris", name)
	}
	return loc, nil
}

// parseWhen reads an RFC 3339 time, a zone-less time in loc, or a date at
// midnight in loc, and reports whether value was a date.
func parseWhen(value string, loc *time.Location) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, false, nil
	}
	for _, layout := range localLayouts {
		if t, err := time.ParseInLocation(layout, value, loc); err == nil {
			return t, false, nil
		}
	}
	if t, err := time.ParseInLocation(dateLayout, value, loc); err == nil {
		return t, true, nil
	}
	return time.Time{}, false, fmt.Errorf("%q is not a date or an RFC 3339 time, such as 2026-11-10 or 2026-11-10T09:00:00+01:00", value)
}

// window reads a from/to range in loc, where a to date means the end of that
// day, and refuses an empty range or one longer than maxDays.
func window(from, to string, loc *time.Location, maxDays int) (time.Time, time.Time, error) {
	start, _, err := parseWhen(from, loc)
	if err != nil {
		return start, start, fmt.Errorf("from: %w", err)
	}
	end, isDate, err := parseWhen(to, loc)
	if err != nil {
		return start, end, fmt.Errorf("to: %w", err)
	}
	if isDate {
		end = end.AddDate(0, 0, 1)
	}
	switch {
	case !end.After(start):
		return start, end, fmt.Errorf("to must be after from")
	case end.Sub(start) > time.Duration(maxDays)*24*time.Hour:
		return start, end, fmt.Errorf("the range is at most %d days: narrow from and to", maxDays)
	}
	return start, end, nil
}

// stamp formats t in loc as RFC 3339.
func stamp(t time.Time, loc *time.Location) string {
	return t.In(loc).Format(time.RFC3339)
}
