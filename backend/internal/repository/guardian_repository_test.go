package repository

import (
	"errors"
	"strings"
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

func TestNormalizeGuardianEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   *string
		want    string
		wantNil bool
		wantErr bool
	}{
		{name: "valid email", input: stringPointer("parent@example.com"), want: "parent@example.com"},
		{name: "trims surrounding whitespace", input: stringPointer(" parent@example.com "), want: "parent@example.com"},
		{name: "empty becomes nil", input: stringPointer("  "), wantNil: true},
		{name: "nil remains nil", input: nil, wantNil: true},
		{name: "missing domain", input: stringPointer("parent@"), wantErr: true},
		{name: "missing local part", input: stringPointer("@example.com"), wantErr: true},
		{name: "display name is not accepted as an email field", input: stringPointer("Parent <parent@example.com>"), wantErr: true},
		{name: "multiple addresses rejected", input: stringPointer("parent@example.com, other@example.com"), wantErr: true},
		{name: "over maximum length", input: stringPointer(strings.Repeat("a", 245) + "@example.com"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeGuardianEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeGuardianEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("normalizeGuardianEmail() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != tt.want {
				t.Fatalf("normalizeGuardianEmail() = %v, want %q", got, tt.want)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
