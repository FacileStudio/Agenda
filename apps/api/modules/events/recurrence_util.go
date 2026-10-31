package events

import (
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
)

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
