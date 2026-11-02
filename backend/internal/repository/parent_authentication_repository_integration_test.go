package repository

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"school-management-system/internal/models"
)

func TestParentAuthenticationRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var schoolID, staffPrincipalID int64
	if err := pool.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &staffPrincipalID); err != nil {
		t.Fatalf("load integration school and staff principal: %v", err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobile := "+9198765" + suffix[len(suffix)-5:]
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin Parent authentication fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var guardianID, studentID, principalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO guardians
			(school_id, full_name, mobile_number, mobile_number_e164,
			 parent_portal_enabled_at, parent_portal_enabled_by)
		VALUES ($1, $2, $3, $3, NOW(), $4)
		RETURNING id
	`, schoolID, "Parent Auth Guardian "+suffix, mobile, staffPrincipalID).Scan(&guardianID); err != nil {
		t.Fatalf("create eligible guardian: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "PARENT-AUTH-"+suffix, "Parent Auth Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create active student: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("create official guardian relationship: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type, status)
		VALUES ('parent', 'active') RETURNING id
	`).Scan(&principalID); err != nil {
		t.Fatalf("create Parent principal: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		VALUES ($1, $2, NOW())
	`, principalID, mobile); err != nil {
		t.Fatalf("create verified Parent account: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		SELECT $1, r.id, NULL
		  FROM roles r
		 WHERE r.role_key = 'parent'
		   AND r.principal_type = 'parent'
		   AND r.scope_type = 'global'
		   AND r.school_id IS NULL
		   AND r.is_system = TRUE
	`, principalID); err != nil {
		t.Fatalf("assign global Parent system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_guardian_links (principal_id, guardian_id, school_id, link_method)
		VALUES ($1, $2, $3, 'otp_verified')
	`, principalID, guardianID, schoolID); err != nil {
		t.Fatalf("create active Parent guardian link: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit Parent authentication fixture: %v", err)
	}

	repo := NewParentAuthenticationRepository(pool)
	principal, err := repo.FindActiveParentByMobile(ctx, mobile)
	if err != nil {
		t.Fatalf("find active Parent by verified mobile: %v", err)
	}
	if principal.ID != principalID || principal.Type != "parent" ||
		principal.Status != "active" || principal.TokenVersion <= 0 || !principal.MobileVerified {
		t.Fatalf("unexpected Parent principal returned: %+v", principal)
	}

	if _, err := repo.FindActiveParentByMobile(ctx, mobile+"9"); !errors.Is(err, models.ErrPrincipalNotAuthenticated) {
		t.Fatalf("non-matching mobile error = %v, want principal not authenticated", err)
	}
}
