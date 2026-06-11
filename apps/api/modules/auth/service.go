package auth

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"api/internal/authcrypto"
	"api/internal/crypto"
	"api/internal/errors"
	"api/internal/oidcavatar"
	"api/schemas"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const SessionTTL = 30 * 24 * time.Hour

type Service struct {
	orm           *gorm.DB
	storageDir    string
	logger        *slog.Logger
	controller    *Controller
	encryptionKey []byte
}

func NewService(orm *gorm.DB, storageDir string, logger *slog.Logger, encryptionKey []byte) *Service {
	service := &Service{orm: orm, storageDir: storageDir, logger: logger, encryptionKey: encryptionKey}
	service.controller = newController(service)
	return service
}

func (service *Service) encryptToken(plain string) string {
	if len(service.encryptionKey) == 0 || plain == "" {
		return plain
	}
	enc, err := crypto.Encrypt(plain, service.encryptionKey)
	if err != nil {
		service.logger.Warn("failed to encrypt token", slog.Any("error", err))
		return plain
	}
	return enc
}

func (service *Service) decryptToken(cipher string) string {
	if len(service.encryptionKey) == 0 || cipher == "" {
		return cipher
	}
	dec, err := crypto.Decrypt(cipher, service.encryptionKey)
	if err != nil {
		return cipher
	}
	return dec
}

func (service *Service) registerUser(context context.Context, email string, password string) (userID string, token string, err error) {
	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return "", "", errors.Invalid("invalid password")
	}

	record := &schemas.User{
		Email:        email,
		PasswordHash: hash,
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return "", "", errors.Conflict("email already registered")
		}
		return "", "", errors.Internal("failed to create user", err)
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) loginUser(context context.Context, email string, password string) (userID string, token string, err error) {
	var record schemas.User
	err = service.orm.WithContext(context).Where("email = ?", email).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", errors.Unauthorized("invalid credentials")
	}
	if err != nil {
		return "", "", errors.Internal("failed to read user", err)
	}
	if !authcrypto.VerifyPassword(password, record.PasswordHash) {
		return "", "", errors.Unauthorized("invalid credentials")
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(context, token, record.ID); err != nil {
		return "", "", err
	}

	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) insertSession(context context.Context, token string, userID int64) error {
	record := &schemas.Session{
		Token:     authcrypto.HashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionTTL),
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		return errors.Internal("failed to persist session", err)
	}
	return nil
}

func normalizeBearer(authorization string) string {
	value := strings.TrimSpace(authorization)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func (service *Service) authenticateRequest(context context.Context, authorization string) (string, *Data, error) {
	token := normalizeBearer(authorization)
	if token == "" {
		return "", nil, errors.Unauthorized("missing auth token")
	}

	hashed := authcrypto.HashToken(token)

	var out struct {
		UserID    int64
		Email     string
		ExpiresAt time.Time
	}
	err := service.orm.WithContext(context).
		Table("sessions s").
		Select("u.id as user_id, u.email as email, s.expires_at as expires_at").
		Joins("join users u on u.id = s.user_id").
		Where("s.token = ?", hashed).
		Scan(&out).Error
	if err != nil {
		return "", nil, errors.Internal("failed to validate auth token", err)
	}
	if out.UserID != 0 {
		if time.Now().After(out.ExpiresAt) {
			return "", nil, errors.Unauthorized("expired auth token")
		}
		return strconv.FormatInt(out.UserID, 10), &Data{Email: out.Email}, nil
	}

	var apiOut struct {
		UserID int64
		Email  string
	}
	err = service.orm.WithContext(context).
		Table("api_tokens t").
		Select("u.id as user_id, u.email as email").
		Joins("join users u on u.id = t.user_id").
		Where("t.token = ?", hashed).
		Scan(&apiOut).Error
	if err != nil {
		return "", nil, errors.Internal("failed to validate api token", err)
	}
	if apiOut.UserID == 0 {
		return "", nil, errors.Unauthorized("invalid auth token")
	}
	return strconv.FormatInt(apiOut.UserID, 10), &Data{Email: apiOut.Email}, nil
}

func (service *Service) Authenticate(context context.Context, authorization string) (string, any, error) {
	return service.authenticateRequest(context, authorization)
}

func (service *Service) VerifyPassword(ctx context.Context, email, password string) (string, *Data, error) {
	var record schemas.User
	err := service.orm.WithContext(ctx).Where("email = ?", email).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, errors.Unauthorized("invalid credentials")
	}
	if err != nil {
		return "", nil, errors.Internal("failed to read user", err)
	}
	if !authcrypto.VerifyPassword(password, record.PasswordHash) {
		return "", nil, errors.Unauthorized("invalid credentials")
	}
	return strconv.FormatInt(record.ID, 10), &Data{Email: record.Email}, nil
}

func (service *Service) deleteSession(ctx context.Context, token string) error {
	hashed := authcrypto.HashToken(token)
	return service.orm.WithContext(ctx).Where("token = ?", hashed).Delete(&schemas.Session{}).Error
}

