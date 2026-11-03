package spaces

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service reads spaces and their members for the people in them.
//
// It writes nothing. Registre owns a space's name, members and roles, and
// Directory applies its snapshots; an edit made here would be undone by the
// next one, so the client links to Registre instead.
type Service struct {
	orm *gorm.DB
}

// NewService builds a space service over the given database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// ListSpaces returns every space the user belongs to, with their role.
func (s *Service) ListSpaces(ctx context.Context, userID int64) ([]SpaceResponse, error) {
	var rows []SpaceResponse
	err := s.orm.WithContext(ctx).Raw(`
		SELECT s.id, s.name, s.description, coalesce(s.registre_space_id, '') AS registre_space_id,
		       sm.role, s.created_at, s.updated_at
		FROM spaces s
		JOIN space_members sm ON sm.space_id = s.id
		WHERE sm.user_id = ?
		ORDER BY s.name
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list spaces", err)
	}
	if rows == nil {
		rows = []SpaceResponse{}
	}
	return rows, nil
}

// GetSpace returns one space the user belongs to.
func (s *Service) GetSpace(ctx context.Context, userID int64, spaceID int64) (*SpaceResponse, error) {
	space, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return nil, err
	}
	response := &SpaceResponse{
		ID:          space.ID,
		Name:        space.Name,
		Description: space.Description,
		Role:        role,
		CreatedAt:   space.CreatedAt,
		UpdatedAt:   space.UpdatedAt,
	}
	if space.RegistreSpaceID != nil {
		response.RegistreSpaceID = *space.RegistreSpaceID
	}
	return response, nil
}

// ListMembers returns a space's members, for a caller who is one of them.
func (s *Service) ListMembers(ctx context.Context, userID int64, spaceID int64) ([]MemberResponse, error) {
	if _, _, err := s.loadWithAccess(ctx, userID, spaceID); err != nil {
		return nil, err
	}
	var rows []MemberResponse
	err := s.orm.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.email, u.name, `+schemas.AvatarSelectExpr+` AS avatar_url, sm.role, sm.joined_at
		FROM space_members sm
		JOIN users u ON u.id = sm.user_id
		WHERE sm.space_id = ?
		ORDER BY sm.joined_at
	`, spaceID).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list members", err)
	}
	if rows == nil {
		rows = []MemberResponse{}
	}
	return rows, nil
}

func (s *Service) loadWithAccess(ctx context.Context, userID int64, spaceID int64) (*schemas.Space, string, error) {
	var space schemas.Space
	if err := s.orm.WithContext(ctx).First(&space, spaceID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.NotFound("space not found")
		}
		return nil, "", errors.Internal("failed to load space", err)
	}
	var member schemas.SpaceMember
	err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", errors.Forbidden("access denied")
	}
	if err != nil {
		return nil, "", errors.Internal("failed to check access", err)
	}
	return &space, member.Role, nil
}

// agendaRole maps a Registre role onto Agenda's, which are the same three
// words. Anything else becomes an ordinary member rather than a rank nobody
// granted.
func agendaRole(role string) string {
	switch role {
	case directory.RoleOwner, directory.RoleAdmin:
		return role
	default:
		return directory.RoleMember
	}
}
