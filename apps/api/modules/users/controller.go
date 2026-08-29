package users

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
)

// Controller adapts user service calls to HTTP handlers.
type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

func (controller *Controller) list(context context.Context) (*ListResponse, error) {
	if _, ok := authcontext.IdentityFromContext(context); !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	users, err := controller.service.listUsers(context)
	if err != nil {
		return nil, err
	}

	return &ListResponse{Users: users}, nil
}

func (controller *Controller) get(context context.Context, userID string) (*MeResponse, error) {
	user, err := controller.service.getUser(context, userID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: *user}, nil
}

func (controller *Controller) me(context context.Context) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	user, err := controller.service.getUser(context, identity.UserID)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		user.Email = identity.Email
	}

	return &MeResponse{User: *user}, nil
}

// updateMe applies a profile edit, and takes w and r rather than a context
// because a password change rotates the caller's session and porte writes that
// cookie itself.
//
// A body carrying both halves is refused rather than applied. The password
// ran first and the profile columns second, so a profile failure — a taken
// address — answered 409 with the password already stored, the account's other
// logins already ended and the cookie already rotated: a request reported as
// failed that had changed the credential. porte's session manager writes
// outside any transaction this app can hold, so there is nothing to roll back,
// and reordering only moves which half is orphaned.
func (controller *Controller) updateMe(w http.ResponseWriter, r *http.Request, req *UpdateRequest) (*MeResponse, error) {
	ctx := r.Context()
	identity, ok := authcontext.IdentityFromContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	name, email, err := normalizeProfile(req)
	if err != nil {
		return nil, err
	}

	if name == nil && email == nil && req.Password == nil {
		return nil, errors.Invalid("at least one field must be provided")
	}

	if req.Password != nil && (name != nil || email != nil) {
		return nil, errors.Invalid("change your password and your profile in separate requests")
	}

	if req.Password != nil {
		if err := controller.service.setPassword(w, r, identity.UserID, req); err != nil {
			return nil, err
		}
	}

	user, err := controller.service.updateUser(ctx, identity.UserID, name, email)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: *user}, nil
}

// normalizeProfile trims and validates the profile half of an update. The
// password floor stays here beside them so a too-short password is refused
// before argon2 is paid for, and matches the eight characters main.go pins on
// porte's kit. It counts runes because porte's does: counting bytes let
// "aébcdé" — eight bytes, six characters — clear this check and be refused one
// layer down.
func normalizeProfile(req *UpdateRequest) (*string, *string, error) {
	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if len(trimmed) > 80 {
			return nil, nil, errors.Invalid("name must be at most 80 characters")
		}
		name = &trimmed
	}

	var email *string
	if req.Email != nil {
		normalized := strings.TrimSpace(strings.ToLower(*req.Email))
		if normalized == "" || !strings.Contains(normalized, "@") {
			return nil, nil, errors.Invalid("invalid email")
		}
		email = &normalized
	}

	if req.Password != nil && len([]rune(*req.Password)) < 8 {
		return nil, nil, errors.Invalid("password must be at least 8 characters")
	}

	return name, email, nil
}

func (controller *Controller) deleteAvatar(context context.Context) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	user, err := controller.service.clearAvatar(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{User: *user}, nil
}

func (controller *Controller) uploadAvatar(context context.Context, request *http.Request) (*MeResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}

	if err := request.ParseMultipartForm(5 << 20); err != nil {
		return nil, errors.TooLarge("avatar file is too large")
	}

	file, _, err := request.FormFile("avatar")
	if err != nil {
		return nil, errors.Invalid("avatar file is required")
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return nil, errors.Internal("failed to read avatar file", err)
	}

	contentType := http.DetectContentType(header[:n])
	user, err := controller.service.storeAvatar(context, identity.UserID, io.MultiReader(bytes.NewReader(header[:n]), file), contentType)
	if err != nil {
		return nil, err
	}

	return &MeResponse{User: *user}, nil
}

func (controller *Controller) getApiToken(context context.Context) (*ApiTokenStatusResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	record, err := controller.service.getApiToken(context, identity.UserID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &ApiTokenStatusResponse{HasToken: false}, nil
	}
	return &ApiTokenStatusResponse{
		HasToken:  true,
		Name:      record.Label,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (controller *Controller) createApiToken(context context.Context, req *CreateApiTokenRequest) (*ApiTokenResponse, error) {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "CLI"
	}
	rawToken, record, err := controller.service.createApiToken(context, identity.UserID, name)
	if err != nil {
		return nil, err
	}
	return &ApiTokenResponse{
		Token:     rawToken,
		Name:      record.Label,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (controller *Controller) deleteApiToken(context context.Context) error {
	identity, ok := authcontext.IdentityFromContext(context)
	if !ok {
		return errors.Unauthorized("missing auth")
	}
	return controller.service.deleteApiToken(context, identity.UserID)
}
