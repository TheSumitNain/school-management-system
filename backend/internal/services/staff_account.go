package services

import (
	"context"
	"errors"
	"strings"

	"school-management-system/internal/models"
)

var ErrInvalidStaffAccount = errors.New("invalid staff account details")

type staffAccountRepository interface {
	CreateStaffAccount(ctx context.Context, actorPrincipalID, schoolID int64, username, passwordHash string) (*models.User, error)
	ListStaffAccounts(ctx context.Context, actorPrincipalID, schoolID int64) ([]models.User, error)
}

type StaffAccountService struct {
	repository staffAccountRepository
}

func NewStaffAccountService(repository staffAccountRepository) *StaffAccountService {
	return &StaffAccountService{repository: repository}
}

func (s *StaffAccountService) Create(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	username, initialPassword string,
) (*models.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || !validStaffUsername(username) {
		return nil, ErrInvalidStaffAccount
	}
	passwordHash, err := HashPassword(initialPassword)
	if err != nil {
		return nil, err
	}
	return s.repository.CreateStaffAccount(ctx, actorPrincipalID, schoolID, username, passwordHash)
}

func validStaffUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for i, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			(i > 0 && (r == '.' || r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return true
}

func (s *StaffAccountService) List(ctx context.Context, actorPrincipalID, schoolID int64) ([]models.User, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrInvalidStaffAccount
	}
	return s.repository.ListStaffAccounts(ctx, actorPrincipalID, schoolID)
}
