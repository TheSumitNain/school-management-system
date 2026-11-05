package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

type fakeUserRepository struct {
	user       *models.User
	getErr     error
	createErr  error
	createdID  int64
	createHash  string
	principal   *models.AuthPrincipal
	principalErr error
}

func (f *fakeUserRepository) GetUserByUsername(
	_ context.Context,
	_ int64,
	_ string,
) (*models.User, error) {
	return f.user, f.getErr
}

func (f *fakeUserRepository) CreateUser(
	_ context.Context,
	schoolID int64,
	username string,
	passwordHash string,
) (*models.User, error) {
	f.createHash = passwordHash

	if f.createErr != nil {
		return nil, f.createErr
	}

	return &models.User{
		ID:           f.createdID,
		SchoolID:     schoolID,
		Username:     username,
		PasswordHash: passwordHash,
		Status:       "active",
	}, nil
}

func (f *fakeUserRepository) GetStaffPrincipalByUsername(
	_ context.Context,
	_ string,
) (*models.AuthPrincipal, error) {
	return f.principal, f.principalErr
}

type fakeSchoolRepository struct {
	school *repository.School
	err    error
}

func (f *fakeSchoolRepository) GetByID(
	_ context.Context,
	_ int64,
) (*repository.School, error) {
	return f.school, f.err
}

func TestAuthenticateSuccess(t *testing.T) {
	hash, err := HashPassword("School@Secure2026")
	if err != nil {
		t.Fatal(err)
	}

	users := &fakeUserRepository{
		user: &models.User{
			ID:           10,
			SchoolID:     1,
			Username:     "sumit",
			PasswordHash: hash,
			Status:       "active",
		},
	}

	schools := &fakeSchoolRepository{
		school: &repository.School{
			ID:     1,
			Status: "active",
		},
	}

	service := NewAuthService(users, schools)

	user, err := service.Authenticate(
		context.Background(),
		1,
		"sumit",
		"School@Secure2026",
	)
	if err != nil {
		t.Fatalf("expected successful authentication, got %v", err)
	}

	if user.ID != 10 {
		t.Fatalf("expected user ID 10, got %d", user.ID)
	}
}

func TestAuthenticateRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("School@Secure2026")
	if err != nil {
		t.Fatal(err)
	}

	users := &fakeUserRepository{
		user: &models.User{
			ID:           10,
			SchoolID:     1,
			Username:     "sumit",
			PasswordHash: hash,
			Status:       "active",
		},
	}

	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "active"},
	}

	service := NewAuthService(users, schools)

	_, err = service.Authenticate(
		context.Background(),
		1,
		"sumit",
		"Wrong@Secure2026",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateRejectsInactiveSchool(t *testing.T) {
	users := &fakeUserRepository{}
	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "suspended"},
	}

	service := NewAuthService(users, schools)

	_, err := service.Authenticate(
		context.Background(),
		1,
		"sumit",
		"School@Secure2026",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateRejectsDisabledUser(t *testing.T) {
	hash, err := HashPassword("School@Secure2026")
	if err != nil {
		t.Fatal(err)
	}

	users := &fakeUserRepository{
		user: &models.User{
			ID:           10,
			SchoolID:     1,
			Username:     "sumit",
			PasswordHash: hash,
			Status:       "disabled",
		},
	}

	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "active"},
	}

	service := NewAuthService(users, schools)

	_, err = service.Authenticate(
		context.Background(),
		1,
		"sumit",
		"School@Secure2026",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestCreateUserRejectsInactiveSchool(t *testing.T) {
	users := &fakeUserRepository{}
	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "pending_setup"},
	}

	service := NewAuthService(users, schools)

	_, err := service.CreateUser(
		context.Background(),
		1,
		"sumit",
		"School@Secure2026",
	)

	if !errors.Is(err, ErrSchoolInactive) {
		t.Fatalf("expected ErrSchoolInactive, got %v", err)
	}

	if users.createHash != "" {
		t.Fatal("user must not be created for an inactive school")
	}
}

func TestCreateUserStoresHashNotPlainPassword(t *testing.T) {
	users := &fakeUserRepository{createdID: 20}
	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "active"},
	}

	service := NewAuthService(users, schools)

	user, err := service.CreateUser(
		context.Background(),
		1,
		"sumit",
		"School@Secure2026",
	)
	if err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	if users.createHash == "" || users.createHash == "School@Secure2026" {
		t.Fatal("repository must receive a password hash, not plain text")
	}

	if user.PasswordHash != users.createHash {
		t.Fatal("returned user should contain the stored hash internally")
	}

	if !CheckPassword("School@Secure2026", user.PasswordHash) {
		t.Fatal("stored hash did not verify against original password")
	}
}

