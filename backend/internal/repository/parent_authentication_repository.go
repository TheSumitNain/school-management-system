package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ParentAuthenticationRepository resolves an existing Parent account from
// its verified mobile number. It never returns guardian or student profile data.
type ParentAuthenticationRepository struct {
	db *pgxpool.Pool
}

func NewParentAuthenticationRepository(db *pgxpool.Pool) *ParentAuthenticationRepository {
	return &ParentAuthenticationRepository{db: db}
}

// FindActiveParentByMobile returns a Parent only when the account and at least
// one complete official guardian-to-student link are currently active.
func (r *ParentAuthenticationRepository) FindActiveParentByMobile(
	ctx context.Context,
	mobileE164 string,
) (*models.AuthPrincipal, error) {
	if r == nil || r.db == nil || !parentGuardianE164Pattern.MatchString(strings.TrimSpace(mobileE164)) {
		return nil, models.ErrPrincipalNotAuthenticated
	}

	const query = `
SELECT ap.id, ap.principal_type, ap.status, ap.token_version,
       (pa.mobile_verified_at IS NOT NULL)
  FROM parent_accounts pa
  JOIN auth_principals ap ON ap.id = pa.principal_id
 WHERE pa.mobile_number = $1
   AND pa.mobile_verified_at IS NOT NULL
   AND ap.principal_type = 'parent'
   AND ap.status = 'active'
   AND ap.token_version > 0
   AND EXISTS (
       SELECT 1
         FROM parent_guardian_links pgl
         JOIN guardians g
           ON g.id = pgl.guardian_id
          AND g.school_id = pgl.school_id
         JOIN student_guardians sg
           ON sg.guardian_id = g.id
          AND sg.school_id = g.school_id
         JOIN students s
           ON s.id = sg.student_id
          AND s.school_id = sg.school_id
        WHERE pgl.principal_id = ap.id
          AND pgl.revoked_at IS NULL
          AND g.status = 'active'
          AND g.parent_portal_enabled_at IS NOT NULL
          AND sg.revoked_at IS NULL
          AND s.status = 'active'
   )`

	var principal models.AuthPrincipal
	var mobileVerified bool
	err := r.db.QueryRow(ctx, query, strings.TrimSpace(mobileE164)).Scan(
		&principal.ID, &principal.Type, &principal.Status,
		&principal.TokenVersion, &mobileVerified,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrPrincipalNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("find active Parent account by verified mobile: %w", err)
	}
	if principal.ID <= 0 || principal.Type != "parent" ||
		principal.Status != "active" || principal.TokenVersion <= 0 || !mobileVerified {
		return nil, models.ErrPrincipalNotAuthenticated
	}
	principal.MobileVerified = mobileVerified
	return &principal, nil
}
