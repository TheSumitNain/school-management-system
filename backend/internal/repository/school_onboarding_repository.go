package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSchoolCodeTaken = errors.New("school code already exists")
	ErrPlatformSchoolListDenied = errors.New("platform school listing is not authorized")
)

type PlatformSchoolRecord struct {
	ID int64 `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SchoolOnboardingResult struct {
	SchoolID         int64
	SchoolName       string
	SchoolCode       string
	SchoolStatus     string
	AdminPrincipalID int64
	AdminUsername    string
}

type SchoolOnboardingRepository struct {
	db *pgxpool.Pool
}

func NewSchoolOnboardingRepository(db *pgxpool.Pool) *SchoolOnboardingRepository {
	return &SchoolOnboardingRepository{db: db}
}

// CreateSchoolWithAdmin commits the school, V13 role provisioning, first
// School Admin identity, role assignment and audit event as one transaction.
func (r *SchoolOnboardingRepository) CreateSchoolWithAdmin(
	ctx context.Context,
	actorPrincipalID int64,
	name, code, adminUsername, passwordHash string,
) (*SchoolOnboardingResult, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 ||
		name == "" || code == "" || adminUsername == "" || passwordHash == "" {
		return nil, ErrInvalidUser
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin school onboarding transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := &SchoolOnboardingResult{}
	err = tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id, name, code, status
	`, name, code).Scan(&result.SchoolID, &result.SchoolName, &result.SchoolCode, &result.SchoolStatus)
	if err != nil {
		return nil, mapSchoolOnboardingError(err)
	}

	// V4's BEFORE INSERT trigger creates the canonical active staff principal.
	err = tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING principal_id, username
	`, result.SchoolID, adminUsername, passwordHash).Scan(&result.AdminPrincipalID, &result.AdminUsername)
	if err != nil {
		return nil, mapSchoolOnboardingError(err)
	}

	// The database trigger independently verifies the assigning Platform
	// Admin has platform.school_admins.create.
	var assignmentID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		SELECT $1, r.id, $2
		  FROM roles r
		 WHERE r.school_id = $3
		   AND r.principal_type = 'staff'
		   AND r.scope_type = 'school'
		   AND r.role_key = 'school_admin'
		   AND r.is_system = TRUE
		RETURNING id
	`, result.AdminPrincipalID, actorPrincipalID, result.SchoolID).Scan(&assignmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provisioned School Admin system role is missing")
		}
		return nil, mapSchoolOnboardingError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events
			(actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'platform.school.created', 'school', $3, 'success',
		        jsonb_build_object('school_code', $4, 'school_admin_principal_id', $5))
	`, actorPrincipalID, result.SchoolID, strconv.FormatInt(result.SchoolID, 10), result.SchoolCode, result.AdminPrincipalID)
	if err != nil {
		return nil, fmt.Errorf("write school onboarding audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit school onboarding transaction: %w", err)
	}
	return result, nil
}

func mapSchoolOnboardingError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "schools_code_key":
			return ErrSchoolCodeTaken
		case "ux_users_username_global":
			return ErrUsernameTaken
		}
	}
	return fmt.Errorf("create school and initial administrator: %w", err)
}


// ListPlatformSchools returns school records only to an active Platform Admin
// with the explicit platform.schools.read permission.
func (r *SchoolOnboardingRepository) ListPlatformSchools(ctx context.Context, actorPrincipalID int64) ([]PlatformSchoolRecord, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 {
		return nil, ErrPlatformSchoolListDenied
	}
	var authorized bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles role ON role.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = role.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1
			  AND ap.principal_type = 'platform_admin'
			  AND ap.status = 'active'
			  AND role.scope_type = 'platform'
			  AND role.school_id IS NULL
			  AND role.principal_type = 'platform_admin'
			  AND role.role_key = 'platform_admin'
			  AND role.is_system = TRUE
			  AND p.permission_key = 'platform.schools.read'
			  AND p.scope_type = 'platform'
			  AND p.is_active = TRUE
		)
	`, actorPrincipalID).Scan(&authorized); err != nil {
		return nil, fmt.Errorf("authorize platform school listing: %w", err)
	}
	if !authorized {
		return nil, ErrPlatformSchoolListDenied
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, name, code, status, created_at
		FROM schools
		ORDER BY LOWER(name), id
	`)
	if err != nil {
		return nil, fmt.Errorf("list platform schools: %w", err)
	}
	defer rows.Close()
	schools := make([]PlatformSchoolRecord, 0)
	for rows.Next() {
		var school PlatformSchoolRecord
		if err := rows.Scan(&school.ID, &school.Name, &school.Code, &school.Status, &school.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan platform school: %w", err)
		}
		schools = append(schools, school)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate platform schools: %w", err)
	}
	return schools, nil
}
