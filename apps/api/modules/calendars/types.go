package calendars

type CreateCalendarRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type UpdateCalendarRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type ShareCalendarRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"` // reader, writer, admin
}

type CalendarResponse struct {
	ID          int64  `json:"id"`
	OwnerID     int64  `json:"owner_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
	IsPersonal  bool   `json:"is_personal"`
	Role        string `json:"role"` // owner, admin, writer, reader
}

type MemberResponse struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}