func (service *Service) upsertOIDCUser(ctx context.Context, email string, profile oidcavatar.Profile, oauth2Token *oauth2.Token) (userID string, token string, err error) {
	var record schemas.User
	err = service.orm.WithContext(ctx).Where("email = ?", email).First(&record).Error
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", errors.Internal("failed to look up user", err)
	}

	isNew := stderrors.Is(err, gorm.ErrRecordNotFound)
	if isNew {
		record = schemas.User{Email: email}
		if displayName := profile.DisplayName(); displayName != "" {
			record.Name = displayName
		}
		if err := service.orm.WithContext(ctx).Create(&record).Error; err != nil {
			return "", "", errors.Internal("failed to create user", err)
		}
		record.OIDCAccessToken = service.encryptToken(oauth2Token.AccessToken)
		record.OIDCRefreshToken = service.encryptToken(oauth2Token.RefreshToken)
		record.OIDCTokenExpiry = oauth2Token.Expiry
		record.ProfileSyncedAt = time.Now()
		if profile.Picture != "" {
			relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
			if fetchErr != nil {
				service.logger.Warn("failed to fetch OIDC avatar for new user", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
			} else {
				record.AvatarURL = "/files/" + relPath
				record.AvatarSource = "oidc"
				record.OIDCPictureURL = profile.Picture
			}
		}
		service.orm.WithContext(ctx).Save(&record)
	} else {
		if displayName := profile.DisplayName(); displayName != "" {
			record.Name = displayName
		}
		needsAvatar := profile.Picture != "" && (profile.Picture != record.OIDCPictureURL || (record.AvatarSource != "upload" && record.AvatarURL == ""))
		if needsAvatar && record.AvatarSource != "upload" {
			oidcavatar.RemoveFile(service.storageDir, strings.TrimPrefix(record.AvatarURL, "/files/"))
			relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
			if fetchErr != nil {
				service.logger.Warn("failed to fetch OIDC avatar", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
			} else {
				record.AvatarURL = "/files/" + relPath
				record.AvatarSource = "oidc"
			}
			record.OIDCPictureURL = profile.Picture
		}
		record.OIDCAccessToken = service.encryptToken(oauth2Token.AccessToken)
		record.OIDCRefreshToken = service.encryptToken(oauth2Token.RefreshToken)
		record.OIDCTokenExpiry = oauth2Token.Expiry
		record.ProfileSyncedAt = time.Now()
		service.orm.WithContext(ctx).Save(&record)
	}

	token, err = authcrypto.NewToken()
	if err != nil {
		return "", "", errors.Internal("failed to create session", err)
	}
	if err := service.insertSession(ctx, token, record.ID); err != nil {
		return "", "", err
	}
	return strconv.FormatInt(record.ID, 10), token, nil
}

func (service *Service) SyncOIDCProfile(ctx context.Context, userID string, provider *gooidc.Provider, oauth2Cfg oauth2.Config) (bool, error) {
	var record schemas.User
	if err := service.orm.WithContext(ctx).Where("id = ?", userID).First(&record).Error; err != nil {
		return false, errors.Internal("failed to load user", err)
	}

	decAccess := service.decryptToken(record.OIDCAccessToken)
	if decAccess == "" {
		return false, nil
	}

	// Skip the cooldown for users who have no avatar yet so the first
	// post-login sync can fetch it immediately.
	hasAvatar := record.AvatarURL != ""
	if hasAvatar && time.Since(record.ProfileSyncedAt) < 5*time.Minute {
		return false, nil
	}

	storedToken := &oauth2.Token{
		AccessToken:  decAccess,
		RefreshToken: service.decryptToken(record.OIDCRefreshToken),
		Expiry:       record.OIDCTokenExpiry,
	}

	tokenSource := oauth2Cfg.TokenSource(ctx, storedToken)

	userInfo, err := provider.UserInfo(ctx, tokenSource)
	if err != nil {
		service.logger.Warn("OIDC profile sync failed, clearing stored tokens", slog.Int64("user_id", record.ID), slog.Any("error", err))
		record.OIDCAccessToken = ""
		record.OIDCRefreshToken = ""
		record.OIDCTokenExpiry = time.Time{}
		service.orm.WithContext(ctx).Save(&record)
		return false, nil
	}

	var claims struct {
		Name             string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		GivenName        string `json:"given_name"`
		FamilyName       string `json:"family_name"`
		Picture          string `json:"picture"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		service.logger.Warn("failed to parse UserInfo claims", slog.Int64("user_id", record.ID), slog.Any("error", err))
		return false, nil
	}

	profile := oidcavatar.Profile{
		Name:             claims.Name,
		PreferredUsername: claims.PreferredUsername,
		GivenName:        claims.GivenName,
		FamilyName:       claims.FamilyName,
		Picture:          claims.Picture,
	}

	if displayName := profile.DisplayName(); displayName != "" {
		record.Name = displayName
	}

	needsAvatar := profile.Picture != "" && (profile.Picture != record.OIDCPictureURL || (record.AvatarSource != "upload" && record.AvatarURL == ""))
	if needsAvatar && record.AvatarSource != "upload" {
		oidcavatar.RemoveFile(service.storageDir, strings.TrimPrefix(record.AvatarURL, "/files/"))
		relPath, fetchErr := oidcavatar.FetchAvatar(profile.Picture, service.storageDir, record.ID, service.logger)
		if fetchErr != nil {
			service.logger.Warn("failed to fetch OIDC avatar during sync", slog.Int64("user_id", record.ID), slog.Any("error", fetchErr))
		} else {
			record.AvatarURL = "/files/" + relPath
			record.AvatarSource = "oidc"
		}
		record.OIDCPictureURL = profile.Picture
	}

	newToken, tokenErr := tokenSource.Token()
	if tokenErr == nil {
		record.OIDCAccessToken = service.encryptToken(newToken.AccessToken)
		record.OIDCRefreshToken = service.encryptToken(newToken.RefreshToken)
		record.OIDCTokenExpiry = newToken.Expiry
	}

	record.ProfileSyncedAt = time.Now()
	service.orm.WithContext(ctx).Save(&record)

	service.logger.Info("synced OIDC profile", slog.Int64("user_id", record.ID))
	return true, nil
}
