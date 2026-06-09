package caldav

import (
	"bytes"
	"context"
	"crypto/rand"
	stderrors "errors"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"api/schemas"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"gorm.io/gorm"
)

const davPrefix = "/dav"

type Backend struct {
	db *gorm.DB
}

func NewBackend(db *gorm.DB) *Backend {
	return &Backend{db: db}
}

func (b *Backend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	user := userFromContext(ctx)
	if user == nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return davPrefix + "/" + user.Email, nil
}

func (b *Backend) CalendarHomeSetPath(ctx context.Context) (string, error) {
	user := userFromContext(ctx)
	if user == nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return davPrefix + "/" + user.Email + "/calendars", nil
}

func (b *Backend) CreateCalendar(ctx context.Context, calendar *caldav.Calendar) error {
	user := userFromContext(ctx)
	if user == nil {
		return webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	// Extract calendarID from path if it already exists, otherwise create new
	_, calID, _, _ := parsePath(calendar.Path)
	if calID != 0 {
		return webdav.NewHTTPError(http.StatusConflict, fmt.Errorf("calendar already exists"))
	}
	// Auto-create via the calendars service would be cleaner, but here we go direct
	slug := fmt.Sprintf("cal-%d-%d", user.ID, time.Now().UnixMilli())
	name := calendar.Name
	if name == "" {
		name = "New Calendar"
	}
	cal := &schemas.Calendar{
		OwnerID:   user.ID,
		Slug:      slug,
		Name:      name,
		Color:     "#6366f1",
		SyncToken: newSyncToken(),
	}
	if err := b.db.WithContext(ctx).Create(cal).Error; err != nil {
		return webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
	return nil
}

func (b *Backend) ListCalendars(ctx context.Context) ([]caldav.Calendar, error) {
	user := userFromContext(ctx)
	if user == nil {
		return nil, webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}

	var cals []schemas.Calendar
	err := b.db.WithContext(ctx).Raw(`
		SELECT c.* FROM calendars c WHERE c.owner_id = ?
		UNION
		SELECT c.* FROM calendars c
		JOIN calendar_members cm ON cm.calendar_id = c.id
		WHERE cm.user_id = ? AND c.owner_id != ?
	`, user.ID, user.ID, user.ID).Scan(&cals).Error
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, err)
	}

	homeSet, _ := b.CalendarHomeSetPath(ctx)
	out := make([]caldav.Calendar, len(cals))
	for i, c := range cals {
		out[i] = toCaldavCalendar(&c, homeSet)
	}
	return out, nil
}

func (b *Backend) GetCalendar(ctx context.Context, reqPath string) (*caldav.Calendar, error) {
	if err := b.validatePathUser(ctx, reqPath); err != nil {
		return nil, err
	}
	cal, err := b.loadCalendarByPath(ctx, reqPath)
	if err != nil {
		return nil, err
	}
	homeSet, _ := b.CalendarHomeSetPath(ctx)
	c := toCaldavCalendar(cal, homeSet)
	return &c, nil
}

func (b *Backend) GetCalendarObject(ctx context.Context, objPath string, req *caldav.CalendarCompRequest) (*caldav.CalendarObject, error) {
	if err := b.validatePathUser(ctx, objPath); err != nil {
		return nil, err
	}
	_, calID, uid, err := parsePath(objPath)
	if err != nil || uid == "" {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("invalid path"))
	}

	if err := b.checkCalendarAccess(ctx, calID); err != nil {
		return nil, err
	}

	uid = strings.TrimSuffix(uid, ".ics")
	var evt schemas.Event
	if err := b.db.WithContext(ctx).Where("uid = ? AND calendar_id = ?", uid, calID).First(&evt).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("event not found"))
		}
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, err)
	}

	return toCalendarObject(&evt, objPath)
}

