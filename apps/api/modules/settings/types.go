package settings

type Settings struct {
	EncryptionKeySet bool `json:"encryption_key_set"`
}

type UpdateRequest struct {
	EncryptionKey *string `json:"encryption_key"`
}

type SettingsResponse struct {
	Settings Settings `json:"settings"`
}
