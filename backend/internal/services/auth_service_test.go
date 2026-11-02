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

func TestAuthenticateStaffRequiresMatchingPrincipalIdentity(t *testing.T) {
	hash, err := HashPassword("StrongPass123!")
	if err != nil { t.Fatal(err) }

	users := &fakeUserRepository{
		user: &models.User{ID: 100, PrincipalID: 999, SchoolID: 7, Username: "teacher.one", PasswordHash: hash, Status: "active"},
		principal: &models.AuthPrincipal{ID: 42, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 7, Username: "teacher.one"},
	}
	service := NewAuthService(users, &fakeSchoolRepository{})

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
	service := NewAuthService(users, &fakeSchoolRepository{})

	principal, err := service.AuthenticateStaff(context.Background(), "teacher.one", "StrongPass123!")
	if err != nil {
		t.Fatalf("expected matching principal authentication to succeed, got %v", err)
	}
	if principal.ID != 42 {
		t.Fatalf("authenticated principal ID = %d, want 42", principal.ID)
	}
}
