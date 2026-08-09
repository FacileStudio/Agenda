package auth

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

type Controller struct {
	service *Service
}

func newController(service *Service) *Controller {
	return &Controller{service: service}
}

// register and login keep Agenda's {user_id, token} response shape. porte
// exports the kit rather than only its routes for exactly this reason: it owns
// the credential and has no idea what an app's response looks like.
//
// The eight-character floor stays here rather than moving to porte's
// MinPasswordLength, so the message an existing user sees does not change on a
// deploy that was not about passwords.
func (controller *Controller) register(w http.ResponseWriter, r *http.Request, req *RegisterRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.Invalid("invalid email")
	}
	if len(req.Password) < 8 {
		return nil, errors.Invalid("password must be at least 8 characters")
	}

	userID, token, err := controller.service.Register(r.Context(), w, r, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
}

func (controller *Controller) login(w http.ResponseWriter, r *http.Request, req *LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, errors.Invalid("email and password required")
	}

	userID, token, err := controller.service.Login(r.Context(), w, r, email, req.Password)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{UserID: userID, Token: token}, nil
}
