package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSchoolCodeTaken = errors.New("school code already exists")

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
		        jsonb_build_object('school_code', $4::text, 'school_admin_principal_id', $5::bigint))
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
