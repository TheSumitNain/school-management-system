package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/repository"
)

type platformAdminCredentialStoreFake struct {
	record *repository.PlatformAdminCredential
	err    error
}

func (f platformAdminCredentialStoreFake) GetByEmail(context.Context, string) (*repository.PlatformAdminCredential, error) {
	return f.record, f.err
}

type platformAdminLoginSecurityFake struct {
	locked          bool
	failureEmail    string
	successEmail    string
	successID       int64
	staffFailures   int
	platformFailures int
}

func (f *platformAdminLoginSecurityFake) AcquireAttemptLock(context.Context, string) (func(), error) {
	return func() {}, nil
}

func (f *platformAdminLoginSecurityFake) IsLocked(context.Context, string) (bool, error) {
	return f.locked, nil
}

func (f *platformAdminLoginSecurityFake) RecordPlatformAdminFailure(_ context.Context, email string) error {
	f.platformFailures++
	f.failureEmail = email
	return nil
}

func (f *platformAdminLoginSecurityFake) RecordPlatformAdminSuccess(_ context.Context, email string, principalID int64) error {
	f.successEmail = email
	f.successID = principalID
	return nil
}

func TestPlatformAdminAuthenticateUsesPlatformAdminFailureAudit(t *testing.T) {
	security := &platformAdminLoginSecurityFake{}
	credentials := platformAdminCredentialStoreFake{err: repository.ErrPlatformAdminNotFound}
	service, err := NewPlatformAdminAuthService(credentials, security)
	if err != nil {
		t.Fatalf("create Platform Admin auth service: %v", err)
	}

	_, err = service.Authenticate(context.Background(), "  ADMIN@EXAMPLE.COM ", "incorrect")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Authenticate error = %v, want ErrInvalidCredentials", err)
	}
	if security.platformFailures != 1 {
		t.Fatalf("Platform Admin failure audit calls = %d, want 1", security.platformFailures)
	}
	if security.failureEmail != "admin@example.com" {
		t.Fatalf("failure audit email = %q, want normalized email", security.failureEmail)
	}
	if security.staffFailures != 0 {
		t.Fatalf("staff failure audit calls = %d, want 0", security.staffFailures)
	}
}

func TestPlatformAdminAuthenticateDoesNotRecordFailureWhenLocked(t *testing.T) {
	security := &platformAdminLoginSecurityFake{locked: true}
	service, err := NewPlatformAdminAuthService(
		platformAdminCredentialStoreFake{err: repository.ErrPlatformAdminNotFound},
		security,
	)
	if err != nil {
		t.Fatalf("create Platform Admin auth service: %v", err)
	}

	_, err = service.Authenticate(context.Background(), "admin@example.com", "incorrect")
	if !errors.Is(err, ErrLoginLocked) {
		t.Fatalf("Authenticate error = %v, want ErrLoginLocked", err)
	}
	if security.platformFailures != 0 {
		t.Fatalf("failure audit calls while locked = %d, want 0", security.platformFailures)
	}
}

