package events

import (
	"context"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// attachAttendeeAvatars sets each attendee's AvatarURL from the Agenda account
// holding that address, read through schemas.AvatarSelectExpr so it agrees with
// every other person the API returns. An attendee outside the suite has no
// account and keeps an empty URL, which the client renders as initials.
func (s *Service) attachAttendeeAvatars(ctx context.Context, out []EventResponse) error {
	var emails []string
	for i := range out {
		for _, a := range out[i].Attendees {
			emails = append(emails, a.Email)
		}
	}
	if len(emails) == 0 {
		return nil
	}
	var rows []struct {
		Email     string
		AvatarURL string
	}
	if err := s.orm.WithContext(ctx).Raw(
		`SELECT u.email, `+schemas.AvatarSelectExpr+` AS avatar_url FROM users u WHERE u.email IN ?`, emails,
	).Scan(&rows).Error; err != nil {
		return errors.Internal("failed to load attendee pictures", err)
	}
	avatars := make(map[string]string, len(rows))
	for _, r := range rows {
		avatars[r.Email] = r.AvatarURL
	}
	for i := range out {
		for j := range out[i].Attendees {
			out[i].Attendees[j].AvatarURL = avatars[out[i].Attendees[j].Email]
		}
	}
	return nil
}
