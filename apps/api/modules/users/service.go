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

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service implements user profile, avatar and API token operations.
type Service struct {
	orm        *gorm.DB
	storageDir string
	tokens     Auth
	controller *Controller
}

// Auth is the auth service, narrowed to what this module needs of it.
//
// A named API token is a porte session with a label on it and no expiry, so
// there is no second table and no second branch in the authentication path —
// which is what there used to be, and what made api_tokens a credential store
// porte's middleware would not have known to read. The password is porte's for
// the same reason.
type Auth interface {
	Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error)
	Sessions() *session.Manager
	SetPassword(ctx context.Context, userID int64, email, password string) error
}

// NewService builds a user service over the given database and avatar storage.
func NewService(orm *gorm.DB, storageDir string, tokens Auth) *Service {
	service := &Service{orm: orm, storageDir: storageDir, tokens: tokens}
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

// updateUser applies profile, email and password changes to an account.
//
// The password is porte's, not a column on this row: writing
// users.password_hash would look like it worked and change nothing, because
// porte reads the identity table — the old password would keep signing in and
// the new one would never work. The address matters twice over, since porte
// keys a local identity on it; changing the email without moving that key
// leaves the password login answering "invalid credentials" to the right
// password, so the identity is re-keyed first and the password is then set on
// the address the account will actually have.
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

	if email != nil || password != nil {
		var current schemas.User
		if err := service.orm.WithContext(context).Select("email").First(&current, id).Error; err != nil {
			return nil, errors.Internal("failed to read the account", err)
		}
		address := current.Email
		if email != nil {
			address = *email
			if !strings.EqualFold(address, current.Email) {
				if err := service.orm.WithContext(context).Exec(
					`UPDATE porte_identities SET subject = ? WHERE provider = 'local' AND subject = ?`,
					strings.ToLower(strings.TrimSpace(address)),
					strings.ToLower(strings.TrimSpace(current.Email)),
				).Error; err != nil {
					return nil, errors.Internal("failed to move the password to the new address", err)
				}
			}
		}
		if password != nil {
			if err := service.tokens.SetPassword(context, id, address, *password); err != nil {
				return nil, err
			}
		}
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

// storeAvatar persists an uploaded image and points the account at it.
//
// Uploading is the fallback for people the IdP has no photo for, so a photo in
// Porte makes this endpoint unavailable rather than merely outranked: accepting
// the file and then never showing it is the worse failure, because the user
// sees a success and no change.
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

	if record.OIDCPictureURL != "" {
		return nil, errors.Invalid("votre photo est geree par le SSO — modifiez-la dans Porte")
	}

	relativePath, absolutePath, err := service.persistAvatarFile(id, reader, contentType)
	if err != nil {
		return nil, err
	}

	newUploadPath := strings.ReplaceAll(relativePath, string(filepath.Separator), "/")
	oldUploadPath := record.AvatarUploadPath
	record.AvatarUploadPath = newUploadPath

	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		_ = os.Remove(absolutePath)
		return nil, errors.Internal("failed to save avatar", err)
	}

	if oldUploadPath != "" {
		service.removeAvatarFile("/files/" + oldUploadPath)
	}

	return mapUser(record), nil
}

// clearAvatar drops the uploaded avatar. Only the upload is the user's to
// clear: the Porte photo is not deleted from here, because it is not ours and
// the next sync would bring it straight back.
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

	oldUploadPath := record.AvatarUploadPath
	record.AvatarUploadPath = ""
	if err := service.orm.WithContext(context).Save(&record).Error; err != nil {
		return nil, errors.Internal("failed to clear avatar", err)
	}

	if oldUploadPath != "" {
		service.removeAvatarFile("/files/" + oldUploadPath)
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
		AvatarURL:    record.Avatar(),
		AvatarSource: record.AvatarOrigin(),
		CreatedAt:    record.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// createApiToken replaces the caller's token, keeping this app's rule that a
// user holds at most one.
func (service *Service) createApiToken(ctx context.Context, userID string, name string) (string, *porte.Session, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return "", nil, errors.Internal("failed to parse user id", err)
	}
	if err := service.deleteApiToken(ctx, userID); err != nil {
		return "", nil, err
	}

	rawToken, issued, err := service.tokens.Issue(ctx, id, name)
	if err != nil {
		return "", nil, err
	}
	return rawToken, &issued, nil
}

// getApiToken returns the caller's labelled session, or nil.
//
// Only labelled sessions are considered: the unlabelled ones are browser
// logins, and listing those here would show somebody their own laptop as an
// API token and offer to revoke it.
func (service *Service) getApiToken(ctx context.Context, userID string) (*porte.Session, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, errors.Internal("failed to parse user id", err)
	}
	sessions, err := service.tokens.Sessions().List(ctx, id)
	if err != nil {
		return nil, errors.Internal("failed to read api tokens", err)
	}
	for _, candidate := range sessions {
		if candidate.Label != "" {
			return &candidate, nil
		}
	}
	return nil, nil
}

func (service *Service) deleteApiToken(ctx context.Context, userID string) error {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return errors.Internal("failed to parse user id", err)
	}
	manager := service.tokens.Sessions()
	sessions, err := manager.List(ctx, id)
	if err != nil {
		return errors.Internal("failed to read api tokens", err)
	}
	for _, candidate := range sessions {
		if candidate.Label == "" {
			continue
		}
		if err := manager.Revoke(ctx, id, candidate.ID); err != nil {
			return errors.Internal("failed to revoke the api token", err)
		}
	}
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
