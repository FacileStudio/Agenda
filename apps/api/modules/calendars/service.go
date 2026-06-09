package calendars

import (
	"context"
	stderrors "errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"api/internal/errors"
	"api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) EnsurePersonalCalendar(ctx context.Context, userID int64) error {
	var count int64
	s.orm.WithContext(ctx).Model(&schemas.Calendar{}).Where("owner_id = ? AND is_personal = true", userID).Count(&count)
	if count > 0 {
		return nil
	}
	cal := &schemas.Calendar{
		OwnerID:    userID,
		Slug:       fmt.Sprintf("personal-%d", userID),
		Name:       "Personal",
		Color:      "#3b82f6",
		IsPersonal: true,
		SyncToken:  syncToken(),
	}
	return s.orm.WithContext(ctx).Create(cal).Error
}

func (s *Service) ListCalendars(ctx context.Context, userID int64) ([]CalendarResponse, error) {
	type row struct {
		ID          int64
		OwnerID     int64
		Name        string
		Color       string
		Description string
		IsPersonal  bool
		Role        string
	}

	var rows []row
	err := s.orm.WithContext(ctx).Raw(`
		SELECT c.id, c.owner_id, c.name, c.color, c.description, c.is_personal, 'owner' AS role
		FROM calendars c
		WHERE c.owner_id = ?
		UNION ALL
		SELECT c.id, c.owner_id, c.name, c.color, c.description, c.is_personal, cm.role
		FROM calendars c
		JOIN calendar_members cm ON cm.calendar_id = c.id
		WHERE cm.user_id = ? AND c.owner_id != ?
	`, userID, userID, userID).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list calendars", err)
	}

	out := make([]CalendarResponse, len(rows))
	for i, r := range rows {
		out[i] = CalendarResponse{
			ID:          r.ID,
			OwnerID:     r.OwnerID,
			Name:        r.Name,
			Color:       r.Color,
			Description: r.Description,
			IsPersonal:  r.IsPersonal,
			Role:        r.Role,
		}
	}
	return out, nil
}

func validateCalendarFields(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.Invalid("name is required")
	}
	if len(name) > 255 {
		return errors.Invalid("name must be 255 characters or fewer")
	}
	return nil
}

func (s *Service) CreateCalendar(ctx context.Context, userID int64, req *CreateCalendarRequest) (*CalendarResponse, error) {
	if err := validateCalendarFields(req.Name); err != nil {
		return nil, err
	}
	slug := slugify(req.Name) + "-" + strconv.FormatInt(userID, 10) + "-" + strconv.FormatInt(time.Now().UnixMilli(), 36)
	cal := &schemas.Calendar{
		OwnerID:     userID,
		Slug:        slug,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		SyncToken:   syncToken(),
	}
	if err := s.orm.WithContext(ctx).Create(cal).Error; err != nil {
		return nil, errors.Internal("failed to create calendar", err)
	}
	return &CalendarResponse{
		ID:          cal.ID,
		OwnerID:     cal.OwnerID,
		Name:        cal.Name,
		Color:       cal.Color,
		Description: cal.Description,
		IsPersonal:  false,
		Role:        "owner",
	}, nil
}

func (s *Service) GetCalendar(ctx context.Context, userID int64, calendarID int64) (*CalendarResponse, error) {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	return &CalendarResponse{
		ID:          cal.ID,
		OwnerID:     cal.OwnerID,
		Name:        cal.Name,
		Color:       cal.Color,
		Description: cal.Description,
		IsPersonal:  cal.IsPersonal,
		Role:        role,
	}, nil
}

