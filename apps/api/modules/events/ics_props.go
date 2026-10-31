package events

import (
	"fmt"
	"strings"

	"github.com/emersion/go-ical"
)

func attendeeProp(a Attendee) *ical.Prop {
	p := ical.NewProp(ical.PropAttendee)
	p.Value = "mailto:" + a.Email
	if a.Name != "" {
		p.Params.Set(ical.ParamCommonName, a.Name)
	}
	p.Params.Set(ical.ParamRole, "REQ-PARTICIPANT")
	if a.Role == "optional" {
		p.Params.Set(ical.ParamRole, "OPT-PARTICIPANT")
	}
	p.Params.Set(ical.ParamParticipationStatus, strings.ToUpper(a.Response))
	if a.Response == "needs-action" {
		p.Params.Set(ical.ParamRSVP, "TRUE")
	}
	return p
}

func alarmComponent(r Reminder) *ical.Component {
	alarm := ical.NewComponent(ical.CompAlarm)
	action := ical.NewProp(ical.PropAction)
	action.Value = strings.ToUpper(r.Method)
	alarm.Props.Add(action)
	trigger := ical.NewProp(ical.PropTrigger)
	trigger.Value = triggerValue(r.MinutesBefore)
	alarm.Props.Add(trigger)
	alarm.Props.SetText(ical.PropDescription, "Reminder")
	if r.Method == "email" {
		alarm.Props.SetText(ical.PropSummary, "Reminder")
	}
	return alarm
}

func triggerValue(minutes int) string {
	if minutes == 0 {
		return "PT0S"
	}
	return fmt.Sprintf("-PT%dM", minutes)
}
