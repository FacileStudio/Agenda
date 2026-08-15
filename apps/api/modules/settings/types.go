package settings

// Settings carries the app-wide settings visible to clients. EncryptionKeySet
// reports whether a key has been configured without ever exposing it.
type Settings struct {
	EncryptionKeySet bool `json:"encryption_key_set"`
}

// UpdateRequest sets app settings. A nil EncryptionKey leaves it unchanged.
type UpdateRequest struct {
	EncryptionKey *string `json:"encryption_key"`
}

// SettingsResponse wraps the current settings for clients.
type SettingsResponse struct {
	Settings Settings `json:"settings"`
}
