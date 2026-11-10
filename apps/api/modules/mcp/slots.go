package mcp

import (
	"slices"
	"time"
)

// interval is a half-open span of time [start, end).
type interval struct {
	start, end time.Time
}

// workday is the part of each day a slot may fall in.
type workday struct {
	start, end time.Duration
	weekends   bool
}

// merge sorts busy intervals and joins the ones that overlap or touch.
func merge(busy []interval) []interval {
	slices.SortFunc(busy, func(a, b interval) int { return a.start.Compare(b.start) })
	out := []interval{}
	for _, b := range busy {
		if n := len(out); n > 0 && !b.start.After(out[n-1].end) {
			if b.end.After(out[n-1].end) {
				out[n-1].end = b.end
			}
			continue
		}
		out = append(out, b)
	}
	return out
}

// search is one free-slot query: the range, the zone of the working hours,
// the slot length and how many slots to return.
type search struct {
	from, to time.Time
	loc      *time.Location
	day      workday
	length   time.Duration
	limit    int
}

// freeSlots lists the free windows of at least q.length inside the working
// hours of each day of the range, around busy, at most q.limit of them.
func freeSlots(busy []interval, q search) []interval {
	busy = merge(busy)
	var free []interval
	for d := dayOf(q.from, q.loc); d.Before(q.to) && len(free) < q.limit; d = d.AddDate(0, 0, 1) {
		if !q.day.weekends && (d.Weekday() == time.Saturday || d.Weekday() == time.Sunday) {
			continue
		}
		hours := interval{start: later(addClock(d, q.day.start), q.from), end: earlier(addClock(d, q.day.end), q.to)}
		free = append(free, gaps(hours, busy, q.length)...)
	}
	return free[:min(len(free), q.limit)]
}

// gaps lists the parts of within that no busy interval covers, of at least
// length.
func gaps(within interval, busy []interval, length time.Duration) []interval {
	var out []interval
	cursor := within.start
	for _, b := range busy {
		if !b.end.After(cursor) || !b.start.Before(within.end) {
			continue
		}
		if b.start.Sub(cursor) >= length {
			out = append(out, interval{start: cursor, end: b.start})
		}
		cursor = later(cursor, b.end)
	}
	if within.end.Sub(cursor) >= length {
		out = append(out, interval{start: cursor, end: within.end})
	}
	return out
}

// addClock is the wall-clock time offset past midnight d, so a working day
// keeps its hours across a DST change.
func addClock(d time.Time, offset time.Duration) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), int(offset.Hours()), int(offset.Minutes())%60, 0, 0, d.Location())
}

// later is the later of a and b.
func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// earlier is the earlier of a and b.
func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
