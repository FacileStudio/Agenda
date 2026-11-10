package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Limits of agenda_find_free_slots.
const (
	maxSlotDays     = 31
	maxSlots        = 50
	minSlotMinutes  = 5
	maxSlotMinutes  = 720
	defaultDayStart = "09:00"
	defaultDayEnd   = "18:00"
	slotsNote       = "Cancelled and all-day events do not count as busy."
)

// errHours is the refusal for working hours that are not HH:MM, or that end
// before they start.
var errHours = errors.New("day_start and day_end must be HH:MM, with day_end after day_start")

// findFreeSlots is agenda_find_free_slots.
func (t *toolset) findFreeSlots(ctx context.Context, req *sdk.CallToolRequest, in slotsInput) (*sdk.CallToolResult, slotsOutput, error) {
	userID, err := userOf(req)
	if err != nil {
		return nil, slotsOutput{}, err
	}
	if in.DurationMinutes < minSlotMinutes || in.DurationMinutes > maxSlotMinutes {
		return nil, slotsOutput{}, fmt.Errorf("duration_minutes must be %d to %d", minSlotMinutes, maxSlotMinutes)
	}
	loc, err := zone(in.Timezone)
	if err != nil {
		return nil, slotsOutput{}, err
	}
	day, err := workdayOf(in)
	if err != nil {
		return nil, slotsOutput{}, err
	}
	from, to, err := window(in.From, in.To, loc, maxSlotDays)
	if err != nil {
		return nil, slotsOutput{}, err
	}
	found, err := t.eventsIn(ctx, userID, in.CalendarIDs, from, to, loc)
	if err != nil {
		return nil, slotsOutput{}, err
	}
	out := slotsOutput{Slots: []freeSlot{}, Note: slotsNote}
	q := search{from: from, to: to, loc: loc, day: day, length: time.Duration(in.DurationMinutes) * time.Minute, limit: maxSlots}
	for _, s := range freeSlots(busyOf(found), q) {
		out.Slots = append(out.Slots, freeSlot{Start: stamp(s.start, loc), End: stamp(s.end, loc), Minutes: int(s.end.Sub(s.start).Minutes())})
	}
	return nil, out, nil
}

// busyOf is the time events take, leaving out all-day and cancelled ones.
func busyOf(found []events.EventResponse) []interval {
	var busy []interval
	for _, e := range found {
		if !e.IsAllDay && e.Status != "cancelled" {
			busy = append(busy, interval{start: e.StartAt, end: e.EndAt})
		}
	}
	return busy
}

// workdayOf reads the working hours of a search.
func workdayOf(in slotsInput) (workday, error) {
	start, errStart := clock(in.DayStart, defaultDayStart)
	end, errEnd := clock(in.DayEnd, defaultDayEnd)
	if errStart != nil || errEnd != nil || end <= start {
		return workday{}, errHours
	}
	return workday{start: start, end: end, weekends: in.IncludeWeekends}, nil
}

// clock reads HH:MM as an offset past midnight, or fallback when empty.
func clock(value, fallback string) (time.Duration, error) {
	if value == "" {
		value = fallback
	}
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
}
