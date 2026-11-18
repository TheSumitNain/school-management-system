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
	ptr := func(value string) *string { return &value }
	tests := []struct {
		name    string
		input   *string
		want    string
		wantNil bool
		wantErr bool
	}{
		{name: "valid email", input: ptr("parent@example.com"), want: "parent@example.com"},
		{name: "trims surrounding whitespace", input: ptr(" parent@example.com "), want: "parent@example.com"},
		{name: "empty becomes nil", input: ptr("  "), wantNil: true},
		{name: "nil remains nil", input: nil, wantNil: true},
		{name: "missing domain", input: ptr("parent@"), wantErr: true},
		{name: "missing local part", input: ptr("@example.com"), wantErr: true},
		{name: "display name is not accepted as an email field", input: ptr("Parent <parent@example.com>"), wantErr: true},
		{name: "multiple addresses rejected", input: ptr("parent@example.com, other@example.com"), wantErr: true},
		{name: "over maximum length", input: ptr(strings.Repeat("a", 245) + "@example.com"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeGuardianEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeGuardianEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidGuardianEmail) {
					t.Fatalf("normalizeGuardianEmail() error = %v, want ErrInvalidGuardianEmail", err)
				}
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

func TestNormalizeGuardianRelationshipAllowsOnlyApprovedOptions(t *testing.T) {
	tests := []struct { input, want string; wantErr bool }{
		{"Father","Father",false}, {" mother ","Mother",false},
		{"GRANDFATHER","Grandfather",false}, {"Grandmother","Grandmother",false},
		{"Brother","Brother",false}, {"Sister","Sister",false},
		{"Uncle","Uncle",false}, {"Aunt","Aunt",false},
		{"Step-father","",true}, {"Step-mother","",true},
		{"Legal Guardian","",true}, {"Foster Parent","",true},
		{"Other","",true}, {"","",true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeGuardianRelationship(tt.input)
			if (err != nil) != tt.wantErr { t.Fatalf("error = %v, wantErr %v", err, tt.wantErr) }
			if !tt.wantErr && got != tt.want { t.Fatalf("got %q, want %q", got, tt.want) }
			if tt.wantErr && !errors.Is(err, ErrInvalidGuardianInput) { t.Fatalf("error = %v, want ErrInvalidGuardianInput", err) }
		})
	}
}
