package repository

import (
	"context"
	"errors"
	"testing"
)

func TestResetStaffPasswordRejectsNilRepository(t *testing.T) {
	ctx := context.Background()

	if err := (*UserRepository)(nil).ResetStaffPassword(ctx, 1, 1, 2, "password-hash"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil repository reset error = %v, want ErrInvalidUser", err)
	}
	if err := NewUserRepository(nil).ResetStaffPassword(ctx, 1, 1, 2, "password-hash"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil database reset error = %v, want ErrInvalidUser", err)
	}
}

func TestStaffPasswordRepositoryMethodsRejectNilDependencies(t *testing.T) {
	ctx := context.Background()
	var nilRepository *UserRepository
	emptyRepository := NewUserRepository(nil)

	if _, _, err := nilRepository.GetStaffPasswordCredential(ctx, 1); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil repository credential lookup error = %v, want ErrInvalidUser", err)
	}
	if _, _, err := emptyRepository.GetStaffPasswordCredential(ctx, 1); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil database credential lookup error = %v, want ErrInvalidUser", err)
	}
	if err := nilRepository.ChangeStaffPassword(ctx, 1, "old-hash", "new-hash"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil repository password change error = %v, want ErrInvalidUser", err)
	}
	if err := emptyRepository.ChangeStaffPassword(ctx, 1, "old-hash", "new-hash"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("nil database password change error = %v, want ErrInvalidUser", err)
	}
}
