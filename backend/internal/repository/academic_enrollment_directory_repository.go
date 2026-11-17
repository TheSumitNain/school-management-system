package repository

import (
	"context"
	"fmt"
)

// enrollmentDirectoryVisibleCTE mirrors the school-admin and teacher roster
// visibility predicate used by complete per-student enrollment history.
const enrollmentDirectoryVisibleCTE = `WITH visible_enrollments AS (
	SELECT e.id,e.school_id,e.student_id,e.academic_year_id,e.class_id,e.section_id,e.roll_number,
		e.class_name_snapshot,e.section_name_snapshot,e.admission_number,e.admission_date,e.status,e.enrolled_at,e.ended_at,e.created_at,e.updated_at,st.full_name,
		EXISTS (SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
		 WHERE pr.principal_id=$3 AND pr.revoked_at IS NULL AND r.school_id=e.school_id
		 AND r.scope_type='school' AND r.principal_type='staff' AND r.role_key='school_admin' AND r.is_system=TRUE),
		EXISTS (SELECT 1 FROM class_incharge_assignments cia
		 WHERE cia.school_id=e.school_id AND cia.academic_year_id=e.academic_year_id
		 AND cia.section_id=e.section_id AND cia.teacher_principal_id=$3 AND cia.revoked_at IS NULL),
		st.status
	FROM student_enrollments e
	JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
	WHERE e.school_id=$1 AND ($2=0 OR e.academic_year_id=$2)
	  AND (
		EXISTS (
			SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
			WHERE pr.principal_id=$3 AND pr.revoked_at IS NULL AND r.school_id=$1
			  AND r.scope_type='school' AND r.principal_type='staff' AND r.role_key='school_admin' AND r.is_system=TRUE
		)
		OR (
			EXISTS (
				SELECT 1 FROM principal_roles tpr JOIN roles tr ON tr.id=tpr.role_id
				WHERE tpr.principal_id=$3 AND tpr.revoked_at IS NULL AND tr.school_id=e.school_id
				  AND tr.scope_type='school' AND tr.principal_type='staff' AND tr.role_key='teacher' AND tr.is_system=TRUE
			)
			AND (
				EXISTS (
					SELECT 1 FROM class_incharge_assignments cia
					WHERE cia.school_id=e.school_id AND cia.academic_year_id=e.academic_year_id
					  AND cia.section_id=e.section_id AND cia.teacher_principal_id=$3 AND cia.revoked_at IS NULL
				)
				OR EXISTS (
					SELECT 1 FROM teacher_subject_assignments tsa
					WHERE tsa.school_id=e.school_id AND tsa.academic_year_id=e.academic_year_id
					  AND tsa.section_id=e.section_id AND tsa.teacher_principal_id=$3 AND tsa.revoked_at IS NULL
				)
			)
		)
	  )
)
`

// ListStudentEnrollmentDirectory returns a bounded page and its visible total.
// ListStudentEnrollments remains the complete, per-student history operation.
func (r *AcademicRepository) ListStudentEnrollmentDirectory(
	ctx context.Context, actorID, schoolID, yearID int64, limit, offset int,
) ([]StudentEnrollment, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, ErrAcademicAccessDenied
	}
	if yearID < 0 || limit < 1 || limit > 100 || offset < 0 {
		return nil, 0, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin student enrollment directory read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_basic"); err != nil {
		return nil, 0, err
	}

	var total int64
	if err := tx.QueryRow(ctx, enrollmentDirectoryVisibleCTE+`SELECT COUNT(*) FROM visible_enrollments`, schoolID, yearID, actorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count visible student enrollments: %w", err)
	}
	rows, err := tx.Query(ctx, enrollmentDirectoryVisibleCTE+`SELECT * FROM visible_enrollments
		ORDER BY enrolled_at DESC,id DESC LIMIT $4 OFFSET $5`, schoolID, yearID, actorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list student enrollment directory: %w", err)
	}
	items := make([]StudentEnrollment, 0, limit)
	for rows.Next() {
		item, scanErr := scanStudentEnrollmentWithName(rows)
		if scanErr != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("scan student enrollment directory row: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, fmt.Errorf("iterate student enrollment directory: %w", err)
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrollment_directory.read','student_enrollment_collection',$3,'success',
		       jsonb_build_object('section','student_enrollment_directory','result_count',$4::integer,
		                          'total_count',$5::bigint,'limit',$6::integer,'offset',$7::integer,
		                          'academic_year_filtered',$8::boolean))
	`, actorID, schoolID, fmt.Sprint(schoolID), len(items), total, limit, offset, yearID > 0); err != nil {
		return nil, 0, fmt.Errorf("audit student enrollment directory read: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit student enrollment directory audit: %w", err)
	}
	return items, total, nil
}
