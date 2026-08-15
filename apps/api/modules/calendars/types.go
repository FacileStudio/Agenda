package calendars

// CreateCalendarRequest is the payload for creating a calendar, optionally
// scoped to a space.
type CreateCalendarRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	EchoURL     string `json:"echo_url"`
	SpaceID     *int64 `json:"space_id"`
}

// UpdateCalendarRequest is the payload for editing a calendar's fields.
type UpdateCalendarRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	EchoURL     string `json:"echo_url"`
}

// ShareCalendarRequest grants a user access to a calendar. Role is one of
// reader, writer or admin.
type ShareCalendarRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// CalendarResponse describes a calendar as returned to its members. Role is one
// of owner, admin, writer or reader.
type CalendarResponse struct {
	ID          int64  `json:"id"`
	OwnerID     int64  `json:"owner_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	EchoURL     string `json:"echo_url"`
	IsPersonal  bool   `json:"is_personal"`
	Role        string `json:"role"`
}

// MemberResponse describes a user's membership in a calendar.
type MemberResponse struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}