func (b *Backend) ListCalendarObjects(ctx context.Context, calPath string, req *caldav.CalendarCompRequest) ([]caldav.CalendarObject, error) {
	if err := b.validatePathUser(ctx, calPath); err != nil {
		return nil, err
	}
	_, calID, _, err := parsePath(calPath)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("invalid path"))
	}

	if err := b.checkCalendarAccess(ctx, calID); err != nil {
		return nil, err
	}

	var evts []schemas.Event
	if err := b.db.WithContext(ctx).Where("calendar_id = ?", calID).Find(&evts).Error; err != nil {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, err)
	}

	out := make([]caldav.CalendarObject, 0, len(evts))
	for i := range evts {
		objPath := calPath + "/" + evts[i].UID + ".ics"
		co, err := toCalendarObject(&evts[i], objPath)
		if err != nil {
			continue
		}
		out = append(out, *co)
	}
	return out, nil
}

func (b *Backend) QueryCalendarObjects(ctx context.Context, calPath string, query *caldav.CalendarQuery) ([]caldav.CalendarObject, error) {
	user := userFromContext(ctx)
	if user == nil {
		return nil, webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	if err := b.validatePathUser(ctx, calPath); err != nil {
		return nil, err
	}

	homeSet, _ := b.CalendarHomeSetPath(ctx)

	// Scope to a specific calendar when the path contains one, otherwise search all.
	_, calID, _, _ := parsePath(calPath)
	var cals []schemas.Calendar
	if calID != 0 {
		if err := b.checkCalendarAccess(ctx, calID); err != nil {
			return nil, err
		}
		var cal schemas.Calendar
		if err := b.db.WithContext(ctx).First(&cal, calID).Error; err != nil {
			return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
		}
		cals = []schemas.Calendar{cal}
	} else {
		b.db.WithContext(ctx).Raw(`
			SELECT c.* FROM calendars c WHERE c.owner_id = ?
			UNION
			SELECT c.* FROM calendars c
			JOIN calendar_members cm ON cm.calendar_id = c.id
			WHERE cm.user_id = ? AND c.owner_id != ?
		`, user.ID, user.ID, user.ID).Scan(&cals)
	}

	var all []caldav.CalendarObject
	for _, cal := range cals {
		q := b.db.WithContext(ctx).Where("calendar_id = ?", cal.ID)
		if !query.CompFilter.Start.IsZero() {
			q = q.Where("end_at >= ?", query.CompFilter.Start)
		}
		if !query.CompFilter.End.IsZero() {
			q = q.Where("start_at <= ?", query.CompFilter.End)
		}
		var evts []schemas.Event
		if err := q.Find(&evts).Error; err != nil {
			continue
		}
		cp := homeSet + "/" + strconv.FormatInt(cal.ID, 10)
		for i := range evts {
			objPath := cp + "/" + evts[i].UID + ".ics"
			co, err := toCalendarObject(&evts[i], objPath)
			if err != nil {
				continue
			}
			all = append(all, *co)
		}
	}
	return all, nil
}

func (b *Backend) PutCalendarObject(ctx context.Context, objPath string, calendar *ical.Calendar, opts *caldav.PutCalendarObjectOptions) (*caldav.CalendarObject, error) {
	if err := b.validatePathUser(ctx, objPath); err != nil {
		return nil, err
	}
	_, calID, rawPathSeg, err := parsePath(objPath)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, fmt.Errorf("invalid path"))
	}
	pathUID := strings.TrimSuffix(rawPathSeg, ".ics")

	if err := b.checkCalendarWriteAccess(ctx, calID); err != nil {
		return nil, err
	}

	// ValidateCalendarObject also returns the component type so we can reject VTODO/VJOURNAL.
	eventType, uid, err := caldav.ValidateCalendarObject(calendar)
	if err != nil {
		return nil, caldav.NewPreconditionError(caldav.PreconditionValidCalendarData)
	}
	if eventType != ical.CompEvent {
		return nil, caldav.NewPreconditionError(caldav.PreconditionSupportedCalendarComponent)
	}

	var buf bytes.Buffer
	if err := ical.NewEncoder(&buf).Encode(calendar); err != nil {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
	rawICS := buf.String()

	var evt schemas.Event
	lookupErr := b.db.WithContext(ctx).Where("uid = ? AND calendar_id = ?", uid, calID).First(&evt).Error
	notFound := stderrors.Is(lookupErr, gorm.ErrRecordNotFound)
	if lookupErr != nil && !notFound {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, lookupErr)
	}

	// RFC 4791 §5.3.2.1: reject if another resource already owns this UID in the calendar.
	if !notFound && uid != pathUID {
		return nil, caldav.NewPreconditionError(caldav.PreconditionNoUIDConflict)
	}

	if opts.IfNoneMatch.IsWildcard() && !notFound {
		return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("resource already exists"))
	}
	// RFC 2616 §14.24: If-Match: * requires the resource to exist.
	if opts.IfMatch.IsWildcard() && notFound {
		return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("resource does not exist"))
	}
	if s := string(opts.IfMatch); s != "" && !opts.IfMatch.IsWildcard() {
		if notFound || evt.ETag != strings.Trim(s, `"`) {
			return nil, webdav.NewHTTPError(http.StatusPreconditionFailed, fmt.Errorf("ETag mismatch"))
		}
	}

	comp := findVEvent(calendar)
	if comp == nil {
		return nil, caldav.NewPreconditionError(caldav.PreconditionValidCalendarData)
	}

	if notFound {
		evt = schemas.Event{
			CalendarID: calID,
			UID:        uid,
		}
	}

	evt.ETag = newETag()
	evt.Title = propText(comp, ical.PropSummary)
	evt.Description = propText(comp, ical.PropDescription)
	evt.Location = propText(comp, ical.PropLocation)
	evt.RecurrenceRule = propText(comp, ical.PropRecurrenceRule)
	evt.RawICS = rawICS

	if seqStr := propText(comp, ical.PropSequence); seqStr != "" {
		if seq, err := strconv.Atoi(seqStr); err == nil {
			evt.Sequence = seq
		}
	}

	if status := strings.ToLower(propText(comp, ical.PropStatus)); status != "" {
		evt.Status = status
	} else {
		evt.Status = "confirmed"
	}

	dtstart, _ := comp.Props.DateTime(ical.PropDateTimeStart, nil)
	dtend, _ := comp.Props.DateTime(ical.PropDateTimeEnd, nil)
	if dtstart.IsZero() {
		dtstart = time.Now()
	}
	if dtend.IsZero() {
		dtend = dtstart.Add(time.Hour)
	}
	evt.StartAt = dtstart
	evt.EndAt = dtend

	if dtProp := comp.Props.Get(ical.PropDateTimeStart); dtProp != nil {
		evt.IsAllDay = !strings.Contains(dtProp.Value, "T")
	}

	var dbErr error
	if notFound {
		dbErr = b.db.WithContext(ctx).Create(&evt).Error
	} else {
		dbErr = b.db.WithContext(ctx).Save(&evt).Error
	}
	if dbErr != nil {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, dbErr)
	}

	b.bumpSyncToken(ctx, calID)

	co, coErr := toCalendarObject(&evt, objPath)
	if coErr != nil {
		return nil, coErr
	}
	return co, nil
}

