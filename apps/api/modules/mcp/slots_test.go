package mcp

import (
	"testing"
	"time"
)

// TestFreeSlotsAcrossDST checks working hours keep their wall-clock time on
// the night clocks go back, and that overlapping busy spans merge.
func TestFreeSlotsAcrossDST(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	from := time.Date(2026, 10, 24, 0, 0, 0, 0, paris)
	to := time.Date(2026, 10, 26, 0, 0, 0, 0, paris)
	busy := []interval{
		{start: time.Date(2026, 10, 25, 10, 0, 0, 0, paris), end: time.Date(2026, 10, 25, 12, 0, 0, 0, paris)},
		{start: time.Date(2026, 10, 25, 11, 0, 0, 0, paris), end: time.Date(2026, 10, 25, 17, 0, 0, 0, paris)},
	}
	day := workday{start: 9 * time.Hour, end: 18 * time.Hour, weekends: true}
	got := freeSlots(busy, search{from: from, to: to, loc: paris, day: day, length: 30 * time.Minute, limit: 10})
	if len(got) != 3 || got[1].start.Hour() != 9 || got[1].end.Hour() != 10 || got[2].start.Hour() != 17 {
		t.Fatalf("want a full Saturday, then 9-10 and 17-18 on Sunday, got %v", got)
	}
}

// TestRescheduleToAllDay checks switching a timed event to all-day spans
// its whole day.
func TestRescheduleToAllDay(t *testing.T) {
	yes := true
	start := time.Date(2026, 11, 12, 14, 0, 0, 0, time.UTC)
	next, err := reschedule(updateInput{AllDay: &yes}, bounds{start: start, end: start.Add(time.Hour)}, time.UTC)
	if err != nil || !next.allDay || next.start.Hour() != 0 || next.end.Sub(next.start) != lastMinute {
		t.Fatalf("want the whole day, got %+v %v", next, err)
	}
}
