package events

import (
	"context"
	"crypto/rand"
	stderrors "errors"
	"encoding/hex"
	"fmt"
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

func (s *Service) ListEvents(ctx context.Context, userID int64, calendarID int64, from, to *time.Time) ([]EventResponse, error) {
	if err := s.checkCalendarAccess(ctx, userID, calendarID); err != nil {
		return nil, err
	}

	q := s.orm.WithContext(ctx).Where("calendar_id = ?", calendarID)
	if from != nil {
		q = q.Where("end_at >= ?", from)
	}
	if to != nil {
		q = q.Where("start_at <= ?", to)
	}

	var evts []schemas.Event
	if err := q.Order("start_at asc").Find(&evts).Error; err != nil {
		return nil, errors.Internal("failed to list events", err)
	}

	out := make([]EventResponse, len(evts))
	for i, e := range evts {
		out[i] = toResponse(&e)
	}
	return out, nil
}

func (s *Service) CreateEvent(ctx context.Context, userID int64, calendarID int64, req *CreateEventRequest) (*EventResponse, error) {
	if err := s.checkCalendarWriteAccess(ctx, userID, calendarID); err != nil {
		return nil, err
	}

	uid := newUID()
	evt := &schemas.Event{
		CalendarID:     calendarID,
		UID:            uid,
		ETag:           newETag(),
		Title:          req.Title,
		Description:    req.Description,
		Location:       req.Location,
		StartAt:        req.StartAt,
		EndAt:          req.EndAt,
		IsAllDay:       req.IsAllDay,
		RecurrenceRule: req.RecurrenceRule,
		Status:         statusOrDefault(req.Status),
	}
	evt.RawICS = buildRawICS(evt)

	if err := s.orm.WithContext(ctx).Create(evt).Error; err != nil {
		return nil, errors.Internal("failed to create event", err)
	}
	s.bumpSyncToken(ctx, calendarID)

	resp := toResponse(evt)
	return &resp, nil
}

func (s *Service) GetEvent(ctx context.Context, userID int64, eventID int64) (*EventResponse, error) {
	evt, err := s.loadWithAccess(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}
	resp := toResponse(evt)
	return &resp, nil
}

func (s *Service) UpdateEvent(ctx context.Context, userID int64, eventID int64, req *UpdateEventRequest) (*EventResponse, error) {
	evt, err := s.loadWithAccess(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}
	if err := s.checkCalendarWriteAccess(ctx, userID, evt.CalendarID); err != nil {
		return nil, err
	}

	evt.ETag = newETag()
	evt.Title = req.Title
	evt.Description = req.Description
	evt.Location = req.Location
	evt.StartAt = req.StartAt
	evt.EndAt = req.EndAt
	evt.IsAllDay = req.IsAllDay
	evt.RecurrenceRule = req.RecurrenceRule
	evt.Status = statusOrDefault(req.Status)
	evt.RawICS = buildRawICS(evt)

	if err := s.orm.WithContext(ctx).Save(evt).Error; err != nil {
		return nil, errors.Internal("failed to update event", err)
	}
	s.bumpSyncToken(ctx, evt.CalendarID)

	resp := toResponse(evt)
	return &resp, nil
}

func (s *Service) DeleteEvent(ctx context.Context, userID int64, eventID int64) error {
	evt, err := s.loadWithAccess(ctx, userID, eventID)
	if err != nil {
		return err
	}
	if err := s.checkCalendarWriteAccess(ctx, userID, evt.CalendarID); err != nil {
		return err
	}
	if err := s.orm.WithContext(ctx).Delete(evt).Error; err != nil {
		return errors.Internal("failed to delete event", err)
	}
	s.bumpSyncToken(ctx, evt.CalendarID)
	return nil
}

func (s *Service) loadWithAccess(ctx context.Context, userID int64, eventID int64) (*schemas.Event, error) {
	var evt schemas.Event
	if err := s.orm.WithContext(ctx).First(&evt, eventID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("event not found")
		}
		return nil, errors.Internal("failed to load event", err)
	}
	if err := s.checkCalendarAccess(ctx, userID, evt.CalendarID); err != nil {
		return nil, err
	}
	return &evt, nil
}

func (s *Service) checkCalendarAccess(ctx context.Context, userID int64, calendarID int64) error {
	var cal schemas.Calendar
	if err := s.orm.WithContext(ctx).First(&cal, calendarID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("calendar not found")
		}
		return errors.Internal("failed to load calendar", err)
	}
	if cal.OwnerID == userID {
		return nil
	}
	var member schemas.CalendarMember
	err := s.orm.WithContext(ctx).Where("calendar_id = ? AND user_id = ?", calendarID, userID).First(&member).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Forbidden("access denied")
	}
	return err
}

func (s *Service) checkCalendarWriteAccess(ctx context.Context, userID int64, calendarID int64) error {
	var cal schemas.Calendar
	if err := s.orm.WithContext(ctx).First(&cal, calendarID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("calendar not found")
		}
		return errors.Internal("failed to load calendar", err)
	}
	if cal.OwnerID == userID {
		return nil
	}
	var member schemas.CalendarMember
	err := s.orm.WithContext(ctx).Where("calendar_id = ? AND user_id = ?", calendarID, userID).First(&member).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Forbidden("access denied")
	}
	if err != nil {
		return errors.Internal("failed to check access", err)
	}
	if member.Role == "reader" {
		return errors.Forbidden("read-only access")
	}
	return nil
}

func (s *Service) bumpSyncToken(ctx context.Context, calendarID int64) {
	token := strconv.FormatInt(time.Now().UnixNano(), 36)
	s.orm.WithContext(ctx).Model(&schemas.Calendar{}).Where("id = ?", calendarID).Update("sync_token", token)
}

func toResponse(e *schemas.Event) EventResponse {
	return EventResponse{
		ID:             e.ID,
		CalendarID:     e.CalendarID,
		UID:            e.UID,
		ETag:           e.ETag,
		Title:          e.Title,
		Description:    e.Description,
		Location:       e.Location,
		StartAt:        e.StartAt,
		EndAt:          e.EndAt,
		IsAllDay:       e.IsAllDay,
		RecurrenceRule: e.RecurrenceRule,
		Status:         e.Status,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func newUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s@agenda", hex.EncodeToString(b))
}

func newETag() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func statusOrDefault(s string) string {
	switch s {
	case "confirmed", "tentative", "cancelled":
		return s
	default:
		return "confirmed"
	}
}

func buildRawICS(e *schemas.Event) string {
	dtFormat := "20060102T150405Z"
	if e.IsAllDay {
		dtFormat = "20060102"
	}
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//FacileStudio//Agenda//EN\r\n"
	ics += "BEGIN:VEVENT\r\n"
	ics += fmt.Sprintf("UID:%s\r\n", e.UID)
	ics += fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(e.Title))
	ics += fmt.Sprintf("DTSTART:%s\r\n", e.StartAt.UTC().Format(dtFormat))
	ics += fmt.Sprintf("DTEND:%s\r\n", e.EndAt.UTC().Format(dtFormat))
	if e.Description != "" {
		ics += fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(e.Description))
	}
	if e.Location != "" {
		ics += fmt.Sprintf("LOCATION:%s\r\n", escapeICS(e.Location))
	}
	if e.RecurrenceRule != "" {
		ics += fmt.Sprintf("RRULE:%s\r\n", e.RecurrenceRule)
	}
	ics += fmt.Sprintf("STATUS:%s\r\n", strings.ToUpper(e.Status))
	ics += "END:VEVENT\r\nEND:VCALENDAR\r\n"
	return ics
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
