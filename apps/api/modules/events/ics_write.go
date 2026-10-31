package events

import (
	"bytes"
	"errors"
	"strings"

	"github.com/emersion/go-ical"
)

// decorate adds the fields buildRawICS does not render to a freshly built
// VCALENDAR. Attendees and reminders come from x when it sets them and from
// the previous ICS when it does not, so an edit that never mentions them
// keeps what a CalDAV client or an earlier call stored. Exceptions follow the
// previous ICS only while keepSeries is true, because they are keyed to the
// series' schedule.
func decorate(raw, prevRaw string, x extras, keepSeries bool) (string, error) {
	cal, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	if err != nil {
		return "", err
	}
	master := masterOf(cal)
	if master == nil {
		return "", errors.New("event has no VEVENT")
	}
	prev, prevInstances := splitComponents(prevRaw)
	carryAttendees(master, prev, x.attendees)
	carryAlarms(master, prev, x.reminders)
	if keepSeries && prev != nil {
		carryExceptions(cal, master, prev, prevInstances)
	}
	return encodeCalendar(cal)
}

func masterOf(cal *ical.Calendar) *ical.Component {
	for _, child := range cal.Children {
		if child.Name == ical.CompEvent && child.Props.Get(ical.PropRecurrenceID) == nil {
			return child
		}
	}
	return nil
}

func encodeCalendar(cal *ical.Calendar) (string, error) {
	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(cal); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func carryAttendees(master, prev *ical.Component, set []Attendee) {
	if set != nil {
		for _, a := range set {
			master.Props.Add(attendeeProp(a))
		}
		return
	}
	if prev == nil {
		return
	}
	for _, p := range prev.Props.Values(ical.PropAttendee) {
		copied := p
		master.Props.Add(&copied)
	}
}

func carryAlarms(master, prev *ical.Component, set []Reminder) {
	if set != nil {
		for _, r := range set {
			master.Children = append(master.Children, alarmComponent(r))
		}
		return
	}
	if prev == nil {
		return
	}
	for _, child := range prev.Children {
		if child.Name == ical.CompAlarm {
			master.Children = append(master.Children, child)
		}
	}
}

func carryExceptions(cal *ical.Calendar, master, prev *ical.Component, instances []*ical.Component) {
	for _, p := range prev.Props.Values(ical.PropExceptionDates) {
		copied := p
		master.Props.Add(&copied)
	}
	for _, inst := range instances {
		cal.Children = append(cal.Children, inst)
	}
}
