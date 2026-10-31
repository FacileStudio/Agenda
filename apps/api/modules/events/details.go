package events

import (
	"sort"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"github.com/emersion/go-ical"
)

// OccurrenceException is one occurrence of a recurring event that departs from
// the series: cancelled, or overridden with its own title, place or times.
type OccurrenceException struct {
	OriginalStart time.Time  `json:"original_start"`
	Date          string     `json:"date"`
	Cancelled     bool       `json:"cancelled"`
	Title         string     `json:"title,omitempty"`
	Description   string     `json:"description,omitempty"`
	Location      string     `json:"location,omitempty"`
	Status        string     `json:"status,omitempty"`
	StartAt       *time.Time `json:"start_at,omitempty"`
	EndAt         *time.Time `json:"end_at,omitempty"`
}

type eventDetails struct {
	timezone   string
	attendees  []Attendee
	reminders  []Reminder
	exceptions []OccurrenceException
}

// readDetails derives the fields kept only in an event's raw ICS: attendees,
// reminders, exceptions and the anchoring zone. The ICS is the source of
// truth so that what a CalDAV client wrote is what the API reports.
func readDetails(e *schemas.Event) eventDetails {
	master, instances := splitComponents(e.RawICS)
	d := eventDetails{
		timezone:   effectiveTimezone(e, master),
		attendees:  []Attendee{},
		reminders:  []Reminder{},
		exceptions: []OccurrenceException{},
	}
	if master == nil {
		return d
	}
	for _, p := range master.Props.Values(ical.PropAttendee) {
		d.attendees = append(d.attendees, attendeeOf(p))
	}
	d.reminders = readReminders(master)
	d.exceptions = readExceptions(e, master, instances)
	return d
}

func effectiveTimezone(e *schemas.Event, master *ical.Component) string {
	if e.Timezone != "" {
		return e.Timezone
	}
	if master != nil {
		if p := master.Props.Get(ical.PropDateTimeStart); p != nil && p.Params.Get(ical.PropTimezoneID) != "" {
			return p.Params.Get(ical.PropTimezoneID)
		}
	}
	return "UTC"
}

func seriesLocation(e *schemas.Event, master *ical.Component) *time.Location {
	if e.IsAllDay {
		return time.UTC
	}
	return anchorLocation(master, requestLocation(effectiveTimezone(e, master)))
}

func attendeeOf(p ical.Prop) Attendee {
	a := Attendee{
		Email:    strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(p.Value, "mailto:"), "MAILTO:")),
		Name:     p.Params.Get(ical.ParamCommonName),
		Role:     "required",
		Response: "needs-action",
	}
	if strings.EqualFold(p.Params.Get(ical.ParamRole), "OPT-PARTICIPANT") {
		a.Role = "optional"
	}
	switch v := strings.ToLower(p.Params.Get(ical.ParamParticipationStatus)); v {
	case "accepted", "declined", "tentative":
		a.Response = v
	}
	return a
}

func readReminders(master *ical.Component) []Reminder {
	out := []Reminder{}
	for _, alarm := range master.Children {
		if alarm.Name != ical.CompAlarm {
			continue
		}
		trigger := alarm.Props.Get(ical.PropTrigger)
		if trigger == nil || trigger.Params.Get(ical.ParamValue) != "" || strings.EqualFold(trigger.Params.Get(ical.ParamRelated), "END") {
			continue
		}
		d, err := trigger.Duration()
		if err != nil || d > 0 {
			continue
		}
		out = append(out, Reminder{MinutesBefore: int(-d / time.Minute), Method: alarmMethod(alarm)})
	}
	return out
}

func alarmMethod(alarm *ical.Component) string {
	switch m := strings.ToLower(propValue(alarm, ical.PropAction)); m {
	case "audio", "email":
		return m
	}
	return "display"
}

func readExceptions(e *schemas.Event, master *ical.Component, instances []*ical.Component) []OccurrenceException {
	loc := seriesLocation(e, master)
	out := []OccurrenceException{}
	for _, t := range exceptionDates(master, loc) {
		out = append(out, OccurrenceException{OriginalStart: t, Date: t.In(loc).Format("2006-01-02"), Cancelled: true})
	}
	for _, comp := range instances {
		if x, ok := overrideException(comp, loc); ok {
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].OriginalStart.Before(out[j].OriginalStart) })
	return out
}

func overrideException(comp *ical.Component, loc *time.Location) (OccurrenceException, bool) {
	rid, err := comp.Props.DateTime(ical.PropRecurrenceID, loc)
	if err != nil {
		return OccurrenceException{}, false
	}
	x := OccurrenceException{
		OriginalStart: rid,
		Date:          rid.In(loc).Format("2006-01-02"),
		Title:         propValue(comp, ical.PropSummary),
		Description:   propValue(comp, ical.PropDescription),
		Location:      propValue(comp, ical.PropLocation),
		Status:        strings.ToLower(propValue(comp, ical.PropStatus)),
	}
	x.Cancelled = x.Status == "cancelled"
	if t, err := comp.Props.DateTime(ical.PropDateTimeStart, loc); err == nil && !t.IsZero() {
		x.StartAt = &t
	}
	if t, err := comp.Props.DateTime(ical.PropDateTimeEnd, loc); err == nil && !t.IsZero() {
		x.EndAt = &t
	}
	return x, true
}
