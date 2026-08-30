package auth

// RegisterRequest is the payload for creating a new local account.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the payload for authenticating a local account.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse carries the authenticated user's ID and session token.
type AuthResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// BackchannelLogoutRequest is the form payload sent by identity providers.
type BackchannelLogoutRequest struct {
	LogoutToken string `json:"logout_token"`
}

// Data is the identity payload submitted to the local login flow.
type Data struct {
	Email string `json:"email"`
}

func (d *Data) GetEmail() string {
	if d == nil {
		return ""
	}
	return d.Email
}
