package repository

import (
	"context"
	"errors"
	"fmt"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PrincipalAuthorizationRepository loads live principal identity and a
// conservative effective-permission set for each protected request.
type PrincipalAuthorizationRepository struct {
	db *pgxpool.Pool
}

func NewPrincipalAuthorizationRepository(db *pgxpool.Pool) *PrincipalAuthorizationRepository {
	return &PrincipalAuthorizationRepository{db: db}
}

func (r *PrincipalAuthorizationRepository) LoadPrincipalForAuthentication(
	ctx context.Context,
	principalID int64,
) (*models.AuthPrincipal, error) {
	if principalID <= 0 {
		return nil, models.ErrPrincipalNotAuthenticated
	}

	const query = `
SELECT ap.id, ap.principal_type, ap.status, ap.token_version,
       COALESCE(u.school_id, 0), COALESCE(u.username, ''),
       COALESCE(u.must_change_password, FALSE),
       COALESCE(u.status, ''), COALESCE(s.status, ''),
       (pa.principal_id IS NOT NULL),
       (parent.mobile_verified_at IS NOT NULL),
       EXISTS (
         SELECT 1
         FROM parent_guardian_links pgl
         JOIN guardians g ON g.id = pgl.guardian_id AND g.school_id = pgl.school_id
         JOIN student_guardians sg ON sg.guardian_id = g.id AND sg.school_id = g.school_id
         JOIN students st ON st.id = sg.student_id AND st.school_id = sg.school_id
         WHERE pgl.principal_id = ap.id
           AND pgl.revoked_at IS NULL
           AND sg.revoked_at IS NULL
           AND g.status = 'active'
           AND st.status = 'active'
           AND g.parent_portal_enabled_at IS NOT NULL
       ),
       CASE
         WHEN ap.principal_type='platform_admin' OR
              (ap.principal_type='staff' AND EXISTS (
                SELECT 1 FROM principal_roles mpr JOIN roles mr ON mr.id=mpr.role_id
                WHERE mpr.principal_id=ap.id AND mpr.revoked_at IS NULL
                  AND mr.is_system=TRUE AND mr.role_key='school_admin'
                  AND mr.principal_type='staff' AND mr.scope_type='school'
                  AND mr.school_id=u.school_id
              ))
         THEN EXISTS (SELECT 1 FROM principal_mfa_credentials mc
                      WHERE mc.principal_id=ap.id AND mc.method='totp' AND mc.verified_at IS NOT NULL)
         ELSE TRUE
       END,
       (u.principal_id IS NOT NULL)::int +
       (pa.principal_id IS NOT NULL)::int +
       (parent.principal_id IS NOT NULL)::int
FROM auth_principals ap
LEFT JOIN users u ON u.principal_id = ap.id
LEFT JOIN schools s ON s.id = u.school_id
LEFT JOIN platform_admin_accounts pa ON pa.principal_id = ap.id
LEFT JOIN parent_accounts parent ON parent.principal_id = ap.id
WHERE ap.id = $1`

	var p models.AuthPrincipal
	var staffStatus, schoolStatus string
	var mustChangePassword bool
	var hasPlatformAccount, mobileVerified, hasEligibleParentLink, mfaVerified bool
	var subtypeCount int
	err := r.db.QueryRow(ctx, query, principalID).Scan(
		&p.ID, &p.Type, &p.Status, &p.TokenVersion,
		&p.SchoolID, &p.Username, &mustChangePassword, &staffStatus, &schoolStatus,
		&hasPlatformAccount, &mobileVerified, &hasEligibleParentLink, &mfaVerified, &subtypeCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, models.ErrPrincipalNotAuthenticated
	}
	if err != nil {
		return nil, fmt.Errorf("load principal for authentication: %w", err)
	}

	if p.Status != "active" || p.TokenVersion <= 0 || subtypeCount != 1 || !mfaVerified {
		return nil, models.ErrPrincipalNotAuthenticated
	}
	switch p.Type {
	case "staff":
		if p.SchoolID <= 0 || p.Username == "" || staffStatus != "active" || schoolStatus != "active" {
			return nil, models.ErrPrincipalNotAuthenticated
		}
	case "platform_admin":
		if !hasPlatformAccount {
			return nil, models.ErrPrincipalNotAuthenticated
		}
	case "parent":
		if !mobileVerified || !hasEligibleParentLink {
			return nil, models.ErrPrincipalNotAuthenticated
		}
	default:
		return nil, models.ErrPrincipalNotAuthenticated
	}
	p.MobileVerified = mobileVerified
	p.MustChangePassword = p.Type == "staff" && mustChangePassword
	return &p, nil
}

// LoadEffectivePermissions intentionally fails closed for user-specific ALLOW
// overrides and custom-role grants. The current V4 schema does not bind those
// grants to the exact delegation that authorized them, so enabling them here
// could revive stale privileges after a delegation is revoked and re-granted.
// Built-in system-role grants and explicit DENY overrides are safe to resolve.
func (r *PrincipalAuthorizationRepository) LoadEffectivePermissions(
	ctx context.Context,
	principal *models.AuthPrincipal,
) ([]string, error) {
	if principal == nil || principal.ID <= 0 {
		return nil, fmt.Errorf("invalid principal")
	}

	const roleQuery = `
SELECT DISTINCT r.role_key, p.permission_key
FROM principal_roles pr
JOIN roles r ON r.id = pr.role_id
JOIN role_permissions rp ON rp.role_id = r.id
JOIN permissions p ON p.id = rp.permission_id
LEFT JOIN role_permission_delegations d
  ON d.id = rp.delegation_id
 AND d.school_id = r.school_id
 AND d.permission_id = rp.permission_id
 AND d.can_assign_to_roles = TRUE
 AND d.revoked_at IS NULL
WHERE pr.principal_id = $1
  AND pr.revoked_at IS NULL
  AND p.is_active = TRUE
  AND (
      (r.is_system = TRUE AND rp.delegation_id IS NULL)
      OR
      (r.is_system = FALSE AND r.scope_type = 'school' AND d.id IS NOT NULL)
  )
  AND r.principal_type = $2
  AND (
      ($2 = 'staff' AND r.scope_type = 'school' AND r.school_id = $3)
      OR ($2 = 'platform_admin' AND r.scope_type = 'platform' AND r.school_id IS NULL)
      OR ($2 = 'parent' AND r.scope_type = 'global' AND r.school_id IS NULL)
  )`

	rows, err := r.db.Query(ctx, roleQuery, principal.ID, principal.Type, nullableSchoolID(principal))
	if err != nil {
		return nil, fmt.Errorf("load built-in role permissions: %w", err)
	}
	set := make(map[string]struct{})
	for rows.Next() {
		var roleKey, key string
		if err := rows.Scan(&roleKey, &key); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		// The migration draft currently seeds every school permission into
		// School Admin. Enforce the approved allowlist at runtime as a second
		// barrier, including protection from future permission catalogue rows.
		if principal.Type == "staff" && roleKey == "school_admin" && !approvedSchoolAdminPermissions[key] {
			continue
		}
		set[key] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate role permissions: %w", err)
	}
	rows.Close()

	if principal.Type == "staff" {
		// ALLOW overrides are effective only while the exact delegation row
		// that authorized them remains active. A later delegation for the same
		// permission has a different ID and cannot revive this override.
		const allowQuery = `
SELECT p.permission_key
FROM user_permission_overrides o
JOIN permissions p ON p.id = o.permission_id
JOIN role_permission_delegations d
  ON d.id = o.delegation_id
 AND d.school_id = o.school_id
 AND d.permission_id = o.permission_id
 AND d.can_override_users = TRUE
 AND d.revoked_at IS NULL
WHERE o.principal_id = $1 AND o.school_id = $2
  AND o.effect = 'allow' AND p.is_active = TRUE`
		rows, err := r.db.Query(ctx, allowQuery, principal.ID, principal.SchoolID)
		if err != nil {
			return nil, fmt.Errorf("load delegated permission allowances: %w", err)
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan delegated permission allowance: %w", err)
			}
			set[key] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate delegated permission allowances: %w", err)
		}
		rows.Close()

		const denyQuery = `
SELECT p.permission_key
FROM user_permission_overrides o
JOIN permissions p ON p.id = o.permission_id
WHERE o.principal_id = $1 AND o.school_id = $2
  AND o.effect = 'deny' AND p.is_active = TRUE`
		rows, err = r.db.Query(ctx, denyQuery, principal.ID, principal.SchoolID)
		if err != nil {
			return nil, fmt.Errorf("load explicit permission denials: %w", err)
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan permission denial: %w", err)
			}
			delete(set, key)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate permission denials: %w", err)
		}
		rows.Close()
	}

	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	return result, nil
}

