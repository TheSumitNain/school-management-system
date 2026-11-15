package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestAcademicWriteErrorIdentifiesAdmissionNumberConstraint(t *testing.T) {
	err := &pgconn.PgError{
		Code: "23505",
		ConstraintName: "ux_student_enrollments_school_year_admission_number",
	}
	if got := academicWriteError(err); !errors.Is(got, ErrAcademicAdmissionNumberConflict) {
		t.Fatalf("academicWriteError() = %v, want admission-number conflict", got)
	}
}

func TestAcademicWriteErrorKeepsOtherUniqueConflictsGeneric(t *testing.T) {
	err := &pgconn.PgError{Code: "23505", ConstraintName: "student_enrollments_student_year_key"}
	if got := academicWriteError(err); !errors.Is(got, ErrAcademicConflict) {
		t.Fatalf("academicWriteError() = %v, want generic academic conflict", got)
	}
	if errors.Is(academicWriteError(err), ErrAcademicAdmissionNumberConflict) {
		t.Fatal("non-admission unique constraint was incorrectly classified as an admission-number conflict")
	}
}
