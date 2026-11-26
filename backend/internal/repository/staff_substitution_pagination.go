package repository

import (
	"context"
	"fmt"
)

type StaffSubstitutionPage struct {
	Items []StaffSubstitution
	Total int64
}

func (r *AcademicRepository) ListStaffSubstitutionsPage(ctx context.Context, actorID, schoolID int64, fromDate, toDate string, staffID int64, page, pageSize int) (StaffSubstitutionPage, error) {
	result := StaffSubstitutionPage{Items: []StaffSubstitution{}}
	if r == nil || r.db == nil || actorID <= 0 || schoolID <= 0 || staffID < 0 || page < 1 || pageSize < 1 || pageSize > 100 {
		return result, ErrInvalidStaffSubstitution
	}
	if err := staffSubstitutionAuthorize(ctx, r.db, actorID, schoolID, "staff.substitutions.read"); err != nil {
		return result, err
	}
	if err := validateStaffSubstitutionDateRange(fromDate, toDate); err != nil { return result, err }
	const filters = ` FROM staff_substitutions ss
		WHERE ss.school_id=$1 AND ss.cover_date BETWEEN $2::date AND $3::date
		AND ($4=0 OR ss.original_teacher_principal_id=$4 OR ss.substitute_teacher_principal_id=$4)`
	if err := r.db.QueryRow(ctx, "SELECT count(*)"+filters, schoolID, fromDate, toDate, staffID).Scan(&result.Total); err != nil {
		return result, fmt.Errorf("count staff substitutions: %w", err)
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `SELECT `+staffSubstitutionSelect+filters+`
		ORDER BY ss.cover_date,ss.period_number_snapshot,ss.id LIMIT $5 OFFSET $6`,
		schoolID, fromDate, toDate, staffID, pageSize, offset)
	if err != nil { return result, fmt.Errorf("list staff substitution page: %w", err) }
	defer rows.Close()
	for rows.Next() {
		var v StaffSubstitution
		if err := rows.Scan(&v.ID,&v.TimetableEntryID,&v.CoverDate,&v.OriginalTeacherPrincipalID,&v.OriginalTeacherName,
			&v.SubstituteTeacherPrincipalID,&v.SubstituteTeacherName,&v.ApprovedLeaveRequestID,&v.Reason,&v.Status,
			&v.SectionName,&v.ClassName,&v.SubjectName,&v.PeriodNumber,&v.PeriodName,&v.StartsAt,&v.EndsAt,
			&v.Room,&v.CreatedByPrincipalID,&v.CreatedAt,&v.CancelledByPrincipalID,&v.CancelledAt); err != nil {
			return result, fmt.Errorf("scan staff substitution page: %w", err)
		}
		result.Items = append(result.Items, v)
	}
	if err := rows.Err(); err != nil { return result, err }
	return result, nil
}