func (b *Backend) DeleteCalendarObject(ctx context.Context, objPath string) error {
	if err := b.validatePathUser(ctx, objPath); err != nil {
		return err
	}
	_, calID, uid, err := parsePath(objPath)
	if err != nil || uid == "" {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("invalid path"))
	}

	if err := b.checkCalendarWriteAccess(ctx, calID); err != nil {
		return err
	}

	uid = strings.TrimSuffix(uid, ".ics")
	result := b.db.WithContext(ctx).Where("uid = ? AND calendar_id = ?", uid, calID).Delete(&schemas.Event{})
	if result.Error != nil {
		return webdav.NewHTTPError(http.StatusInternalServerError, result.Error)
	}
	if result.RowsAffected == 0 {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("event not found"))
	}

	b.bumpSyncToken(ctx, calID)
	return nil
}

// parsePath extracts (email, calendarID, uid) from a DAV path like
// /dav/{email}/calendars/{calendarID}/{uid}.ics
func parsePath(p string) (email string, calendarID int64, uid string, err error) {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, davPrefix+"/")
	parts := strings.SplitN(p, "/", 4)
	if len(parts) < 1 {
		return "", 0, "", fmt.Errorf("invalid path")
	}
	email = parts[0]
	if len(parts) >= 3 {
		calendarID, _ = strconv.ParseInt(parts[2], 10, 64)
	}
	if len(parts) >= 4 {
		uid = parts[3]
	}
	return email, calendarID, uid, nil
}

