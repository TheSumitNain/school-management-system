package repository

import (
	"errors"
	"testing"
	"time"
)

func TestValidateEnrollmentDateRangeRequiresAdmissionOnOrBeforeEnrollment(t *testing.T) {
	yearStart := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(2027, time.March, 31, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		admissionDate time.Time
		enrolledAt    time.Time
		wantErr       bool
	}{
		{
			name:          "admission and enrollment on same day",
			admissionDate: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
			enrolledAt:    time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "admission before enrollment",
			admissionDate: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC),
			enrolledAt:    time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "admission after enrollment",
			admissionDate: time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC),
			enrolledAt:    time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC),
			wantErr:       true,
		},
		{
			name:          "admission before academic year",
			admissionDate: time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
			enrolledAt:    time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnrollmentDateRange(tt.admissionDate, tt.enrolledAt, yearStart, yearEnd)
			if tt.wantErr && !errors.Is(err, ErrInvalidAcademicInput) {
				t.Fatalf("validateEnrollmentDateRange() error = %v, want ErrInvalidAcademicInput", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateEnrollmentDateRange() error = %v, want nil", err)
			}
		})
	}
}
