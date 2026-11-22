package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTimetableWriteErrorIdentifiesSlotConflict(t *testing.T) {
	tests := []struct {
		name string
		constraint string
		want error
	}{
		{name: "teacher slot", constraint: "ux_timetable_entries_teacher_slot", want: ErrTimetableTeacherConflict},
		{name: "section slot", constraint: "ux_timetable_entries_section_slot", want: ErrTimetableSectionConflict},
		{name: "unknown unique constraint", constraint: "some_other_unique_constraint", want: ErrTimetableConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timetableWriteError(&pgconn.PgError{Code: "23505", ConstraintName: tt.constraint})
			if got != tt.want {
				t.Fatalf("timetableWriteError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimetableWriteErrorMapsExclusionConflict(t *testing.T) {
	got := timetableWriteError(&pgconn.PgError{Code: "23P01", ConstraintName: "period_overlap"})
	if got != ErrTimetableConflict {
		t.Fatalf("timetableWriteError() = %v, want %v", got, ErrTimetableConflict)
	}
}
