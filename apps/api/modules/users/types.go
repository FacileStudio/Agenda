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
//
// CurrentPassword is what separates adding a first password from replacing
// one. Sending Password alone against an account that already has a password
// is refused rather than treated as a change, so a borrowed session cannot
// take the account over.
type UpdateRequest struct {
	Name            *string `json:"name"`
	Email           *string `json:"email"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
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

// DeletedResponse reports the deletion of a resource.
type DeletedResponse struct {
	Deleted bool `json:"deleted"`
}

// AvatarUploadRequest represents a multipart avatar file upload.
type AvatarUploadRequest struct {
	Avatar string `json:"avatar" doc:"Binary avatar image file (PNG, JPEG, GIF, WebP)."`
}
