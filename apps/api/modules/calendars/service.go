package calendars

import (
	"context"
	stderrors "errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service implements calendar persistence, access control and membership.
type Service struct {
	orm *gorm.DB
}

// NewService builds a calendar service over the given database.
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

// EnsureCalendar returns the id of the user's calendar with the given slug,
// creating it with the given name and color on first use.
func (s *Service) EnsureCalendar(ctx context.Context, userID int64, slug, name, color string) (int64, error) {
	var cal schemas.Calendar
	err := s.orm.WithContext(ctx).Where("owner_id = ? AND slug = ?", userID, slug).First(&cal).Error
	if err == nil {
		return cal.ID, nil
	}
	if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return 0, errors.Internal("failed to look up calendar", err)
	}
	cal = schemas.Calendar{OwnerID: userID, Slug: slug, Name: name, Color: color, SyncToken: syncToken()}
	if err := s.orm.WithContext(ctx).Create(&cal).Error; err != nil {
		return 0, errors.Internal("failed to create calendar", err)
	}
	return cal.ID, nil
}

// ListCalendars returns the calendars the user can open in one scope: a
// space's own calendar when spaceID is set, otherwise every calendar that
// belongs to no space.
func (s *Service) ListCalendars(ctx context.Context, userID int64, spaceID *int64) ([]CalendarResponse, error) {
	_ = s.EnsurePersonalCalendar(ctx, userID)
	rows, err := schemas.AccessibleCalendars(s.orm.WithContext(ctx), userID)
	if err != nil {
		return nil, errors.Internal("failed to list calendars", err)
	}
	out := []CalendarResponse{}
	for _, r := range rows {
		if inScope(r.SpaceID, spaceID) {
			out = append(out, toResponse(&r.Calendar, r.Role))
		}
	}
	return out, nil
}

func inScope(calendarSpace, wanted *int64) bool {
	if wanted == nil || *wanted == 0 {
		return calendarSpace == nil
	}
	return calendarSpace != nil && *calendarSpace == *wanted
}

func toResponse(cal *schemas.Calendar, role string) CalendarResponse {
	return CalendarResponse{
		ID:          cal.ID,
		OwnerID:     cal.OwnerID,
		SpaceID:     cal.SpaceID,
		Name:        cal.Name,
		Color:       cal.Color,
		Description: cal.Description,
		EchoURL:     cal.EchoURL,
		IsPersonal:  cal.IsPersonal,
		Role:        role,
	}
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

// CreateCalendar adds a personal calendar. A space's calendar comes with
// the space, so a request naming a space is refused.
func (s *Service) CreateCalendar(ctx context.Context, userID int64, req *CreateCalendarRequest) (*CalendarResponse, error) {
	if err := validateCalendarFields(req.Name); err != nil {
		return nil, err
	}
	if req.SpaceID != nil && *req.SpaceID != 0 {
		return nil, errors.Invalid("a space has exactly one calendar, created with the space")
	}
	slug := slugify(req.Name) + "-" + strconv.FormatInt(userID, 10) + "-" + strconv.FormatInt(time.Now().UnixMilli(), 36)
	cal := &schemas.Calendar{
		OwnerID:     userID,
		Slug:        slug,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		EchoURL:     req.EchoURL,
		SyncToken:   syncToken(),
	}
	if err := s.orm.WithContext(ctx).Create(cal).Error; err != nil {
		return nil, errors.Internal("failed to create calendar", err)
	}
	resp := toResponse(cal, "owner")
	return &resp, nil
}

// GetCalendar returns one calendar the user can open.
func (s *Service) GetCalendar(ctx context.Context, userID int64, calendarID int64) (*CalendarResponse, error) {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	resp := toResponse(cal, role)
	return &resp, nil
}

// UpdateCalendar edits a calendar's fields. A space's calendar keeps the
// space's name, which Registre owns.
func (s *Service) UpdateCalendar(ctx context.Context, userID int64, calendarID int64, req *UpdateCalendarRequest) (*CalendarResponse, error) {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return nil, err
	}
	if cal.SpaceID != nil {
		req.Name = cal.Name
	}
	if err := validateCalendarFields(req.Name); err != nil {
		return nil, err
	}
	if role == "reader" {
		return nil, errors.Forbidden("insufficient permissions")
	}
	cal.Name = req.Name
	cal.Color = req.Color
	cal.Description = req.Description
	cal.EchoURL = req.EchoURL
	if err := s.orm.WithContext(ctx).Save(cal).Error; err != nil {
		return nil, errors.Internal("failed to update calendar", err)
	}
	resp := toResponse(cal, role)
	return &resp, nil
}

func (s *Service) DeleteCalendar(ctx context.Context, userID int64, calendarID int64) error {
	cal, role, err := s.loadWithAccess(ctx, userID, calendarID)
	if err != nil {
		return err
	}
	if cal.SpaceID != nil {
		return errors.Invalid("a space's calendar lives as long as the space")
	}
	if role != "owner" {
		return errors.Forbidden("only the owner can delete a calendar")
	}
	if cal.IsPersonal {
		return errors.Invalid("cannot delete personal calendar")
	}
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		eventIDs := tx.Model(&schemas.Event{}).Select("id").Where("calendar_id = ?", calendarID)
		if err := tx.Where("event_id IN (?)", eventIDs).Delete(&schemas.EventAttendee{}).Error; err != nil {
			return err
		}
		if err := tx.Where("calendar_id = ?", calendarID).Delete(&schemas.Event{}).Error; err != nil {
			return err
		}
		if err := tx.Where("calendar_id = ?", calendarID).Delete(&schemas.CalendarMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(cal).Error
	})
}

func (s *Service) loadWithAccess(ctx context.Context, userID int64, calendarID int64) (*schemas.Calendar, string, error) {
	access, found, err := schemas.LoadCalendarAccess(s.orm.WithContext(ctx), userID, calendarID)
	if err != nil {
		return nil, "", errors.Internal("failed to load calendar", err)
	}
	if !found {
		return nil, "", errors.NotFound("calendar not found")
	}
	if access.Role == "" {
		return nil, "", errors.Forbidden("access denied")
	}
	return &access.Calendar, access.Role, nil
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
