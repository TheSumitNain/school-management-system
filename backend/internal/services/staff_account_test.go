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
	updateCalls int
	enableCalls int
	enablePasswordHash string
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

func (f *fakeStaffAccountRepository) UpdateStaffAccount(context.Context, int64, int64, int64, string) (*models.User, error) {
	f.updateCalls++
	return f.user, nil
}

func (f *fakeStaffAccountRepository) EnableStaffAccount(_ context.Context, _, _, _ int64, passwordHash string) error {
	f.enableCalls++
	f.enablePasswordHash = passwordHash
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


func TestStaffAccountServiceValidatesUsernameUpdate(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8, Username: "new.teacher"}}
	service := NewStaffAccountService(repo)
	user, err := service.Update(context.Background(), 31, 8, 27, " New.Teacher ")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if user != repo.user || repo.updateCalls != 1 {
		t.Fatalf("Update did not call repository exactly once: user=%+v calls=%d", user, repo.updateCalls)
	}
	if _, err := service.Update(context.Background(), 31, 8, 27, "bad name"); err != ErrInvalidStaffAccount {
		t.Fatalf("invalid username update error = %v, want ErrInvalidStaffAccount", err)
	}
}

func TestStaffAccountServiceHashesTemporaryPasswordBeforeReEnable(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo)
	if err := service.Enable(context.Background(), 31, 8, 27, "A-Strong-Temporary-Pass-123"); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if repo.enableCalls != 1 || repo.enablePasswordHash == "" ||
		repo.enablePasswordHash == "A-Strong-Temporary-Pass-123" ||
		!CheckPassword("A-Strong-Temporary-Pass-123", repo.enablePasswordHash) {
		t.Fatalf("Enable did not pass a valid password hash: calls=%d hash_set=%v",
			repo.enableCalls, repo.enablePasswordHash != "")
	}
	if err := service.Enable(context.Background(), 31, 8, 27, "short"); err != ErrInvalidPassword {
		t.Fatalf("invalid temporary password error = %v, want ErrInvalidPassword", err)
	}
}
