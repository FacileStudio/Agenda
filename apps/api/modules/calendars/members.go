package calendars

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const spaceMembersOnly = "a space's calendar is shared with the space; manage its members in Registre"

// normalizeRole maps legacy client aliases onto canonical roles.
func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "viewer", "reader":
		return "reader"
	case "editor", "writer":
		return "writer"
	case "admin":
		return "admin"
	default:
		return ""
	}
}

// ShareCalendar grants a user a role on a personal calendar.
func (s *Service) ShareCalendar(ctx context.Context, ownerID int64, calendarID int64, req *ShareCalendarRequest) error {
	if err := s.requireShareable(ctx, ownerID, calendarID, true); err != nil {
		return err
	}
	req.Role = normalizeRole(req.Role)
	if req.Role == "" {
		return errors.Invalid("role must be reader, writer, or admin")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return errors.Invalid("email is required")
	}
	var target schemas.User
	if err := s.orm.WithContext(ctx).Where("LOWER(email) = ?", req.Email).First(&target).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to find user", err)
	}
	if target.ID == ownerID {
		return errors.Invalid("cannot share with yourself")
	}
	member := schemas.CalendarMember{CalendarID: calendarID, UserID: target.ID, Role: req.Role}
	result := s.orm.WithContext(ctx).
		Where("calendar_id = ? AND user_id = ?", calendarID, target.ID).
		Assign(schemas.CalendarMember{Role: req.Role}).
		FirstOrCreate(&member)
	if result.Error != nil {
		return errors.Internal("failed to share calendar", result.Error)
	}
	return nil
}

// RemoveMember drops a member from a personal calendar. Owners and admins
// may manage any member; everyone else may only remove themselves, which is
// how leaving works.
func (s *Service) RemoveMember(ctx context.Context, ownerID int64, calendarID int64, memberID int64) error {
	if err := s.requireShareable(ctx, ownerID, calendarID, ownerID != memberID); err != nil {
		return err
	}
	return s.orm.WithContext(ctx).
		Where("calendar_id = ? AND user_id = ?", calendarID, memberID).
		Delete(&schemas.CalendarMember{}).Error
}

func (s *Service) requireShareable(ctx context.Context, userID, calendarID int64, manage bool) error {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if cal.SpaceID != nil {
		return errors.Invalid(spaceMembersOnly)
	}
	if manage && role != "owner" && role != "admin" {
		return errors.Forbidden("insufficient permissions")
	}
	return nil
}

// ListMembers returns who can open a calendar: the space's members for a
// space's calendar, the explicit shares otherwise.
func (s *Service) ListMembers(ctx context.Context, userID int64, calendarID int64) ([]MemberResponse, error) {
	cal, _, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	query := `SELECT u.id AS user_id, u.email, u.name, ` + schemas.AvatarSelectExpr + ` AS avatar_url, cm.role
		FROM calendar_members cm JOIN users u ON u.id = cm.user_id WHERE cm.calendar_id = ?`
	scope := calendarID
	if cal.SpaceID != nil {
		query = `SELECT u.id AS user_id, u.email, u.name, ` + schemas.AvatarSelectExpr + ` AS avatar_url,
			CASE sm.role WHEN 'member' THEN 'writer' ELSE 'admin' END AS role
			FROM space_members sm JOIN users u ON u.id = sm.user_id WHERE sm.space_id = ? ORDER BY sm.joined_at`
		scope = *cal.SpaceID
	}
	out := []MemberResponse{}
	if err := s.orm.WithContext(ctx).Raw(query, scope).Scan(&out).Error; err != nil {
		return nil, errors.Internal("failed to list members", err)
	}
	return out, nil
}
