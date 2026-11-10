package ideas

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"gorm.io/gorm"
)

// Plans answers whether a person or a Registre space has ai.access.
type Plans interface {
	HasAI(ctx context.Context, userID int64, registreSpace string) (bool, error)
}

type scopeKey struct{}

type scope struct {
	space    string
	aiAccess bool
}

// Service resolves the space and the plan of every Ideas request.
type Service struct {
	db  *gorm.DB
	cfg Config
}

// NewService builds the service over the database and the Bulle wiring.
func NewService(db *gorm.DB, cfg Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func scopeOf(r *http.Request) scope {
	resolved, _ := r.Context().Value(scopeKey{}).(scope)
	return resolved
}

func (s *Service) resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolved, err := s.scope(r)
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, resolved)))
	})
}

func (s *Service) scope(r *http.Request) (scope, error) {
	identity, ok := kit.FromContext(r.Context())
	if !ok {
		return scope{}, errors.Unauthorized("missing auth")
	}
	userID := identity.UserID
	registre, err := s.registreSpace(r, userID)
	if err != nil {
		return scope{}, err
	}
	return s.withPlan(r.Context(), userID, registre)
}

func (s *Service) withPlan(ctx context.Context, userID int64, registre string) (scope, error) {
	if s.cfg.Plans == nil {
		return scope{space: registre, aiAccess: true}, nil
	}
	allowed, err := s.cfg.Plans.HasAI(ctx, userID, registre)
	if err != nil {
		return scope{}, errors.Internal("failed to read the plan", err)
	}
	return scope{space: registre, aiAccess: allowed}, nil
}

func (s *Service) registreSpace(r *http.Request, userID int64) (string, error) {
	raw := r.URL.Query().Get("space_id")
	if raw == "" {
		return "", nil
	}
	spaceID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || spaceID <= 0 {
		return "", errors.Invalid("invalid space_id")
	}
	var registre *string
	err = s.db.WithContext(r.Context()).Table("spaces").Select("spaces.registre_space_id").
		Joins("JOIN space_members sm ON sm.space_id = spaces.id").
		Where("spaces.id = ? AND sm.user_id = ?", spaceID, userID).Limit(1).Scan(&registre).Error
	if err != nil {
		return "", errors.Internal("failed to load the space", err)
	}
	return linked(registre)
}

func linked(registre *string) (string, error) {
	if registre == nil {
		return "", errors.NotFound("space not found")
	}
	if *registre == "" {
		return "", errors.Failed("this space is not linked to Registre yet, so Bulle cannot suggest ideas for it")
	}
	return *registre, nil
}
