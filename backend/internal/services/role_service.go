package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"school-management-system/internal/repository"
)

var ErrInvalidCustomRole = errors.New("invalid custom school role")

type roleRepository interface {
	ListSchoolRoles(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolRole, error)
	ListSchoolPermissions(ctx context.Context) ([]repository.SchoolPermission, error)
	CreateCustomRole(ctx context.Context, actorPrincipalID, schoolID int64, roleKey, name string) (*repository.SchoolRole, error)
	GrantSchoolPermissionDelegation(ctx context.Context, actorPrincipalID, schoolID int64, permissionKey string, canAssignToRoles, canOverrideUsers bool) (*repository.SchoolPermissionDelegation, error)
	RevokeSchoolPermissionDelegation(ctx context.Context, actorPrincipalID, delegationID int64) error
	ListSchoolPermissionDelegations(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolPermissionDelegation, error)
	GrantCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID, delegationID int64, permissionKey string) error
	RevokeCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID int64, permissionKey string) error
	ListStaffPermissionOverrides(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffPermissionOverride, error)
	SetStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, delegationID int64, permissionKey, effect, reason string) error
	DeleteStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, permissionKey string) error
	ListStaffRoleAssignments(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffRoleAssignment, error)
	AssignStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error
	RevokeStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error
}

type RoleService struct {
	repository roleRepository
}

func NewRoleService(repository roleRepository) *RoleService {
	return &RoleService{repository: repository}
}

func (s *RoleService) ListSchoolRoles(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolRole, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.ListSchoolRoles(ctx, actorPrincipalID, schoolID)
}

func (s *RoleService) ListSchoolPermissions(ctx context.Context) ([]repository.SchoolPermission, error) {
	if s == nil || s.repository == nil {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.ListSchoolPermissions(ctx)
}

func (s *RoleService) CreateCustomRole(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	name string,
) (*repository.SchoolRole, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrInvalidCustomRole
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 120 {
		return nil, ErrInvalidCustomRole
	}
	roleKey := customRoleKey(name)
	if roleKey == "" {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.CreateCustomRole(ctx, actorPrincipalID, schoolID, roleKey, name)
}

func customRoleKey(name string) string {
	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			separator = false
		default:
			if b.Len() > 0 && !separator {
				b.WriteByte('_')
				separator = true
			}
		}
	}
	key := strings.Trim(b.String(), "_")
	if len(key) > 68 {
		key = strings.TrimRight(key[:68], "_")
	}
	if key == "" {
		return ""
	}
	return "custom_" + key
}

func (s *RoleService) GrantSchoolPermissionDelegation(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	permissionKey string,
	canAssignToRoles, canOverrideUsers bool,
) (*repository.SchoolPermissionDelegation, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		strings.TrimSpace(permissionKey) == "" || (!canAssignToRoles && !canOverrideUsers) {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.GrantSchoolPermissionDelegation(
		ctx, actorPrincipalID, schoolID, strings.TrimSpace(permissionKey), canAssignToRoles, canOverrideUsers,
	)
}

func (s *RoleService) RevokeSchoolPermissionDelegation(
	ctx context.Context,
	actorPrincipalID, delegationID int64,
) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || delegationID <= 0 {
		return ErrInvalidCustomRole
	}
	return s.repository.RevokeSchoolPermissionDelegation(ctx, actorPrincipalID, delegationID)
}

func (s *RoleService) ListSchoolPermissionDelegations(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolPermissionDelegation, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.ListSchoolPermissionDelegations(ctx, actorPrincipalID, schoolID)
}

func (s *RoleService) GrantCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID, delegationID int64, permissionKey string) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || roleID <= 0 || delegationID <= 0 || strings.TrimSpace(permissionKey) == "" {
		return ErrInvalidCustomRole
	}
	return s.repository.GrantCustomRolePermission(ctx, actorPrincipalID, schoolID, roleID, delegationID, strings.TrimSpace(permissionKey))
}

func (s *RoleService) RevokeCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID int64, permissionKey string) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || roleID <= 0 || strings.TrimSpace(permissionKey) == "" {
		return ErrInvalidCustomRole
	}
	return s.repository.RevokeCustomRolePermission(ctx, actorPrincipalID, schoolID, roleID, strings.TrimSpace(permissionKey))
}

func (s *RoleService) ListStaffPermissionOverrides(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffPermissionOverride, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.ListStaffPermissionOverrides(ctx, actorPrincipalID, schoolID, targetPrincipalID)
}

func (s *RoleService) SetStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, delegationID int64, permissionKey, effect, reason string) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 || delegationID <= 0 {
		return ErrInvalidCustomRole
	}
	permissionKey, effect, reason = strings.TrimSpace(permissionKey), strings.TrimSpace(effect), strings.TrimSpace(reason)
	if permissionKey == "" || (effect != "allow" && effect != "deny") || reason == "" || len(reason) > 1000 {
		return ErrInvalidCustomRole
	}
	return s.repository.SetStaffPermissionOverride(ctx, actorPrincipalID, schoolID, targetPrincipalID, delegationID, permissionKey, effect, reason)
}

func (s *RoleService) DeleteStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, permissionKey string) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 || strings.TrimSpace(permissionKey) == "" {
		return ErrInvalidCustomRole
	}
	return s.repository.DeleteStaffPermissionOverride(ctx, actorPrincipalID, schoolID, targetPrincipalID, strings.TrimSpace(permissionKey))
}

func (s *RoleService) ListStaffRoleAssignments(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffRoleAssignment, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 {
		return nil, ErrInvalidCustomRole
	}
	return s.repository.ListStaffRoleAssignments(ctx, actorPrincipalID, schoolID, targetPrincipalID)
}

func (s *RoleService) AssignStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 || roleID <= 0 {
		return ErrInvalidCustomRole
	}
	return s.repository.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, roleID)
}

func (s *RoleService) RevokeStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 || roleID <= 0 {
		return ErrInvalidCustomRole
	}
	return s.repository.RevokeStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, roleID)
}
