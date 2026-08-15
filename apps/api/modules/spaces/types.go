package spaces

import "time"

// CreateSpaceRequest is the payload for creating a space.
type CreateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateSpaceRequest is the payload for editing a space.
type UpdateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AddMemberRequest adds a user to a space with the given role.
type AddMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// UpdateRoleRequest changes a space member's role.
type UpdateRoleRequest struct {
	Role string `json:"role"`
}

// SpaceResponse describes a space as returned to its members.
type SpaceResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MemberResponse describes a user's membership in a space.
type MemberResponse struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}
