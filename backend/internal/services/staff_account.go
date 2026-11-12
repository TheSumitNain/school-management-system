package services

import (
	"context"
	"errors"
	"strings"
	"regexp"
	"time"

	"school-management-system/internal/models"
)

var ErrInvalidStaffAccount = errors.New("invalid staff account details")

type staffAccountRepository interface {
	CreateStaffAccount(ctx context.Context, actorPrincipalID, schoolID int64, username, passwordHash string) (*models.User, error)
	ListStaffAccounts(ctx context.Context, actorPrincipalID, schoolID int64) ([]models.User, error)
	DisableStaffAccount(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) error
	UpdateStaffAccount(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, username string) (*models.User, error)
	EnableStaffAccount(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, newPasswordHash string) error
}

type staffProfileCreator interface {
	CreateStaffAccountWithProfile(ctx context.Context, actorPrincipalID, schoolID int64, username, passwordHash string, profile models.StaffProfile) (*models.User, error)
}
type staffProfileUpdater interface {
    UpdateStaffProfile(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, profile models.StaffProfile) (*models.User, error)
}

type StaffAccountService struct {
	repository staffAccountRepository
	breachChecker PasswordBreachChecker
}

func NewStaffAccountService(repository staffAccountRepository, checkers ...PasswordBreachChecker) *StaffAccountService {
	checker := defaultPasswordBreachChecker
	if len(checkers) > 0 { checker = checkers[0] }
	return &StaffAccountService{repository: repository, breachChecker: checker}
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
	passwordHash, err := hashPasswordWithChecker(ctx, initialPassword, s.breachChecker)
	if err != nil {
		return nil, err
	}
	return s.repository.CreateStaffAccount(ctx, actorPrincipalID, schoolID, username, passwordHash)
}

var staffEmployeeCodePattern = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$")
var staffMobilePattern = regexp.MustCompile("^\\\\+?[0-9]{10,15}$")

// CreateWithProfile creates a login account and employee profile in one repository transaction.
func (s *StaffAccountService) CreateWithProfile(ctx context.Context, actorPrincipalID, schoolID int64, username, initialPassword string, profile models.StaffProfile) (*models.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	profile.EmployeeCode = strings.TrimSpace(profile.EmployeeCode)
	profile.FullName = strings.TrimSpace(profile.FullName)
	profile.Designation = strings.TrimSpace(profile.Designation)
	profile.MobileNumber = strings.TrimSpace(profile.MobileNumber)
	profile.Email = strings.TrimSpace(profile.Email)
	profile.EmploymentStatus = "active"
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || !validStaffUsername(username) ||
		!staffEmployeeCodePattern.MatchString(profile.EmployeeCode) || profile.FullName == "" || len(profile.FullName) > 160 ||
		profile.Designation == "" || len(profile.Designation) > 100 || !staffMobilePattern.MatchString(profile.MobileNumber) ||
		(profile.Email != "" && (!strings.Contains(profile.Email, "@") || len(profile.Email) > 254)) ||
		(profile.JoiningDate != nil && profile.JoiningDate.After(time.Now().AddDate(1, 0, 0))) {
		return nil, ErrInvalidStaffAccount
	}
	creator, ok := s.repository.(staffProfileCreator)
	if !ok { return nil, ErrInvalidStaffAccount }
	passwordHash, err := hashPasswordWithChecker(ctx, initialPassword, s.breachChecker)
	if err != nil { return nil, err }
	return creator.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID, username, passwordHash, profile)
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

func (s *StaffAccountService) Disable(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || actorPrincipalID == targetPrincipalID {
		return ErrInvalidStaffAccount
	}
	return s.repository.DisableStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID)
}



// UpdateProfile validates and persists an employee profile without re-enabling
// a disabled login. An inactive profile remains blocked by the database guard.
func (s *StaffAccountService) UpdateProfile(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, profile models.StaffProfile) (*models.User, error) {
    profile.EmployeeCode = strings.TrimSpace(profile.EmployeeCode)
    profile.FullName = strings.TrimSpace(profile.FullName)
    profile.Designation = strings.TrimSpace(profile.Designation)
    profile.MobileNumber = strings.TrimSpace(profile.MobileNumber)
    profile.Email = strings.TrimSpace(profile.Email)
    if profile.EmploymentStatus == "" { profile.EmploymentStatus = "active" }
    if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 ||
       !staffEmployeeCodePattern.MatchString(profile.EmployeeCode) || profile.FullName == "" || len(profile.FullName) > 160 ||
       profile.Designation == "" || len(profile.Designation) > 100 || !staffMobilePattern.MatchString(profile.MobileNumber) ||
       (profile.Email != "" && (!strings.Contains(profile.Email, "@") || len(profile.Email) > 254)) ||
       (profile.EmploymentStatus != "active" && profile.EmploymentStatus != "inactive") {
        return nil, ErrInvalidStaffAccount
    }
    updater, ok := s.repository.(staffProfileUpdater)
    if !ok { return nil, ErrInvalidStaffAccount }
    return updater.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, targetPrincipalID, profile)
}

// Update changes the username of an active same-school staff account.
func (s *StaffAccountService) Update(
	ctx context.Context,
	actorPrincipalID, schoolID, targetPrincipalID int64,
	username string,
) (*models.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || !validStaffUsername(username) {
		return nil, ErrInvalidStaffAccount
	}
	return s.repository.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID, username)
}

// Enable resets the disabled account's credential to a temporary password.
// The account must change it after the next successful login.
func (s *StaffAccountService) Enable(
	ctx context.Context,
	actorPrincipalID, schoolID, targetPrincipalID int64,
	temporaryPassword string,
) error {
	if s == nil || s.repository == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || actorPrincipalID == targetPrincipalID {
		return ErrInvalidStaffAccount
	}
	passwordHash, err := hashPasswordWithChecker(ctx, temporaryPassword, s.breachChecker)
	if err != nil {
		return err
	}
	return s.repository.EnableStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID, passwordHash)
}
