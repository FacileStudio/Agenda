package crypto

import (
	"encoding/base64"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

func isEncrypted(value string) bool {
	if value == "" {
		return true
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	return len(decoded) > 12
}

func MigrateOIDCTokens(db *gorm.DB, key []byte, logger *slog.Logger) error {
	type row struct {
		ID               int64
		OIDCAccessToken  string
		OIDCRefreshToken string
	}

	var rows []row
	if err := db.Raw("SELECT id, oidc_access_token, oidc_refresh_token FROM users WHERE oidc_access_token != '' OR oidc_refresh_token != ''").Scan(&rows).Error; err != nil {
		return fmt.Errorf("read users: %w", err)
	}

	migrated := 0
	for _, r := range rows {
		needsAccess := r.OIDCAccessToken != "" && !isEncrypted(r.OIDCAccessToken)
		needsRefresh := r.OIDCRefreshToken != "" && !isEncrypted(r.OIDCRefreshToken)
		if !needsAccess && !needsRefresh {
			continue
		}

		updates := map[string]any{}
		if needsAccess {
			enc, err := Encrypt(r.OIDCAccessToken, key)
			if err != nil {
				logger.Error("encrypt OIDC access token failed", slog.Int64("user_id", r.ID), slog.Any("error", err))
				continue
			}
			updates["oidc_access_token"] = enc
		}
		if needsRefresh {
			enc, err := Encrypt(r.OIDCRefreshToken, key)
			if err != nil {
				logger.Error("encrypt OIDC refresh token failed", slog.Int64("user_id", r.ID), slog.Any("error", err))
				continue
			}
			updates["oidc_refresh_token"] = enc
		}
		if err := db.Table("users").Where("id = ?", r.ID).Updates(updates).Error; err != nil {
			logger.Error("update user failed", slog.Int64("user_id", r.ID), slog.Any("error", err))
			continue
		}
		migrated++
	}
	logger.Info("OIDC token migration complete", slog.Int("migrated_users", migrated), slog.Int("total_users", len(rows)))
	return nil
}
