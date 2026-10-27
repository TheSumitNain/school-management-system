package services

import (
	"context"
	"errors"
	"strings"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

var (
	ErrSchoolInactive      = errors.New("school is not active")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUserInactive        = errors.New("user is not active")
	ErrInvalidRegistration = errors.New("invalid registration data")
)

type authUserRepository interface {
	GetUserByUsername(
		ctx context.Context,
		schoolID int64,
		username string,
	) (*models.User, error)

	CreateUser(
		ctx context.Context,
		schoolID int64,
		username string,
		passwordHash string,
	) (*models.User, error)
}

type authSchoolRepository interface {
	GetByID(
		ctx context.Context,
		schoolID int64,
	) (*repository.School, error)
}

type AuthService struct {
	users   authUserRepository
	schools authSchoolRepository
}

func NewAuthService(
	users authUserRepository,
	schools authSchoolRepository,
) *AuthService {
	return &AuthService{
		users:   users,
		schools: schools,
	}
}

// CreateUser creates a user for an existing active school.
// This is an internal service operation, not a public registration endpoint.
func (s *AuthService) CreateUser(
	ctx context.Context,
	schoolID int64,
	username string,
	password string,
) (*models.User, error) {
	username = strings.TrimSpace(username)

	if schoolID <= 0 ||
		username == "" ||
		len(username) > 100 {
		return nil, ErrInvalidRegistration
	}

	school, err := s.schools.GetByID(ctx, schoolID)
	if err != nil {
		return nil, err
	}

	if school.Status != "active" {
		return nil, ErrSchoolInactive
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	return s.users.CreateUser(
		ctx,
		schoolID,
		username,
		passwordHash,
	)
}

// Authenticate verifies the school, user status, and password.
// JWT generation will be added in a later step.
func (s *AuthService) Authenticate(
	ctx context.Context,
	schoolID int64,
	username string,
	password string,
) (*models.User, error) {
	username = strings.TrimSpace(username)

	if schoolID <= 0 || username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	school, err := s.schools.GetByID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, repository.ErrSchoolNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if school.Status != "active" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetUserByUsername(
		ctx,
		schoolID,
		username,
	)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != "active" {
		return nil, ErrInvalidCredentials
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
