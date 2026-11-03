package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	authorizationRepo := NewPrincipalAuthorizationRepository(pool)
	if _, err := authorizationRepo.LoadPrincipalForAuthentication(ctx, principalID); err != nil {
		t.Fatalf("eligible Parent should authenticate before link revocation: %v", err)
	}

	refreshRepo := NewPrincipalRefreshSessionRepository(pool)
	oldDigest := sha256.Sum256([]byte("parent-session-before-link-revocation-" + suffix))
	oldHash := hex.EncodeToString(oldDigest[:])
	newDigest := sha256.Sum256([]byte("parent-session-after-link-revocation-" + suffix))
	newHash := hex.EncodeToString(newDigest[:])
	if err := refreshRepo.CreatePrincipal(ctx, principalID, oldHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create refresh session for eligible Parent: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE parent_guardian_links
		   SET revoked_at = NOW(),
		       revoked_by_principal_id = $2,
		       revocation_reason = 'integration test: revoke Parent eligibility'
		 WHERE principal_id = $1 AND revoked_at IS NULL
	`, principalID); err != nil {
		t.Fatalf("revoke all Parent guardian links: %v", err)
	}

	if _, err := authorizationRepo.LoadPrincipalForAuthentication(ctx, principalID); !errors.Is(err, models.ErrPrincipalNotAuthenticated) {
		t.Fatalf("Parent authentication after all links revoked = %v, want not authenticated", err)
	}
	if _, err := repo.FindActiveParentByMobile(ctx, mobile); !errors.Is(err, models.ErrPrincipalNotAuthenticated) {
		t.Fatalf("Parent mobile lookup after all links revoked = %v, want not authenticated", err)
	}
	if _, err := refreshRepo.RotatePrincipal(ctx, oldHash, newHash, time.Now().Add(2*time.Hour)); !errors.Is(err, ErrRefreshSessionInactive) {
		t.Fatalf("Parent refresh after all links revoked = %v, want inactive session", err)
	}

	var activeSessions int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_sessions
		 WHERE principal_id = $1 AND revoked_at IS NULL
	`, principalID).Scan(&activeSessions); err != nil {
		t.Fatalf("count remaining active Parent refresh sessions: %v", err)
	}
	if activeSessions != 0 {
		t.Fatalf("active Parent refresh sessions after link revocation = %d, want 0", activeSessions)
	}
}