func nullableSchoolID(principal *models.AuthPrincipal) any {
	if principal.SchoolID <= 0 {
		return nil
	}
	return principal.SchoolID
}

var approvedSchoolAdminPermissions = map[string]bool{
	"school.settings.read":              true,
	"school.settings.update":            true,
	"school.audit.read":                 true,
	"academic_years.read":               true,
	"academic_years.manage":             true,
	"classes.read":                      true,
	"classes.manage":                    true,
	"subjects.read":                     true,
	"subjects.manage":                   true,
	"teacher_assignments.read":          true,
	"teacher_assignments.manage":        true,
	"class_incharge_assignments.read":   true,
	"class_incharge_assignments.manage": true,
	"students.read_basic":               true,
	"students.read_sensitive":           true,
	"students.create":                   true,
	"students.update":                   true,
	"students.archive":                  true,
	"attendance.read":                   true,
	"attendance.mark":                   true,
	"attendance.update_same_day":        true,
	"attendance.correct_past":           true,
	"timetable.read":                    true,
	"timetable.manage":                  true,
	"homework.read":                     true,
	"homework.create":                   true,
	"homework.update":                   true,
	"homework.delete":                   true,
	"exams.read":                        true,
	"exams.manage":                      true,
	"results.read":                      true,
	"results.enter":                     true,
	"results.update":                    true,
	"results.publish":                   true,
	"notices.read":                      true,
	"notices.create":                    true,
	"notices.update":                    true,
	"notices.publish":                   true,
	"notices.delete":                    true,
	"users.read":                        true,
	"users.create":                      true,
	"users.update":                      true,
	"users.disable":                     true,
	"users.password_reset":              true,
	"roles.read":                        true,
	"roles.create_custom":               true,
	"roles.update_custom":               true,
	"roles.assign":                      true,
	"permissions.read":                  true,
	"permissions.assign":                true,
	"permissions.override":              true,
}
