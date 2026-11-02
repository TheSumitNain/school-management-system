package repository

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var parentGuardianE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// ParentGuardianMatch is the minimum official guardian identity needed by the
// parent-registration flow. It deliberately excludes student names and other
// student profile data.
type ParentGuardianMatch struct {
	GuardianID         int64
	SchoolID           int64
	GuardianName       string
	ActiveStudentCount int64
}

// ParentGuardianMatchRepository resolves an exact verified mobile number to
// active, portal-enabled guardian records with at least one active official
// student relationship. A shared mobile may legitimately match records in
// multiple schools; callers must retain every returned match.
type ParentGuardianMatchRepository struct {
	db *pgxpool.Pool
}

func NewParentGuardianMatchRepository(db *pgxpool.Pool) *ParentGuardianMatchRepository {
	return &ParentGuardianMatchRepository{db: db}
}

func (r *ParentGuardianMatchRepository) FindPortalEligibleGuardiansByMobile(
	ctx context.Context, mobileE164 string,
) ([]ParentGuardianMatch, error) {
	if r == nil || r.db == nil || !parentGuardianE164Pattern.MatchString(mobileE164) {
		return nil, fmt.Errorf("invalid parent guardian matching request")
	}
	return findPortalEligibleGuardiansByMobile(ctx, r.db, mobileE164)
}

type parentGuardianMatchQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func findPortalEligibleGuardiansByMobile(
	ctx context.Context, q parentGuardianMatchQuerier, mobileE164 string,
) ([]ParentGuardianMatch, error) {
	rows, err := q.Query(ctx, `
		SELECT g.id, g.school_id, g.full_name,
		       COUNT(DISTINCT s.id)::BIGINT AS active_student_count
		  FROM guardians g
		  JOIN student_guardians sg
		    ON sg.guardian_id = g.id
		   AND sg.school_id = g.school_id
		   AND sg.revoked_at IS NULL
		  JOIN students s
		    ON s.id = sg.student_id
		   AND s.school_id = sg.school_id
		   AND s.status = 'active'
		 WHERE g.mobile_number_e164 = $1
		   AND g.status = 'active'
		   AND g.parent_portal_enabled_at IS NOT NULL
		   AND NOT EXISTS (
		       SELECT 1 FROM parent_guardian_links pgl
		        WHERE pgl.guardian_id = g.id AND pgl.revoked_at IS NULL
		   )
		 GROUP BY g.id, g.school_id, g.full_name
		 ORDER BY g.school_id, g.id
	`, mobileE164)
	if err != nil {
		return nil, fmt.Errorf("query portal-eligible guardians: %w", err)
	}
	defer rows.Close()

	matches := make([]ParentGuardianMatch, 0)
	for rows.Next() {
		var match ParentGuardianMatch
		if err := rows.Scan(
			&match.GuardianID,
			&match.SchoolID,
			&match.GuardianName,
			&match.ActiveStudentCount,
		); err != nil {
			return nil, fmt.Errorf("scan portal-eligible guardian: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portal-eligible guardians: %w", err)
	}
	return matches, nil
}
