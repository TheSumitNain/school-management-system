package repository

import (
	"context"
	"errors"
	"fmt"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrParentAccountAlreadyExists  = errors.New("a Parent account already exists for this mobile")
	ErrNoEligibleParentGuardians    = errors.New("no eligible official guardian records found")
	ErrParentGuardianAlreadyClaimed = errors.New("one or more guardian records are already linked")
)

// ParentRegistrationResult contains only the authenticated identity and the
// number of linked guardian records. Student and guardian profile data is not
// returned by the account-provisioning boundary.
type ParentRegistrationResult struct {
	Principal           *models.AuthPrincipal
	LinkedGuardianCount int
}

// ParentRegistrationRepository provisions a verified Parent identity and all
// currently eligible official guardian links in one PostgreSQL transaction.
type ParentRegistrationRepository struct {
	db *pgxpool.Pool
}

func NewParentRegistrationRepository(db *pgxpool.Pool) *ParentRegistrationRepository {
	return &ParentRegistrationRepository{db: db}
}

// ProvisionVerifiedMobile serializes registrations for the same mobile,
// rechecks official guardian eligibility under row locks, and atomically
// creates the Parent subtype, global Parent role, and OTP-verified links.
// The caller must have consumed a valid pre-account OTP before calling.
func (r *ParentRegistrationRepository) ProvisionVerifiedMobile(
	ctx context.Context, mobileE164 string,
) (*ParentRegistrationResult, error) {
	if r == nil || r.db == nil || !parentGuardianE164Pattern.MatchString(mobileE164) {
		return nil, fmt.Errorf("invalid Parent registration request")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin Parent registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize duplicate registrations for one mobile. This lock is held
	// through commit and complements the unique mobile index.
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('parent-registration:' || $1, 0))`,
		mobileE164,
	); err != nil {
		return nil, fmt.Errorf("lock Parent registration mobile: %w", err)
	}

	var existingPrincipalID int64
	err = tx.QueryRow(ctx,
		`SELECT principal_id FROM parent_accounts WHERE mobile_number=$1`,
		mobileE164,
	).Scan(&existingPrincipalID)
	if err == nil {
		return nil, ErrParentAccountAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check existing Parent account: %w", err)
	}

	// Lock guardian, relationship, and student rows while taking a fresh
	// eligibility snapshot. Repeated rows represent multiple active children
	// for one guardian and are deduplicated before links are inserted.
	rows, err := tx.Query(ctx, `
		SELECT g.id, g.school_id
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
		 ORDER BY g.id, s.id
		 FOR UPDATE OF g, sg, s
	`, mobileE164)
	if err != nil {
		return nil, fmt.Errorf("lock eligible official guardian records: %w", err)
	}

	type guardianRef struct{ id, schoolID int64 }
	guardians := make([]guardianRef, 0)
	seen := make(map[int64]struct{})
	for rows.Next() {
		var guardian guardianRef
		if err := rows.Scan(&guardian.id, &guardian.schoolID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan eligible guardian record: %w", err)
		}
		if _, ok := seen[guardian.id]; ok {
			continue
		}
		seen[guardian.id] = struct{}{}
		guardians = append(guardians, guardian)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate eligible guardian records: %w", err)
	}
	rows.Close()

	if len(guardians) == 0 {
		return nil, ErrNoEligibleParentGuardians
	}

	var principalID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO auth_principals (principal_type, status)
		 VALUES ('parent', 'active') RETURNING id`,
	).Scan(&principalID); err != nil {
		return nil, fmt.Errorf("create Parent principal: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		 VALUES ($1, $2, NOW())`,
		principalID, mobileE164,
	); err != nil {
		if isUniqueConstraintViolation(err) {
			return nil, ErrParentAccountAlreadyExists
		}
		return nil, fmt.Errorf("create verified Parent account: %w", err)
	}

	// NULL assigned_by is allowed by the database only for the verified Parent
	// principal receiving the migration-seeded global Parent system role.
	var roleAssignmentID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		SELECT $1, r.id, NULL
		  FROM roles r
		 WHERE r.role_key='parent'
		   AND r.principal_type='parent'
		   AND r.scope_type='global'
		   AND r.school_id IS NULL
		   AND r.is_system=TRUE
		RETURNING id
	`, principalID).Scan(&roleAssignmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("global Parent system role is missing")
	}
	if err != nil {
		return nil, fmt.Errorf("assign global Parent role: %w", err)
	}

	for _, guardian := range guardians {
		if _, err := tx.Exec(ctx, `
			INSERT INTO parent_guardian_links
				(principal_id, guardian_id, school_id, link_method)
			VALUES ($1, $2, $3, 'otp_verified')
		`, principalID, guardian.id, guardian.schoolID); err != nil {
			if isUniqueConstraintViolation(err) {
				return nil, ErrParentGuardianAlreadyClaimed
			}
			return nil, fmt.Errorf("link verified Parent to official guardian: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit Parent registration: %w", err)
	}

	return &ParentRegistrationResult{
		Principal: &models.AuthPrincipal{
			ID: principalID, Type: "parent", Status: "active",
			TokenVersion: 1, MobileVerified: true,
		},
		LinkedGuardianCount: len(guardians),
	}, nil
}

func isUniqueConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
