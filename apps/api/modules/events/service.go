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

func validateEventFields(title string, startAt, endAt time.Time) error {
	if strings.TrimSpace(title) == "" {
		return errors.Invalid("title is required")
	}
	if len(title) > 500 {
		return errors.Invalid("title must be 500 characters or fewer")
	}
	if !endAt.IsZero() && !startAt.IsZero() && !endAt.After(startAt) {
		return errors.Invalid("end time must be after start time")
	}
	return nil
}

func (s *Service) CreateEvent(ctx context.Context, userID int64, calendarID int64, req *CreateEventRequest) (*EventResponse, error) {
	if err := validateEventFields(req.Title, req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
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
	if err := validateEventFields(req.Title, req.StartAt, req.EndAt); err != nil {
		return nil, err
	}
	evt, err := s.loadWithAccess(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}
	if err := s.checkCalendarWriteAccess(ctx, userID, evt.CalendarID); err != nil {
		return nil, err
	}

	// Moving the event to another calendar: require write access to the
	// destination too, and remember the source so we can bump both sync
	// tokens (CalDAV clients see a delete from one collection + add to the other).
	sourceCalendarID := evt.CalendarID
	moved := req.CalendarID != 0 && req.CalendarID != evt.CalendarID
	if moved {
		if err := s.checkCalendarWriteAccess(ctx, userID, req.CalendarID); err != nil {
			return nil, err
		}
		evt.CalendarID = req.CalendarID
	}

	evt.ETag = newETag()
	evt.Sequence++
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
	if moved {
		s.bumpSyncToken(ctx, sourceCalendarID)
	}

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

// foldICSLine folds a single ICS property line per RFC 5545 §3.1.
// First line: up to 75 octets. Continuation lines: 1 LWSP + up to 74 octets = 75 octets.
// The trailing CRLF is included in the output.
func foldICSLine(line string) string {
	const firstMax = 75
	const contMax = 74 // 1-byte LWSP prefix + 74 = 75 octets per continuation line
	b := []byte(line)
	if len(b) <= firstMax {
		return line + "\r\n"
	}
	var sb strings.Builder
	sb.Write(b[:firstMax])
	sb.WriteString("\r\n ")
	b = b[firstMax:]
	for len(b) > contMax {
		sb.Write(b[:contMax])
		sb.WriteString("\r\n ")
		b = b[contMax:]
	}
	sb.Write(b)
	sb.WriteString("\r\n")
	return sb.String()
}

func buildRawICS(e *schemas.Event) string {
	const dtFmt = "20060102T150405Z"
	createdAt := e.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := e.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	fold := foldICSLine
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n"
	ics += fold("PRODID:-//FacileStudio//Agenda//EN")
	ics += "BEGIN:VEVENT\r\n"
	ics += fold(fmt.Sprintf("UID:%s", e.UID))
	ics += fold(fmt.Sprintf("DTSTAMP:%s", createdAt.UTC().Format(dtFmt)))
	ics += fold(fmt.Sprintf("CREATED:%s", createdAt.UTC().Format(dtFmt)))
	ics += fold(fmt.Sprintf("LAST-MODIFIED:%s", updatedAt.UTC().Format(dtFmt)))
	ics += fold(fmt.Sprintf("SEQUENCE:%d", e.Sequence))
	ics += fold(fmt.Sprintf("SUMMARY:%s", escapeICS(e.Title)))
	if e.IsAllDay {
		ics += fold(fmt.Sprintf("DTSTART;VALUE=DATE:%s", e.StartAt.UTC().Format("20060102")))
		ics += fold(fmt.Sprintf("DTEND;VALUE=DATE:%s", e.EndAt.UTC().Format("20060102")))
	} else {
		ics += fold(fmt.Sprintf("DTSTART:%s", e.StartAt.UTC().Format(dtFmt)))
		ics += fold(fmt.Sprintf("DTEND:%s", e.EndAt.UTC().Format(dtFmt)))
	}
	if e.Description != "" {
		ics += fold(fmt.Sprintf("DESCRIPTION:%s", escapeICS(e.Description)))
	}
	if e.Location != "" {
		ics += fold(fmt.Sprintf("LOCATION:%s", escapeICS(e.Location)))
	}
	if e.RecurrenceRule != "" {
		ics += fold(fmt.Sprintf("RRULE:%s", e.RecurrenceRule))
	}
	ics += fold(fmt.Sprintf("STATUS:%s", strings.ToUpper(e.Status)))
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
