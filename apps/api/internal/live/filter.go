package live

import (
	"context"

	"github.com/FacileStudio/Agenda/apps/api/schemas"

	"gorm.io/gorm"
)

// Frame is the data of one "change" event on the stream. Calendar marks a
// change to the calendar itself or who can read it, so the client reloads
// its calendar list too; Resync asks it to reload everything.
type Frame struct {
	CalendarID int64 `json:"calendar_id,omitempty"`
	Calendar   bool  `json:"calendar,omitempty"`
	Activity   bool  `json:"activity,omitempty"`
	Resync     bool  `json:"resync,omitempty"`
}

// Access reports whether the user can read the calendar.
type Access func(ctx context.Context, userID, calendarID int64) (bool, error)

// CalendarAccess resolves access the way the calendars, events and CalDAV
// handlers do, through schemas.LoadCalendarAccess.
func CalendarAccess(db *gorm.DB) Access {
	return func(ctx context.Context, userID, calendarID int64) (bool, error) {
		access, _, err := schemas.LoadCalendarAccess(db.WithContext(ctx), userID, calendarID)
		return access.Role != "", err
	}
}

// Filter decides which changes one user's stream may see, caching their
// access per calendar. Any change to a calendar, its shares or its space's
// members refreshes that calendar's entry, which is what keeps the cache
// right for event changes in between.
type Filter struct {
	userID int64
	access Access
	cache  map[int64]bool
}

// NewFilter builds the filter for userID.
func NewFilter(userID int64, access Access) *Filter {
	return &Filter{userID: userID, access: access, cache: map[int64]bool{}}
}

// Frame returns the frame for c and whether the user may receive it. A
// calendar change also goes to a user who just lost access, so their client
// drops the calendar.
func (f *Filter) Frame(ctx context.Context, c Change) (Frame, bool, error) {
	switch c.Kind {
	case KindActivity:
		return Frame{Activity: true}, c.ID == f.userID, nil
	case KindResync:
		return Frame{Resync: true}, true, nil
	case KindCalendar:
		return f.calendarFrame(ctx, c.ID)
	case KindEvent:
		return f.eventFrame(ctx, c.ID)
	}
	return Frame{}, false, nil
}

func (f *Filter) calendarFrame(ctx context.Context, calendarID int64) (Frame, bool, error) {
	before := f.cache[calendarID]
	now, err := f.access(ctx, f.userID, calendarID)
	if err != nil {
		delete(f.cache, calendarID)
		return Frame{}, false, err
	}
	f.cache[calendarID] = now
	return Frame{CalendarID: calendarID, Calendar: true}, before || now, nil
}

func (f *Filter) eventFrame(ctx context.Context, calendarID int64) (Frame, bool, error) {
	allowed, known := f.cache[calendarID]
	if !known {
		var err error
		if allowed, err = f.access(ctx, f.userID, calendarID); err != nil {
			return Frame{}, false, err
		}
		f.cache[calendarID] = allowed
	}
	return Frame{CalendarID: calendarID}, allowed, nil
}
