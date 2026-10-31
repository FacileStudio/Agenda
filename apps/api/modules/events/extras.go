package events

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

const (
	maxAttendees      = 100
	maxReminders      = 20
	maxReminderMinute = 40320
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Attendee is a person invited to an event. Role is required or optional and
// Response is needs-action, accepted, declined or tentative. Agenda records
// the invitation and the answer; it does not send mail.
type Attendee struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Response string `json:"response"`
}

// Reminder fires MinutesBefore minutes ahead of the event start. Method is
// display, audio or email; CalDAV clients deliver it, Agenda does not.
type Reminder struct {
	MinutesBefore int    `json:"minutes_before"`
	Method        string `json:"method"`
}

// extras groups the optional event fields that keep their value when a
// request leaves them out: a nil slice or pointer means "unchanged".
type extras struct {
	attendees []Attendee
	reminders []Reminder
	color     *string
}

func validateStatus(status string) error {
	switch status {
	case "", "confirmed", "tentative", "cancelled":
		return nil
	}
	return errors.Invalid("status must be confirmed, tentative or cancelled")
}

func validateTimezone(name string) error {
	if name == "" {
		return nil
	}
	if _, err := time.LoadLocation(name); err != nil {
		return errors.Invalid("timezone must be an IANA zone name such as Europe/Paris")
	}
	return nil
}

func validateColor(color *string) error {
	if color == nil || *color == "" || colorPattern.MatchString(*color) {
		return nil
	}
	return errors.Invalid("color must be a #RRGGBB hex value or empty")
}

func normalizeAttendees(in []Attendee) ([]Attendee, error) {
	if in == nil {
		return nil, nil
	}
	if len(in) > maxAttendees {
		return nil, errors.Invalid("an event takes at most 100 attendees")
	}
	seen := map[string]bool{}
	out := make([]Attendee, 0, len(in))
	for _, a := range in {
		n, err := normalizeAttendee(a)
		if err != nil {
			return nil, err
		}
		if !seen[n.Email] {
			seen[n.Email] = true
			out = append(out, n)
		}
	}
	return out, nil
}

func normalizeAttendee(a Attendee) (Attendee, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(a.Email))
	if err != nil || addr.Name != "" {
		return a, errors.Invalid("attendee email is not a valid address: " + a.Email)
	}
	a.Email = strings.ToLower(addr.Address)
	a.Name = strings.NewReplacer(`"`, "", "\r", " ", "\n", " ").Replace(strings.TrimSpace(a.Name))
	if a.Role == "" {
		a.Role = "required"
	}
	if a.Role != "required" && a.Role != "optional" {
		return a, errors.Invalid("attendee role must be required or optional")
	}
	if a.Response == "" {
		a.Response = "needs-action"
	}
	switch a.Response {
	case "needs-action", "accepted", "declined", "tentative":
		return a, nil
	}
	return a, errors.Invalid("attendee response must be needs-action, accepted, declined or tentative")
}

func normalizeReminders(in []Reminder) ([]Reminder, error) {
	if in == nil {
		return nil, nil
	}
	if len(in) > maxReminders {
		return nil, errors.Invalid("an event takes at most 20 reminders")
	}
	out := make([]Reminder, 0, len(in))
	for _, r := range in {
		n, err := normalizeReminder(r)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func normalizeReminder(r Reminder) (Reminder, error) {
	if r.Method == "" {
		r.Method = "display"
	}
	if r.Method != "display" && r.Method != "audio" && r.Method != "email" {
		return r, errors.Invalid("reminder method must be display, audio or email")
	}
	if r.MinutesBefore < 0 || r.MinutesBefore > maxReminderMinute {
		return r, errors.Invalid("reminder minutes_before must be between 0 and 40320")
	}
	return r, nil
}

// prepareExtras validates and normalises the optional fields of a request.
func prepareExtras(attendees []Attendee, reminders []Reminder, color *string) (extras, error) {
	if err := validateColor(color); err != nil {
		return extras{}, err
	}
	a, err := normalizeAttendees(attendees)
	if err != nil {
		return extras{}, err
	}
	r, err := normalizeReminders(reminders)
	if err != nil {
		return extras{}, err
	}
	return extras{attendees: a, reminders: r, color: color}, nil
}
