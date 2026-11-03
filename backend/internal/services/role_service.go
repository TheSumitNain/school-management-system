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
