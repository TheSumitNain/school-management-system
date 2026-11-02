package services

import (
	"context"
	"errors"
	"fmt"
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

// staffPrincipalRepository resolves globally unique usernames to canonical
// principals after the V4 identity cutover.
type staffPrincipalRepository interface {
	GetStaffPrincipalByUsername(ctx context.Context, username string) (*models.AuthPrincipal, error)
}

// AuthenticateStaff authenticates the V4 globally unique staff username.
func (s *AuthService) AuthenticateStaff(ctx context.Context, username, password string) (*models.AuthPrincipal, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	repo, ok := s.users.(staffPrincipalRepository)
	if !ok {
		return nil, fmt.Errorf("staff principal repository is not configured")
	}

	principal, err := repo.GetStaffPrincipalByUsername(ctx, username)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrUserNotFound):
			return nil, ErrInvalidCredentials
		case errors.Is(err, repository.ErrStaffInactive):
			return nil, ErrUserInactive
		case errors.Is(err, repository.ErrSchoolInactive):
			return nil, ErrSchoolInactive
		default:
			return nil, err
		}
	}

	// Fetch the password hash through the canonical username lookup. This
	// keeps password material out of AuthPrincipal and response models.
	user, err := s.users.GetUserByUsername(ctx, principal.SchoolID, principal.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.PrincipalID != principal.ID {
		return nil, ErrInvalidCredentials
	}
	if user.Status != "active" || !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	return principal, nil
}
