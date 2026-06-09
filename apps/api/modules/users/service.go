package users

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"api/internal/authcrypto"
	"api/internal/errors"
	"api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm        *gorm.DB
	storageDir string
	controller *Controller
}

func NewService(orm *gorm.DB, storageDir string) *Service {
	service := &Service{orm: orm, storageDir: storageDir}
	service.controller = newController(service)
	return service
}

func (service *Service) getUser(context context.Context, userID string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	return mapUser(record), nil
}

func (service *Service) listUsers(context context.Context) ([]User, error) {
	var records []schemas.User
	if err := service.orm.WithContext(context).Order("name asc, email asc, id asc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list users", err)
	}

	users := make([]User, 0, len(records))
	for _, record := range records {
		users = append(users, *mapUser(record))
	}

	return users, nil
}

func (service *Service) updateUser(context context.Context, userID string, name *string, email *string, password *string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}
	if email != nil {
		updates["email"] = *email
	}
	if password != nil {
		hash, err := authcrypto.HashPassword(*password)
		if err != nil {
			return nil, errors.Invalid("invalid password")
		}
		updates["password_hash"] = hash
	}

	result := service.orm.WithContext(context).
		Model(&schemas.User{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		if stderrors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("email already registered")
		}
		return nil, errors.Internal("failed to update user", result.Error)
	}
	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	return mapUser(record), nil
}

func (service *Service) storeAvatar(context context.Context, userID string, reader io.Reader, contentType string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	relativePath, absolutePath, err := service.persistAvatarFile(id, reader, contentType)
	if err != nil {
		return nil, err
	}

	newAvatarURL := "/files/" + strings.ReplaceAll(relativePath, string(filepath.Separator), "/")
	oldAvatarURL := record.AvatarURL
	record.AvatarURL = newAvatarURL
	record.AvatarSource = "upload"

	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		_ = os.Remove(absolutePath)
		return nil, errors.Internal("failed to save avatar", err)
	}

	if oldAvatarURL != "" {
		service.removeAvatarFile(oldAvatarURL)
	}

	return mapUser(record), nil
}

func (service *Service) clearAvatar(context context.Context, userID string) (*User, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.User
	if err := service.orm.WithContext(context).Where("id = ?", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to read user", err)
	}

	oldAvatarURL := record.AvatarURL
	record.AvatarURL = ""
	record.AvatarSource = ""
	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		return nil, errors.Internal("failed to clear avatar", err)
	}

	if oldAvatarURL != "" {
		service.removeAvatarFile(oldAvatarURL)
	}

	return mapUser(record), nil
}

func (service *Service) persistAvatarFile(userID int64, reader io.Reader, contentType string) (string, string, error) {
	extension, ok := avatarExtension(contentType)
	if !ok {
		return "", "", errors.Invalid("avatar must be a PNG, JPEG, GIF, or WebP image")
	}

	filename := fmt.Sprintf("user-%d-%d%s", userID, time.Now().UnixNano(), extension)
	relativePath := filepath.Join("avatars", filename)
	absolutePath := filepath.Join(service.storageDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", "", errors.Internal("failed to prepare avatar storage", err)
	}

	file, err := os.Create(absolutePath)
	if err != nil {
		return "", "", errors.Internal("failed to create avatar file", err)
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return "", "", errors.Internal("failed to write avatar file", err)
	}
	if err := file.Close(); err != nil {
		return "", "", errors.Internal("failed to finalize avatar file", err)
	}

	return relativePath, absolutePath, nil
}

func (service *Service) removeAvatarFile(avatarURL string) {
	oldPath := strings.TrimPrefix(avatarURL, "/files/")
	oldAbsolutePath := filepath.Join(service.storageDir, filepath.Clean(oldPath))
	if strings.HasPrefix(oldAbsolutePath, filepath.Clean(filepath.Join(service.storageDir, "avatars"))) {
		_ = os.Remove(oldAbsolutePath)
	}
}

func mapUser(record schemas.User) *User {
	return &User{
		ID:           strconv.FormatInt(record.ID, 10),
		Email:        record.Email,
		Name:         record.Name,
		AvatarURL:    record.AvatarURL,
		AvatarSource: record.AvatarSource,
		CreatedAt:    record.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (service *Service) createApiToken(context context.Context, userID string, name string) (string, *schemas.ApiToken, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", nil, errors.Internal("failed to parse user id", err)
	}

	service.orm.WithContext(context).Where("user_id = ?", id).Delete(&schemas.ApiToken{})

	rawToken, err := authcrypto.NewToken()
	if err != nil {
		return "", nil, errors.Internal("failed to generate token", err)
	}

	hasher := authcrypto.HashToken(rawToken)
	record := &schemas.ApiToken{
		Token:  hasher,
		UserID: id,
		Name:   name,
	}
	if err := service.orm.WithContext(context).Create(record).Error; err != nil {
		return "", nil, errors.Internal("failed to store api token", err)
	}

	return rawToken, record, nil
}

func (service *Service) getApiToken(context context.Context, userID string) (*schemas.ApiToken, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}

	var record schemas.ApiToken
	err = service.orm.WithContext(context).Where("user_id = ?", id).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Internal("failed to read api token", err)
	}
	return &record, nil
}

func (service *Service) deleteApiToken(context context.Context, userID string) error {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return errors.Internal("failed to parse user id", err)
	}

	service.orm.WithContext(context).Where("user_id = ?", id).Delete(&schemas.ApiToken{})
	return nil
}

func avatarExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}
