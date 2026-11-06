package auth

// BackchannelLogoutRequest is the form payload sent by identity providers.
type BackchannelLogoutRequest struct {
	LogoutToken string `json:"logout_token"`
}
