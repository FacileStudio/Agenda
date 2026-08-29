package spaces

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service implements space persistence, membership and access control.
type Service struct {
	orm *gorm.DB
}

// NewService builds a space service over the given database.
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
	if roleRank(req.Role) > roleRank(role) {
		return errors.Forbidden("you cannot grant a rank above your own")
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
	if err := s.guardTargetRank(ctx, spaceID, target.ID, role); err != nil {
		return err
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
	if roleRank(target.Role) > roleRank(role) {
		return errors.Forbidden("you cannot remove a member ranked above you")
	}
	if err := s.guardLastOwner(ctx, spaceID, target.Role); err != nil {
		return err
	}
	return s.orm.WithContext(ctx).Delete(&target).Error
}

// UpdateMemberRole rewrites one membership's role.
//
// It reads the target's current role before writing, for two reasons. The
// owner count question belongs to the role being taken away and not the one
// being granted, and this was the one role write that never asked it: the sole
// owner could demote themselves and get nil back, into a state no route
// through the API undoes. And a blind UPDATE matched no row for somebody who
// had never joined, then reported success.
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
	if roleRank(req.Role) > roleRank(role) {
		return errors.Forbidden("you cannot grant a rank above your own")
	}
	var target schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, targetUserID).First(&target).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("member not found")
		}
		return errors.Internal("failed to find member", err)
	}
	if req.Role != target.Role {
		if err := s.guardLastOwner(ctx, spaceID, target.Role); err != nil {
			return err
		}
	}
	return s.orm.WithContext(ctx).Model(&target).Update("role", req.Role).Error
}

func (s *Service) LeaveSpace(ctx context.Context, userID int64, spaceID int64) error {
	if _, _, err := s.loadWithAccess(ctx, userID, spaceID); err != nil {
		return err
	}
	var member schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		return errors.Internal("failed to find membership", err)
	}
	if err := s.guardLastOwner(ctx, spaceID, member.Role); err != nil {
		return err
	}
	return s.orm.WithContext(ctx).Delete(&member).Error
}

// guardTargetRank refuses to let an actor rewrite a membership ranked above
// their own, and says nothing about somebody who is not a member yet.
//
// AddMember writes through an upsert, so aiming it at an existing member
// rewrites their role. Checking only the role being granted catches an admin
// minting an owner and misses the same call pointed downward: AddMember with
// the owner's address and role "member" demoted them. That is the worse half
// of the pair, because deleting a space and changing its roles are both
// owner-only, so a space left with no owner cannot be repaired from inside it.
// RemoveMember already asked this question of its target; AddMember did not.
func (s *Service) guardTargetRank(ctx context.Context, spaceID int64, targetUserID int64, actorRole string) error {
	var existing schemas.SpaceMember
	err := s.orm.WithContext(ctx).
		Where("space_id = ? AND user_id = ?", spaceID, targetUserID).
		First(&existing).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return errors.Internal("failed to read the membership", err)
	}
	if roleRank(existing.Role) > roleRank(actorRole) {
		return errors.Forbidden("you cannot change a member ranked above you")
	}
	return nil
}

// guardLastOwner refuses to take the final owner out of a space.
//
// Deleting a space and changing its roles are both owner-only, so a space with
// no owner cannot be administered and cannot be repaired from inside it. The
// rule is the owner count and not the role: refusing every owner would trap
// anyone who was ever made one, and refusing none is what let an owner strand
// the space by removing the only one. Leaving and being removed are the same
// departure, so they ask the same question here rather than each keeping a
// count of their own.
func (s *Service) guardLastOwner(ctx context.Context, spaceID int64, role string) error {
	if role != "owner" {
		return nil
	}
	var owners int64
	err := s.orm.WithContext(ctx).
		Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND role = 'owner'", spaceID).
		Count(&owners).Error
	if err != nil {
		return errors.Internal("failed to count the owners", err)
	}
	if owners <= 1 {
		return errors.Invalid("the space would be left without an owner — transfer ownership or delete the space")
	}
	return nil
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

// roleRank orders the three roles so that "nobody may act above their own
// rank" is one comparison rather than a table of pairs.
//
// AddMember checked only that the actor was an owner or an admin and then
// wrote whatever role the body named, and its write is an upsert — so an admin
// could name an existing member and the role "owner" and promote them, walking
// around UpdateMemberRole, which is owner-only for exactly that reason. The
// actor's own rank was never in the comparison.
//
// An unrecognised role ranks below every real one, so a value that slipped
// past normalizeRole could never clear a check by being unknown.
func roleRank(role string) int {
	switch role {
	case "owner":
		return 3
	case "admin":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
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
