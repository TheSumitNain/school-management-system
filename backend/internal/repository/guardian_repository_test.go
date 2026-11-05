package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestGuardianWriteErrorClassifiesActiveParentLinkAsConflict(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "guardian contact mobile change",
			message: "guardian mobile cannot change while an active Parent link exists; revoke the link first",
		},
		{
			name:    "guardian portal disable",
			message: "revoke active Parent links with an authorized actor before disabling guardian portal eligibility",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guardianWriteError(&pgconn.PgError{Code: "P0001", Message: tt.message})
			if !errors.Is(err, ErrGuardianConflict) {
				t.Fatalf("guardianWriteError() = %v, want ErrGuardianConflict", err)
			}
		})
	}
}

func TestGuardianWriteErrorKeepsUnrelatedTriggerFailuresAsInvalidInput(t *testing.T) {
	err := guardianWriteError(&pgconn.PgError{
		Code:    "P0001",
		Message: "guardian portal eligibility requires an active guardian with an E.164 mobile",
	})
	if !errors.Is(err, ErrInvalidGuardianInput) {
		t.Fatalf("guardianWriteError() = %v, want ErrInvalidGuardianInput", err)
	}
}
