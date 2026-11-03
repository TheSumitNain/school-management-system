package services

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"school-management-system/internal/repository"
)

var ErrInvalidSchoolOnboarding = errors.New("invalid school onboarding data")

var (
	schoolCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,49}$`)
	staffUsernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)
)

type SchoolOnboardingInput struct {
	Name                 string
	Code                 string
	AdminUsername        string
	AdminInitialPassword string
}

type SchoolOnboardingResult struct {
	SchoolID         int64  `json:"id"`
	SchoolName       string `json:"name"`
	SchoolCode       string `json:"code"`
	SchoolStatus     string `json:"status"`
	AdminPrincipalID int64  `json:"principal_id"`
	AdminUsername    string `json:"username"`
}

type schoolOnboardingRepository interface {
	CreateSchoolWithAdmin(ctx context.Context, actorPrincipalID int64, name, code, adminUsername, passwordHash string) (*repository.SchoolOnboardingResult, error)
}

type SchoolOnboardingService struct {
	repository schoolOnboardingRepository
}

func NewSchoolOnboardingService(repository schoolOnboardingRepository) *SchoolOnboardingService {
	return &SchoolOnboardingService{repository: repository}
}

// CreateSchoolWithAdmin atomically creates an active school, provisions its
// V13 system roles, creates the first staff principal and assigns School Admin.
func (s *SchoolOnboardingService) CreateSchoolWithAdmin(ctx context.Context, actorPrincipalID int64, input SchoolOnboardingInput) (*SchoolOnboardingResult, error) {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 {
		return nil, ErrInvalidSchoolOnboarding
	}

	name := strings.TrimSpace(input.Name)
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	username := strings.ToLower(strings.TrimSpace(input.AdminUsername))
	if name == "" || len(name) > 150 ||
		!schoolCodePattern.MatchString(code) ||
		!staffUsernamePattern.MatchString(username) {
		return nil, ErrInvalidSchoolOnboarding
	}

	passwordHash, err := HashPassword(input.AdminInitialPassword)
	if err != nil {
		return nil, err
	}

	created, err := s.repository.CreateSchoolWithAdmin(ctx, actorPrincipalID, name, code, username, passwordHash)
	if err != nil {
		return nil, err
	}
	return &SchoolOnboardingResult{
		SchoolID: created.SchoolID,
		SchoolName: created.SchoolName,
		SchoolCode: created.SchoolCode,
		SchoolStatus: created.SchoolStatus,
		AdminPrincipalID: created.AdminPrincipalID,
		AdminUsername: created.AdminUsername,
	}, nil
}
