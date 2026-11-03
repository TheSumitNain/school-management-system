package services

import (
	"context"
	"errors"
	"testing"
)

type fakeStaffPasswordRepository struct {
	userID int64
	passwordHash string
	getErr error
	changeErr error
	changeCalls int
	expectedHash string
	newHash string
}

func (f *fakeStaffPasswordRepository) GetStaffPasswordCredential(_ context.Context, principalID int64) (int64, string, error) {
	if f.getErr != nil {
		return 0, "", f.getErr
	}
	return f.userID, f.passwordHash, nil
}

func (f *fakeStaffPasswordRepository) ChangeStaffPassword(_ context.Context, _ int64, expectedHash, newHash string) error {
	f.changeCalls++
	f.expectedHash = expectedHash
	f.newHash = newHash
	return f.changeErr
}

func TestStaffPasswordServiceChangesPassword(t *testing.T) {
	oldHash, err := HashPassword("OldStrong!Password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeStaffPasswordRepository{userID: 7, passwordHash: oldHash}
	service, err := NewStaffPasswordService(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(context.Background(), 11, "OldStrong!Password123", "NewStrong!Password456"); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	if repo.changeCalls != 1 || repo.expectedHash != oldHash || repo.newHash == "" || repo.newHash == oldHash {
		t.Fatalf("unexpected repository password update: %+v", repo)
	}
	if !CheckPassword("NewStrong!Password456", repo.newHash) {
		t.Fatal("repository did not receive a hash for the new password")
	}
}

func TestStaffPasswordServiceRejectsWrongCurrentPassword(t *testing.T) {
	oldHash, err := HashPassword("OldStrong!Password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeStaffPasswordRepository{userID: 7, passwordHash: oldHash}
	service, _ := NewStaffPasswordService(repo)
	err = service.ChangePassword(context.Background(), 11, "WrongStrong!Password", "NewStrong!Password456")
	if !errors.Is(err, ErrInvalidCredentials) || repo.changeCalls != 0 {
		t.Fatalf("expected invalid credentials without update, err=%v calls=%d", err, repo.changeCalls)
	}
}

func TestStaffPasswordServiceRejectsWeakNewPassword(t *testing.T) {
	oldHash, err := HashPassword("OldStrong!Password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeStaffPasswordRepository{userID: 7, passwordHash: oldHash}
	service, _ := NewStaffPasswordService(repo)
	err = service.ChangePassword(context.Background(), 11, "OldStrong!Password123", "short")
	if !errors.Is(err, ErrInvalidPassword) || repo.changeCalls != 0 {
		t.Fatalf("expected password-policy rejection without update, err=%v calls=%d", err, repo.changeCalls)
	}
}

func TestStaffPasswordServiceRejectsUnchangedPassword(t *testing.T) {
	oldHash, err := HashPassword("OldStrong!Password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeStaffPasswordRepository{userID: 7, passwordHash: oldHash}
	service, _ := NewStaffPasswordService(repo)
	err = service.ChangePassword(context.Background(), 11, "OldStrong!Password123", "OldStrong!Password123")
	if !errors.Is(err, ErrPasswordUnchanged) || repo.changeCalls != 0 {
		t.Fatalf("expected unchanged-password rejection, err=%v calls=%d", err, repo.changeCalls)
	}
}

func TestStaffPasswordServiceReturnsConcurrentChangeConflict(t *testing.T) {
	oldHash, err := HashPassword("OldStrong!Password123")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeStaffPasswordRepository{userID: 7, passwordHash: oldHash, changeErr: ErrPasswordChangeConflict}
	service, _ := NewStaffPasswordService(repo)
	err = service.ChangePassword(context.Background(), 11, "OldStrong!Password123", "NewStrong!Password456")
	if !errors.Is(err, ErrPasswordChangeConflict) {
		t.Fatalf("expected concurrent change conflict, got %v", err)
	}
}
