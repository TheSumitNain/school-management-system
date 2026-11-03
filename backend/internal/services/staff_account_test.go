package services

import (
	"context"
	"testing"
	"strings"

	"school-management-system/internal/models"
)

type fakeStaffAccountRepository struct {
	user *models.User
	actorID int64
	schoolID int64
	username string
	passwordHash string
	calls int
}

func (f *fakeStaffAccountRepository) CreateStaffAccount(_ context.Context, actorID, schoolID int64, username, passwordHash string) (*models.User, error) {
	f.calls++
	f.actorID, f.schoolID, f.username, f.passwordHash = actorID, schoolID, username, passwordHash
	return f.user, nil
}

func (f *fakeStaffAccountRepository) ListStaffAccounts(context.Context, int64, int64) ([]models.User, error) {
	return []models.User{}, nil
}

func (f *fakeStaffAccountRepository) DisableStaffAccount(context.Context, int64, int64, int64) error {
	return nil
}

func TestStaffAccountServiceCreatesHashedTemporaryAccount(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one",
		Status: "active", MustChangePassword: true,
	}}
	service := NewStaffAccountService(repo)
	user, err := service.Create(context.Background(), 31, 8, " Teacher.One ", "A-Strong-Initial-Pass-123")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if user != repo.user || repo.calls != 1 || repo.actorID != 31 || repo.schoolID != 8 ||
		repo.username != "teacher.one" || repo.passwordHash == "" || repo.passwordHash == "A-Strong-Initial-Pass-123" {
		t.Fatalf("unexpected staff account service behavior: user=%+v repo=%+v", user, repo)
	}
	if CheckPassword("A-Strong-Initial-Pass-123", repo.passwordHash) != true {
		t.Fatal("repository did not receive a valid hash for the initial password")
	}
}

func TestStaffAccountServiceRejectsInvalidUsernameOrPassword(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo)
	for _, username := range []string{"ab", "bad name", "-teacher", "नमस्ते", strings.Repeat("a", 33)} {
		if _, err := service.Create(context.Background(), 31, 8, username, "A-Strong-Initial-Pass-123"); err == nil {
			t.Errorf("Create(%q) expected username validation error", username)
		}
	}
	if _, err := service.Create(context.Background(), 31, 8, "teacher.one", "short"); err != ErrInvalidPassword {
		t.Fatalf("invalid password error = %v, want ErrInvalidPassword", err)
	}
	if repo.calls != 0 {
		t.Fatalf("invalid requests reached repository %d times", repo.calls)
	}
}
