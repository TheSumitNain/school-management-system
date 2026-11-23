package repository

import "testing"

func TestValidateStaffSubstitutionDateRange(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "valid single day", from: "2026-11-23", to: "2026-11-23"},
		{name: "valid range", from: "2026-11-01", to: "2026-11-30"},
		{name: "missing start", to: "2026-11-30", wantErr: true},
		{name: "missing end", from: "2026-11-01", wantErr: true},
		{name: "both missing", wantErr: true},
		{name: "invalid start format", from: "2026-1-01", to: "2026-11-30", wantErr: true},
		{name: "invalid calendar date", from: "2026-02-30", to: "2026-03-01", wantErr: true},
		{name: "year zero", from: "0000-01-01", to: "0000-01-02", wantErr: true},
		{name: "invalid end date", from: "2026-11-01", to: "not-a-date", wantErr: true},
		{name: "reversed range", from: "2026-11-30", to: "2026-11-01", wantErr: true},
		{name: "whitespace is not canonical", from: " 2026-11-01", to: "2026-11-30", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStaffSubstitutionDateRange(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
