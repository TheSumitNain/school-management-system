package services

import (
	"context"
	"errors"
	"fmt"

	"school-management-system/internal/repository"
)

var (
	ErrPlatformAdminPasswordUnchanged = errors.New("new password must differ from current password")
	ErrPlatformAdminPasswordConflict = repository.ErrPlatformAdminPasswordConflict
	ErrPlatformAdminPasswordNotFound = repository.ErrPlatformAdminNotFound
)

type platformAdminPasswordStore interface {
	GetPasswordHashByPrincipalID(context.Context, int64) (string, error)
	ChangePassword(context.Context, int64, string, string) error
}

type PlatformAdminPasswordService struct {
	repository platformAdminPasswordStore
	breachChecker PasswordBreachChecker
}

func NewPlatformAdminPasswordService(store platformAdminPasswordStore, checkers ...PasswordBreachChecker) (*PlatformAdminPasswordService, error) {
	if store == nil {
		return nil, fmt.Errorf("Platform Admin password repository is required")
	}
	checker := defaultPasswordBreachChecker
	if len(checkers) > 0 {
		checker = checkers[0]
	}
	return &PlatformAdminPasswordService{repository: store, breachChecker: checker}, nil
}

// ChangePassword verifies the current credential, applies the common and
// breached-password checks, and delegates an atomic session-invalidating update.
func (s *PlatformAdminPasswordService) ChangePassword(ctx context.Context, principalID int64, currentPassword, newPassword string) error {
	if s == nil || s.repository == nil || principalID <= 0 || currentPassword == "" || newPassword == "" {
		return ErrInvalidCredentials
	}
	currentHash, err := s.repository.GetPasswordHashByPrincipalID(ctx, principalID)
	if err != nil {
		if errors.Is(err, repository.ErrPlatformAdminNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if !CheckPassword(currentPassword, currentHash) {
		return ErrInvalidCredentials
	}
	if CheckPassword(newPassword, currentHash) {
		return ErrPlatformAdminPasswordUnchanged
	}
	newHash, err := hashPasswordWithChecker(ctx, newPassword, s.breachChecker)
	if err != nil {
		return err
	}
	return s.repository.ChangePassword(ctx, principalID, currentHash, newHash)
}
