package services

import (
	"context"
	"errors"
	"testing"
)

type platformAdminPasswordTestStore struct {
	hash string
	getErr error
	changeErr error
	changeCalls int
	principalID int64
	expectedHash string
	newHash string
}

func (s *platformAdminPasswordTestStore) GetPasswordHashByPrincipalID(_ context.Context, principalID int64) (string, error) {
	s.principalID = principalID
	if s.getErr != nil { return "", s.getErr }
	return s.hash, nil
}

func (s *platformAdminPasswordTestStore) ChangePassword(_ context.Context, principalID int64, expectedHash, newHash string) error {
	s.changeCalls++
	s.principalID, s.expectedHash, s.newHash = principalID, expectedHash, newHash
	return s.changeErr
}

type platformAdminClearPasswordChecker struct{}
func (platformAdminClearPasswordChecker) IsCompromised(context.Context, string) (bool, error) { return false, nil }

type platformAdminCompromisedPasswordChecker struct{}
func (platformAdminCompromisedPasswordChecker) IsCompromised(context.Context, string) (bool, error) { return true, nil }

type platformAdminUnavailablePasswordChecker struct{}
func (platformAdminUnavailablePasswordChecker) IsCompromised(context.Context, string) (bool, error) {
	return false, errors.New("breach service unavailable")
}

func TestPlatformAdminPasswordServiceChangesPassword(t *testing.T) {
	oldHash, err := HashPassword("OldPlatform!Password123")
	if err != nil { t.Fatal(err) }
	store := &platformAdminPasswordTestStore{hash: oldHash}
	service, err := NewPlatformAdminPasswordService(store, platformAdminClearPasswordChecker{})
	if err != nil { t.Fatal(err) }
	if err := service.ChangePassword(context.Background(), 42, "OldPlatform!Password123", "NewPlatform!Password456"); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}
	if store.changeCalls != 1 || store.principalID != 42 || store.expectedHash != oldHash ||
		store.newHash == "" || store.newHash == oldHash || !CheckPassword("NewPlatform!Password456", store.newHash) {
		t.Fatalf("unexpected repository update: %+v", store)
	}
}

func TestPlatformAdminPasswordServiceRejectsInvalidCurrentPassword(t *testing.T) {
	oldHash, _ := HashPassword("OldPlatform!Password123")
	store := &platformAdminPasswordTestStore{hash: oldHash}
	service, _ := NewPlatformAdminPasswordService(store, platformAdminClearPasswordChecker{})
	err := service.ChangePassword(context.Background(), 42, "WrongPlatform!Password", "NewPlatform!Password456")
	if !errors.Is(err, ErrInvalidCredentials) || store.changeCalls != 0 {
		t.Fatalf("expected invalid credentials without update, err=%v calls=%d", err, store.changeCalls)
	}
}

func TestPlatformAdminPasswordServiceRejectsUnchangedPassword(t *testing.T) {
	oldHash, _ := HashPassword("OldPlatform!Password123")
	store := &platformAdminPasswordTestStore{hash: oldHash}
	service, _ := NewPlatformAdminPasswordService(store, platformAdminClearPasswordChecker{})
	err := service.ChangePassword(context.Background(), 42, "OldPlatform!Password123", "OldPlatform!Password123")
	if !errors.Is(err, ErrPlatformAdminPasswordUnchanged) || store.changeCalls != 0 {
		t.Fatalf("expected unchanged-password rejection, err=%v calls=%d", err, store.changeCalls)
	}
}

func TestPlatformAdminPasswordServiceRejectsCompromisedPassword(t *testing.T) {
	oldHash, _ := HashPassword("OldPlatform!Password123")
	store := &platformAdminPasswordTestStore{hash: oldHash}
	service, _ := NewPlatformAdminPasswordService(store, platformAdminCompromisedPasswordChecker{})
	err := service.ChangePassword(context.Background(), 42, "OldPlatform!Password123", "NewPlatform!Password456")
	if !errors.Is(err, ErrCompromisedPassword) || store.changeCalls != 0 {
		t.Fatalf("expected compromised-password rejection, err=%v calls=%d", err, store.changeCalls)
	}
}

func TestPlatformAdminPasswordServiceFailsClosedWhenBreachCheckUnavailable(t *testing.T) {
	oldHash, _ := HashPassword("OldPlatform!Password123")
	store := &platformAdminPasswordTestStore{hash: oldHash}
	service, _ := NewPlatformAdminPasswordService(store, platformAdminUnavailablePasswordChecker{})
	err := service.ChangePassword(context.Background(), 42, "OldPlatform!Password123", "NewPlatform!Password456")
	if !errors.Is(err, ErrPasswordBreachCheckUnavailable) || store.changeCalls != 0 {
		t.Fatalf("expected fail-closed breach-check error, err=%v calls=%d", err, store.changeCalls)
	}
}

func TestPlatformAdminPasswordServiceReturnsConcurrentChangeConflict(t *testing.T) {
	oldHash, _ := HashPassword("OldPlatform!Password123")
	store := &platformAdminPasswordTestStore{hash: oldHash, changeErr: ErrPlatformAdminPasswordConflict}
	service, _ := NewPlatformAdminPasswordService(store, platformAdminClearPasswordChecker{})
	err := service.ChangePassword(context.Background(), 42, "OldPlatform!Password123", "NewPlatform!Password456")
	if !errors.Is(err, ErrPlatformAdminPasswordConflict) {
		t.Fatalf("expected concurrent-change conflict, got %v", err)
	}
}