func (b *Backend) loadCalendarByPath(ctx context.Context, p string) (*schemas.Calendar, error) {
	_, calID, _, err := parsePath(p)
	if err != nil || calID == 0 {
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}
	var cal schemas.Calendar
	if err := b.db.WithContext(ctx).First(&cal, calID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
		}
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
	return &cal, nil
}

func (b *Backend) checkCalendarAccess(ctx context.Context, calID int64) error {
	user := userFromContext(ctx)
	if user == nil {
		return webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	var cal schemas.Calendar
	if err := b.db.WithContext(ctx).First(&cal, calID).Error; err != nil {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}
	if cal.OwnerID == user.ID {
		return nil
	}
	var member schemas.CalendarMember
	err := b.db.WithContext(ctx).Where("calendar_id = ? AND user_id = ?", calID, user.ID).First(&member).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("access denied"))
		}
		return webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
	return nil
}

func (b *Backend) checkCalendarWriteAccess(ctx context.Context, calID int64) error {
	user := userFromContext(ctx)
	if user == nil {
		return webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	var cal schemas.Calendar
	if err := b.db.WithContext(ctx).First(&cal, calID).Error; err != nil {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("calendar not found"))
	}
	if cal.OwnerID == user.ID {
		return nil
	}
	var member schemas.CalendarMember
	err := b.db.WithContext(ctx).Where("calendar_id = ? AND user_id = ?", calID, user.ID).First(&member).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("access denied"))
	}
	if err != nil {
		return webdav.NewHTTPError(http.StatusInternalServerError, err)
	}
	if member.Role == "reader" {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("read-only access"))
	}
	return nil
}

func (b *Backend) bumpSyncToken(ctx context.Context, calID int64) {
	b.db.WithContext(ctx).Model(&schemas.Calendar{}).Where("id = ?", calID).Update("sync_token", newSyncToken())
}

func toCaldavCalendar(c *schemas.Calendar, homeSet string) caldav.Calendar {
	return caldav.Calendar{
		Path:                  homeSet + "/" + strconv.FormatInt(c.ID, 10),
		Name:                  c.Name,
		Description:           c.Description,
		SupportedComponentSet: []string{ical.CompEvent},
	}
}

func toCalendarObject(e *schemas.Event, objPath string) (*caldav.CalendarObject, error) {
	cal, err := ical.NewDecoder(strings.NewReader(e.RawICS)).Decode()
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to parse stored ICS: %w", err))
	}
	// ETag must be the raw unquoted value — go-webdav wraps it in %q when writing headers.
	// ContentLength is 0 because go-webdav re-encodes cal.Data and the byte count may differ.
	return &caldav.CalendarObject{
		Path:    objPath,
		ModTime: e.UpdatedAt,
		ETag:    e.ETag,
		Data:    cal,
	}, nil
}

func findVEvent(cal *ical.Calendar) *ical.Component {
	for _, comp := range cal.Children {
		if comp.Name == ical.CompEvent {
			return comp
		}
	}
	return nil
}

func propText(comp *ical.Component, name string) string {
	if p := comp.Props.Get(name); p != nil {
		return p.Value
	}
	return ""
}

func newETag() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func newSyncToken() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// validatePathUser rejects requests where the email segment in the CalDAV path
// does not match the authenticated user, preventing cross-namespace access.
func (b *Backend) validatePathUser(ctx context.Context, p string) error {
	user := userFromContext(ctx)
	if user == nil {
		return webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	email, _, _, _ := parsePath(p)
	if email != "" && email != user.Email {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("access denied"))
	}
	return nil
}
