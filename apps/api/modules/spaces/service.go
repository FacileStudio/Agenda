package spaces

import (
	"context"
	stderrors "errors"
	"strings"

	"api/internal/errors"
	"api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) ListSpaces(ctx context.Context, userID int64) ([]SpaceResponse, error) {
	var rows []SpaceResponse
	err := s.orm.WithContext(ctx).Raw(`
		SELECT s.id, s.name, s.description, sm.role, s.created_at, s.updated_at
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

func (s *Service) GetSpace(ctx context.Context, userID int64, spaceID int64) (*SpaceResponse, error) {
	space, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return nil, err
	}
	return &SpaceResponse{
		ID:          space.ID,
		Name:        space.Name,
		Description: space.Description,
		Role:        role,
		CreatedAt:   space.CreatedAt,
		UpdatedAt:   space.UpdatedAt,
	}, nil
}

func (s *Service) CreateSpace(ctx context.Context, userID int64, req *CreateSpaceRequest) (*SpaceResponse, error) {
	if err := validateSpaceName(req.Name); err != nil {
		return nil, err
	}
	space := &schemas.Space{
		Name:        req.Name,
		Description: req.Description,
	}
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(space).Error; err != nil {
			return err
		}
		member := &schemas.SpaceMember{
			SpaceID: space.ID,
			UserID:  userID,
			Role:    "owner",
		}
		return tx.Create(member).Error
	})
	if err != nil {
		return nil, errors.Internal("failed to create space", err)
	}
	return &SpaceResponse{
		ID:          space.ID,
		Name:        space.Name,
		Description: space.Description,
		Role:        "owner",
		CreatedAt:   space.CreatedAt,
		UpdatedAt:   space.UpdatedAt,
	}, nil
}

func (s *Service) UpdateSpace(ctx context.Context, userID int64, spaceID int64, req *UpdateSpaceRequest) (*SpaceResponse, error) {
	if err := validateSpaceName(req.Name); err != nil {
		return nil, err
	}
	space, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return nil, err
	}
	if role != "owner" && role != "admin" {
		return nil, errors.Forbidden("insufficient permissions")
	}
	space.Name = req.Name
	space.Description = req.Description
	if err := s.orm.WithContext(ctx).Save(space).Error; err != nil {
		return nil, errors.Internal("failed to update space", err)
	}
	return &SpaceResponse{
		ID:          space.ID,
		Name:        space.Name,
		Description: space.Description,
		Role:        role,
		CreatedAt:   space.CreatedAt,
		UpdatedAt:   space.UpdatedAt,
	}, nil
}

func (s *Service) DeleteSpace(ctx context.Context, userID int64, spaceID int64) error {
	_, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return errors.Forbidden("only the owner can delete a space")
	}
	return s.orm.WithContext(ctx).Delete(&schemas.Space{}, spaceID).Error
}

func (s *Service) ListMembers(ctx context.Context, userID int64, spaceID int64) ([]MemberResponse, error) {
	if _, _, err := s.loadWithAccess(ctx, userID, spaceID); err != nil {
		return nil, err
	}
	var rows []MemberResponse
	err := s.orm.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.email, u.name, u.avatar_url, sm.role, sm.joined_at
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

func (s *Service) AddMember(ctx context.Context, userID int64, spaceID int64, req *AddMemberRequest) error {
	_, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return errors.Forbidden("insufficient permissions")
	}
	req.Role = normalizeRole(req.Role)
	if req.Role == "" {
		return errors.Invalid("role must be owner, admin, or member")
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return errors.Invalid("email is required")
	}
	var target schemas.User
	if err := s.orm.WithContext(ctx).Where("LOWER(email) = ?", req.Email).First(&target).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to find user", err)
	}
	if target.ID == userID {
		return errors.Invalid("you are already a member")
	}
	member := schemas.SpaceMember{
		SpaceID: spaceID,
		UserID:  target.ID,
		Role:    req.Role,
	}
	result := s.orm.WithContext(ctx).
		Where("space_id = ? AND user_id = ?", spaceID, target.ID).
		Assign(schemas.SpaceMember{Role: req.Role}).
		FirstOrCreate(&member)
	if result.Error != nil {
		return errors.Internal("failed to add member", result.Error)
	}
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, userID int64, spaceID int64, targetUserID int64) error {
	_, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return err
	}
	if role != "owner" && role != "admin" {
		return errors.Forbidden("insufficient permissions")
	}
	var target schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, targetUserID).First(&target).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("member not found")
		}
		return errors.Internal("failed to find member", err)
	}
	if target.Role == "owner" && role != "owner" {
		return errors.Forbidden("cannot remove the owner")
	}
	return s.orm.WithContext(ctx).Delete(&target).Error
}

func (s *Service) UpdateMemberRole(ctx context.Context, userID int64, spaceID int64, targetUserID int64, req *UpdateRoleRequest) error {
	_, role, err := s.loadWithAccess(ctx, userID, spaceID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return errors.Forbidden("only the owner can change roles")
	}
	req.Role = normalizeRole(req.Role)
	if req.Role == "" {
		return errors.Invalid("role must be owner, admin, or member")
	}
	return s.orm.WithContext(ctx).
		Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND user_id = ?", spaceID, targetUserID).
		Update("role", req.Role).Error
}

func (s *Service) LeaveSpace(ctx context.Context, userID int64, spaceID int64) error {
	if _, _, err := s.loadWithAccess(ctx, userID, spaceID); err != nil {
		return err
	}
	var member schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		return errors.Internal("failed to find membership", err)
	}
	if member.Role == "owner" {
		var count int64
		s.orm.WithContext(ctx).Model(&schemas.SpaceMember{}).Where("space_id = ? AND role = 'owner'", spaceID).Count(&count)
		if count <= 1 {
			return errors.Invalid("the sole owner cannot leave — transfer ownership or delete the space")
		}
	}
	return s.orm.WithContext(ctx).Delete(&member).Error
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

func validateSpaceName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.Invalid("name is required")
	}
	if len(name) > 255 {
		return errors.Invalid("name must be 255 characters or fewer")
	}
	return nil
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner":
		return "owner"
	case "admin":
		return "admin"
	case "member":
		return "member"
	default:
		return ""
	}
}
