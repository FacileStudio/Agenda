package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service is what is left of Agenda's authentication after porte took the
// credential: the profile lookup the rest of the app reads, and a thin wrapper
// over porte/local so the register and login routes keep their response shape.
type Service struct {
	orm        *gorm.DB
	sessions   *session.Manager
	passwords  *local.Kit
	logger     *slog.Logger
	controller *Controller
}

// NewService wires the auth service onto the given database, session manager
// and porte/local kit.
func NewService(orm *gorm.DB, sessions *session.Manager, passwords *local.Kit, logger *slog.Logger) *Service {
	service := &Service{orm: orm, sessions: sessions, passwords: passwords, logger: logger}
	service.controller = newController(service)
	return service
}

// RequireAuth is porte's session middleware, re-exported so the module routers
// keep passing this one service to middleware.RequireAuth.
func (service *Service) RequireAuth(next http.Handler) http.Handler {
	return service.sessions.RequireAuth(next)
}

// IdentityForUser turns the user id porte authenticated into the identity the
// rest of Agenda reads. It is no longer where authentication happens.
//
// porte deliberately carries neither the email nor any role: what a role may
// do is the app's business, and the profile lives in the app's table. So the
// address is looked up here, which costs the one query the old join cost.
//
// A zero id means the session outlived the user: porte's foreign key cascades
// a delete, so this is a race, and it is still not authenticated.
func (service *Service) IdentityForUser(ctx context.Context, userID int64) (string, string, error) {
	var out struct {
		ID    int64
		Email string
	}
	err := service.orm.WithContext(ctx).
		Model(&schemas.User{}).
		Select("id", "email").
		Where("id = ?", userID).
		Scan(&out).Error
	if err != nil {
		return "", "", errors.Internal("failed to load the account", err)
	}
	if out.ID == 0 {
		return "", "", errors.Unauthorized("invalid auth token")
	}
	return strconv.FormatInt(out.ID, 10), out.Email, nil
}

// Register creates an account through porte/local and signs it in. The cookie
// is set on the way out and the token comes back in the body, so one call
// serves the browser and anything holding the old {user_id, token} shape.
func (service *Service) Register(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (string, string, error) {
	userID, token, err := service.passwords.Register(ctx, w, r, email, "", password)
	if err != nil {
		return "", "", err
	}
	return strconv.FormatInt(userID, 10), token, nil
}

func (service *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (string, string, error) {
	userID, token, err := service.passwords.Login(ctx, w, r, email, password)
	if err != nil {
		return "", "", err
	}
	return strconv.FormatInt(userID, 10), token, nil
}

// VerifyPassword checks a password and returns the user id, issuing nothing.
//
// CalDAV clients send Basic credentials on every request, so this cannot be
// Login: a session row per PROPFIND is an unbounded write to the credential
// table and a Set-Cookie no CalDAV client will ever read. porte/local exposes
// Verify for exactly this, which keeps the enumeration guarantees — an unknown
// address costs a real hash — instead of the app re-deriving argon2 against
// the identity table.
func (service *Service) VerifyPassword(ctx context.Context, email, password string) (int64, error) {
	return service.passwords.Verify(ctx, email, password)
}

// SetPassword is what PATCH /users/me calls when the body carries one.
func (service *Service) SetPassword(ctx context.Context, userID int64, email, password string) error {
	return service.passwords.SetPassword(ctx, userID, email, password)
}

// Issue mints a named API token: a porte session with a label and no expiry,
// which is what the separate api_tokens table used to be.
func (service *Service) Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error) {
	return service.sessions.Issue(ctx, userID, label)
}

// AuthenticateRequest resolves the caller of a route that is not mounted
// behind RequireAuth — the inline-image endpoint, which a browser reaches with
// an <img src> and therefore with a cookie and no header.
func (service *Service) AuthenticateRequest(w http.ResponseWriter, r *http.Request) (int64, error) {
	identity, err := service.sessions.Authenticate(w, r)
	if err != nil {
		return 0, err
	}
	return identity.UserID, nil
}

// Sessions exposes the manager for the modules that list or revoke tokens.
func (service *Service) Sessions() *session.Manager { return service.sessions }
