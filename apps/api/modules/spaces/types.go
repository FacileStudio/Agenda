package spaces

import "time"

// SpaceResponse describes a space as returned to its members.
// RegistreSpaceID is empty until the space is linked to Registre.
type SpaceResponse struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	RegistreSpaceID string    `json:"registre_space_id"`
	Role            string    `json:"role"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