func TestAuthenticateRejectsUnknownUser(t *testing.T) {
	users := &fakeUserRepository{getErr: repository.ErrUserNotFound}
	schools := &fakeSchoolRepository{
		school: &repository.School{ID: 1, Status: "active"},
	}

	service := NewAuthService(users, schools)

	_, err := service.Authenticate(
		context.Background(),
		1,
		"unknown",
		"School@Secure2026",
	)

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateStaffFailsClosedWithoutLoginSecurity(t *testing.T) {
	users := &fakeUserRepository{}
	service := NewAuthService(users, &fakeSchoolRepository{})

	_, err := service.AuthenticateStaff(context.Background(), "teacher.one", "StrongPass123!")
	if !errors.Is(err, ErrLoginSecurityUnavailable) {
		t.Fatalf("missing login security error = %v, want ErrLoginSecurityUnavailable", err)
	}
	if users.principal != nil {
		t.Fatal("credential lookup must not proceed when login security controls are unavailable")
	}
}

func TestAuthenticateStaffRequiresMatchingPrincipalIdentity(t *testing.T) {
	hash, err := HashPassword("StrongPass123!")
	if err != nil { t.Fatal(err) }

	users := &fakeUserRepository{
		user: &models.User{ID: 100, PrincipalID: 999, SchoolID: 7, Username: "teacher.one", PasswordHash: hash, Status: "active"},
		principal: &models.AuthPrincipal{ID: 42, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 7, Username: "teacher.one"},
	}
	security := &fakeStaffLoginSecurity{}
	service := NewAuthService(users, &fakeSchoolRepository{}, security)

	_, err = service.AuthenticateStaff(context.Background(), "teacher.one", "StrongPass123!")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected mismatched principal identity to be rejected, got %v", err)
	}
}

func TestAuthenticateStaffAcceptsMatchingPrincipalIdentity(t *testing.T) {
	hash, err := HashPassword("StrongPass123!")
	if err != nil { t.Fatal(err) }

	users := &fakeUserRepository{
		user: &models.User{ID: 100, PrincipalID: 42, SchoolID: 7, Username: "teacher.one", PasswordHash: hash, Status: "active"},
		principal: &models.AuthPrincipal{ID: 42, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 7, Username: "teacher.one"},
	}
	security := &fakeStaffLoginSecurity{}
	service := NewAuthService(users, &fakeSchoolRepository{}, security)

	principal, err := service.AuthenticateStaff(context.Background(), "teacher.one", "StrongPass123!")
	if err != nil {
		t.Fatalf("expected matching principal authentication to succeed, got %v", err)
	}
	if principal.ID != 42 {
		t.Fatalf("authenticated principal ID = %d, want 42", principal.ID)
	}
}


type fakeStaffLoginSecurity struct {
	locked bool
	lockErr error
	failures int
	successes int
	lastPrincipalID int64
}

func (f *fakeStaffLoginSecurity) AcquireAttemptLock(context.Context, string) (func(), error) {
	return func() {}, nil
}
func (f *fakeStaffLoginSecurity) IsLocked(context.Context, string) (bool, error) {
	return f.locked, f.lockErr
}
func (f *fakeStaffLoginSecurity) RecordFailure(context.Context, string) error {
	f.failures++
	return nil
}
func (f *fakeStaffLoginSecurity) RecordSuccess(_ context.Context, _ string, principalID int64) error {
	f.successes++
	f.lastPrincipalID = principalID
	return nil
}

func TestAuthenticateStaffRejectsLockedUsernameBeforeCredentialLookup(t *testing.T) {
	security := &fakeStaffLoginSecurity{locked: true}
	users := &fakeUserRepository{}
	service := NewAuthService(users, &fakeSchoolRepository{}, security)

	_, err := service.AuthenticateStaff(context.Background(), "teacher.one", "WrongPassword123!")
	if !errors.Is(err, ErrLoginLocked) {
		t.Fatalf("expected ErrLoginLocked, got %v", err)
	}
	if users.principal != nil {
		t.Fatal("locked login must not proceed to principal lookup")
	}
}

func TestAuthenticateStaffRecordsInvalidCredentialAndSuccessfulLogin(t *testing.T) {
	hash, err := HashPassword("StrongPass123!")
	if err != nil { t.Fatal(err) }
	users := &fakeUserRepository{
		user: &models.User{ID: 100, PrincipalID: 42, SchoolID: 7, Username: "teacher.one", PasswordHash: hash, Status: "active"},
		principal: &models.AuthPrincipal{ID: 42, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 7, Username: "teacher.one"},
	}
	security := &fakeStaffLoginSecurity{}
	service := NewAuthService(users, &fakeSchoolRepository{}, security)

	_, err = service.AuthenticateStaff(context.Background(), "teacher.one", "WrongPassword123!")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if security.failures != 1 {
		t.Fatalf("failure events = %d, want 1", security.failures)
	}

	principal, err := service.AuthenticateStaff(context.Background(), "teacher.one", "StrongPass123!")
	if err != nil {
		t.Fatalf("expected valid login, got %v", err)
	}
	if principal.ID != 42 || security.successes != 1 || security.lastPrincipalID != 42 {
		t.Fatalf("successful login was not recorded correctly: principal=%v security=%+v", principal, security)
	}
}
