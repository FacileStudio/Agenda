package events

import (
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"github.com/emersion/go-ical"
)

func openSeries(evt *schemas.Event, when string) (*seriesCtx, error) {
	if evt.RecurrenceRule == "" {
		return nil, errors.Invalid("the event does not recur")
	}
	cal, err := ical.NewDecoder(strings.NewReader(evt.RawICS)).Decode()
	if err != nil || masterOf(cal) == nil {
		return nil, errors.Internal("failed to read the event's calendar data", err)
	}
	master := masterOf(cal)
	c := &seriesCtx{evt: evt, cal: cal, master: master, loc: seriesLocation(evt, master)}
	if p := master.Props.Get(ical.PropDateTimeStart); p != nil {
		c.tzid = p.Params.Get(ical.PropTimezoneID)
	}
	c.at, err = resolveOccurrence(evt, c.loc, when)
	return c, err
}

func resolveOccurrence(evt *schemas.Event, loc *time.Location, when string) (time.Time, error) {
	from, to, err := occurrenceWindow(when, loc)
	if err != nil {
		return time.Time{}, err
	}
	rule, err := seriesRule(evt, loc)
	if err != nil {
		return time.Time{}, errors.Invalid("the event's recurrence rule cannot be expanded")
	}
	found := rule.Between(from, to, true)
	if len(found) == 0 {
		return time.Time{}, errors.Invalid("the event has no occurrence on " + when)
	}
	return found[0], nil
}

func occurrenceWindow(when string, loc *time.Location) (time.Time, time.Time, error) {
	if t, err := time.Parse(time.RFC3339, when); err == nil {
		return t, t, nil
	}
	day, err := time.ParseInLocation("2006-01-02", when, loc)
	if err != nil {
		return time.Time{}, time.Time{}, errors.Invalid("occurrence must be YYYY-MM-DD or an RFC 3339 instant")
	}
	return day, day.AddDate(0, 0, 1).Add(-time.Second), nil
}
