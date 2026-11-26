package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type StaffSubstitutionPage struct {
	Items []StaffSubstitution
	Total int64
}

func (r *AcademicRepository) ListStaffSubstitutionsPage(ctx context.Context, actorID, schoolID int64, fromDate, toDate string, staffID int64, page, pageSize int) (StaffSubstitutionPage, error) {
	result := StaffSubstitutionPage{Items: []StaffSubstitution{}}
	if r == nil || r.db == nil || actorID <= 0 || schoolID <= 0 || staffID < 0 || page < 1 || pageSize < 1 || pageSize > 100 || page-1 > int(^uint(0)>>1)/pageSize {
		return result, ErrInvalidStaffSubstitution
	}
	offset := (page - 1) * pageSize
	if offset > 1000000 {
		return result, ErrInvalidStaffSubstitution
	}
	if err := validateStaffSubstitutionDateRange(fromDate, toDate); err != nil {
		return result, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin staff substitution page snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, callerOwnsTransaction := r.db.(pgx.Tx); !callerOwnsTransaction {
		if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
			return result, fmt.Errorf("set staff substitution page snapshot: %w", err)
		}
	}
	if err := staffSubstitutionAuthorize(ctx, tx, actorID, schoolID, "staff.substitutions.read"); err != nil {
		return result, err
	}

	const filters = ` FROM staff_substitutions ss
		WHERE ss.school_id=$1 AND ss.cover_date BETWEEN $2::date AND $3::date
		AND ($4=0 OR ss.original_teacher_principal_id=$4 OR ss.substitute_teacher_principal_id=$4)`
	if err := tx.QueryRow(ctx, "SELECT count(*)"+filters, schoolID, fromDate, toDate, staffID).Scan(&result.Total); err != nil {
		return result, fmt.Errorf("count staff substitutions: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT `+staffSubstitutionSelect+filters+`
		ORDER BY ss.cover_date,ss.period_number_snapshot,ss.id LIMIT $5 OFFSET $6`,
		schoolID, fromDate, toDate, staffID, pageSize, offset)
	if err != nil {
		return result, fmt.Errorf("list staff substitution page: %w", err)
	}
	for rows.Next() {
		var v StaffSubstitution
		if err := rows.Scan(&v.ID, &v.TimetableEntryID, &v.CoverDate, &v.OriginalTeacherPrincipalID, &v.OriginalTeacherName,
			&v.SubstituteTeacherPrincipalID, &v.SubstituteTeacherName, &v.ApprovedLeaveRequestID, &v.Reason, &v.Status,
			&v.SectionName, &v.ClassName, &v.SubjectName, &v.PeriodNumber, &v.PeriodName, &v.StartsAt, &v.EndsAt,
			&v.Room, &v.CreatedByPrincipalID, &v.CreatedAt, &v.CancelledByPrincipalID, &v.CancelledAt); err != nil {
			rows.Close()
			return result, fmt.Errorf("scan staff substitution page: %w", err)
		}
		result.Items = append(result.Items, v)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit staff substitution page snapshot: %w", err)
	}
	return result, nil
}