func (s *Service) UpdateCalendar(ctx context.Context, userID int64, calendarID int64, req *UpdateCalendarRequest) (*CalendarResponse, error) {
	if err := validateCalendarFields(req.Name); err != nil {
		return nil, err
	}
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	if role == "reader" {
		return nil, errors.Forbidden("insufficient permissions")
	}
	cal.Name = req.Name
	cal.Color = req.Color
	cal.Description = req.Description
	if err := s.orm.WithContext(ctx).Save(cal).Error; err != nil {
		return nil, errors.Internal("failed to update calendar", err)
	}
	return &CalendarResponse{
		ID:          cal.ID,
		OwnerID:     cal.OwnerID,
		Name:        cal.Name,
		Color:       cal.Color,
		Description: cal.Description,
		IsPersonal:  cal.IsPersonal,
		Role:        role,
	}, nil
}

func (s *Service) DeleteCalendar(ctx context.Context, userID int64, calendarID int64) error {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return errors.Forbidden("only the owner can delete a calendar")
	}
	if cal.IsPersonal {
		return errors.Invalid("cannot delete personal calendar")
	}
	return s.orm.WithContext(ctx).Delete(cal).Error
}

func (s *Service) ShareCalendar(ctx context.Context, ownerID int64, calendarID int64, req *ShareCalendarRequest) error {
	_, role, err := s.loadWithAccess(ctx, ownerID, calendarID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return errors.Forbidden("insufficient permissions")
	}
	if req.Role != "reader" && req.Role != "writer" && req.Role != "admin" {
		return errors.Invalid("role must be reader, writer, or admin")
	}
	var target schemas.User
	if err := s.orm.WithContext(ctx).Where("email = ?", req.Email).First(&target).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to find user", err)
	}
	if target.ID == ownerID {
		return errors.Invalid("cannot share with yourself")
	}

	member := schemas.CalendarMember{
		CalendarID: calendarID,
		UserID:     target.ID,
		Role:       req.Role,
	}
	result := s.orm.WithContext(ctx).
		Where("calendar_id = ? AND user_id = ?", calendarID, target.ID).
		Assign(schemas.CalendarMember{Role: req.Role}).
		FirstOrCreate(&member)
	if result.Error != nil {
		return errors.Internal("failed to share calendar", result.Error)
	}
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, ownerID int64, calendarID int64, memberID int64) error {
	_, role, err := s.loadWithAccess(ctx, ownerID, calendarID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return errors.Forbidden("insufficient permissions")
	}
	return s.orm.WithContext(ctx).
		Where("calendar_id = ? AND user_id = ?", calendarID, memberID).
		Delete(&schemas.CalendarMember{}).Error
}

func (s *Service) ListMembers(ctx context.Context, userID int64, calendarID int64) ([]MemberResponse, error) {
	if _, _, err := s.loadWithAccess(ctx, userID, calendarID); err != nil {
		return nil, err
	}
	type row struct {
		UserID int64
		Email  string
		Name   string
		Role   string
	}
	var rows []row
	err := s.orm.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.email, u.name, cm.role
		FROM calendar_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.calendar_id = ?
	`, calendarID).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list members", err)
	}
	out := make([]MemberResponse, len(rows))
	for i, r := range rows {
		out[i] = MemberResponse{UserID: r.UserID, Email: r.Email, Name: r.Name, Role: r.Role}
	}
	return out, nil
}

func (s *Service) loadWithAccess(ctx context.Context, userID int64, calendarID int64) (*schemas.Calendar, string, error) {
	var cal schemas.Calendar
	if err := s.orm.WithContext(ctx).First(&cal, calendarID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.NotFound("calendar not found")
		}
		return nil, "", errors.Internal("failed to load calendar", err)
	}
	if cal.OwnerID == userID {
		return &cal, "owner", nil
	}
	var member schemas.CalendarMember
	err := s.orm.WithContext(ctx).Where("calendar_id = ? AND user_id = ?", calendarID, userID).First(&member).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", errors.Forbidden("access denied")
	}
	if err != nil {
		return nil, "", errors.Internal("failed to check access", err)
	}
	return &cal, member.Role, nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "calendar"
	}
	return s
}

func syncToken() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
