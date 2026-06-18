package spaces

import "time"

type CreateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

type SpaceResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MemberResponse struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}
