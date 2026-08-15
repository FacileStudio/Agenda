package users

// User is the public profile of an account.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url"`
	AvatarSource string `json:"avatar_source"`
	CreatedAt    string `json:"created_at"`
}

// MeResponse returns the calling user's own profile.
type MeResponse struct {
	User User `json:"user"`
}

// ListResponse returns all users visible to the caller.
type ListResponse struct {
	Users []User `json:"users"`
}

// UpdateRequest edits a user's profile. Nil fields are left unchanged.
type UpdateRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// ApiTokenResponse carries a freshly issued API token, shown only once.
type ApiTokenResponse struct {
	Token     string `json:"token"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ApiTokenStatusResponse reports whether a token exists and, if so, its name
// and creation time.
type ApiTokenStatusResponse struct {
	HasToken  bool   `json:"has_token"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CreateApiTokenRequest names a new API token.
type CreateApiTokenRequest struct {
	Name string `json:"name"`
}
