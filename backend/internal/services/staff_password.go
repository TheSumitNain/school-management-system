package services

import (
	"context"
	"errors"
	"fmt"

	"school-management-system/internal/repository"
)

var (
	ErrPasswordUnchanged = errors.New("new password must differ from the current password")
	ErrPasswordChangeConflict = repository.ErrStaffPasswordConflict
)

type staffPasswordRepository interface {
	GetStaffPasswordCredential(ctx context.Context, principalID int64) (int64, string, error)
	ChangeStaffPassword(ctx context.Context, principalID int64, expectedPasswordHash, newPasswordHash string) error
	ResetStaffPassword(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, newPasswordHash string) error
}

type StaffPasswordService struct {
	repository staffPasswordRepository
}

func NewStaffPasswordService(repository staffPasswordRepository) (*StaffPasswordService, error) {
	if repository == nil {
		return nil, fmt.Errorf("staff password repository is required")
	}
	return &StaffPasswordService{repository: repository}, nil
}

// ChangePassword verifies the current credential and applies the password
// policy before asking the repository to atomically rotate the credential and
// invalidate all sessions. The repository compares the old hash again to
// prevent a stale verification from overwriting a concurrent password change.
func (s *StaffPasswordService) ChangePassword(
	ctx context.Context,
	principalID int64,
	currentPassword, newPassword string,
) error {
	if s == nil || s.repository == nil || principalID <= 0 ||
		currentPassword == "" || newPassword == "" {
		return ErrInvalidCredentials
	}
	_, currentHash, err := s.repository.GetStaffPasswordCredential(ctx, principalID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if !CheckPassword(currentPassword, currentHash) {
		return ErrInvalidCredentials
	}
	if CheckPassword(newPassword, currentHash) {
		return ErrPasswordUnchanged
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repository.ChangeStaffPassword(ctx, principalID, currentHash, newHash)
}


// ResetPassword sets a same-school staff member's password to a new temporary
// credential. The repository independently verifies the actor's School Admin
// grant, school boundary, target status, token-version rotation and audit trail.
func (s *StaffPasswordService) ResetPassword(
	ctx context.Context,
	actorPrincipalID, schoolID, targetPrincipalID int64,
	newPassword string,
) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 ||
		schoolID <= 0 || targetPrincipalID <= 0 || actorPrincipalID == targetPrincipalID {
		return ErrInvalidRegistration
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repository.ResetStaffPassword(ctx, actorPrincipalID, schoolID, targetPrincipalID, newHash)
}
