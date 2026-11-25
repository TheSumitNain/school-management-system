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
