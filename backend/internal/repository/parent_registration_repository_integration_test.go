package repository

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestParentRegistrationRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV12 bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.parent_guardian_links') IS NOT NULL
		   AND to_regclass('public.parent_accounts') IS NOT NULL
	`).Scan(&hasV12); err != nil {
		t.Fatalf("detect Parent registration schema: %v", err)
	}
	if !hasV12 {
		t.Skip("apply V12 only to school_management_test to run Parent registration integration tests")
	}

	var schoolID, staffPrincipalID int64
	if err := pool.QueryRow(ctx,
		`SELECT school_id, principal_id FROM users WHERE id=$1`, userID,
	).Scan(&schoolID, &staffPrincipalID); err != nil {
		t.Fatalf("load integration school and staff principal: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin Parent registration integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobile := "+9198765" + suffix[len(suffix)-5:]
	var guardianIDs [2]int64
	var studentIDs [2]int64

	for i := range guardianIDs {
		if err := tx.QueryRow(ctx, `
			INSERT INTO guardians
				(school_id, full_name, mobile_number, mobile_number_e164,
				 parent_portal_enabled_at, parent_portal_enabled_by)
			VALUES ($1, $2, $3, $3, NOW(), $4)
			RETURNING id
		`, schoolID, "Registration Guardian "+suffix+"-"+strconv.Itoa(i),
			mobile, staffPrincipalID).Scan(&guardianIDs[i]); err != nil {
			t.Fatalf("create portal-enabled guardian %d: %v", i, err)
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO students (school_id, admission_number, full_name)
			VALUES ($1, $2, $3) RETURNING id
		`, schoolID, "PREG-"+suffix+"-"+strconv.Itoa(i),
			"Registration Student "+strconv.Itoa(i)).Scan(&studentIDs[i]); err != nil {
			t.Fatalf("create student %d: %v", i, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_guardians
				(school_id, student_id, guardian_id, relationship_type)
			VALUES ($1, $2, $3, 'guardian')
		`, schoolID, studentIDs[i], guardianIDs[i]); err != nil {
			t.Fatalf("create official student-guardian relationship %d: %v", i, err)
		}
	}

	repo := NewParentRegistrationRepository(pool)

	if _, err := repo.ProvisionVerifiedMobileInTx(ctx, tx, mobile+"9"); !errors.Is(err, ErrNoEligibleParentGuardians) {
		t.Fatalf("unmatched mobile error = %v, want ErrNoEligibleParentGuardians", err)
	}

	result, err := repo.ProvisionVerifiedMobileInTx(ctx, tx, mobile)
	if err != nil {
		t.Fatalf("provision verified Parent: %v", err)
	}
	if result.Principal == nil || result.Principal.Type != "parent" ||
		result.Principal.Status != "active" || !result.Principal.MobileVerified ||
		result.LinkedGuardianCount != 2 {
		t.Fatalf("unexpected Parent registration result: %+v", result)
	}

	var accountCount, activeRoleCount, activeLinkCount, auditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM parent_accounts
		 WHERE principal_id=$1 AND mobile_number=$2 AND mobile_verified_at IS NOT NULL
	`, result.Principal.ID, mobile).Scan(&accountCount); err != nil {
		t.Fatalf("verify Parent account: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM principal_roles pr
		  JOIN roles r ON r.id=pr.role_id
		 WHERE pr.principal_id=$1 AND pr.revoked_at IS NULL
		   AND r.role_key='parent' AND r.is_system=TRUE
	`, result.Principal.ID).Scan(&activeRoleCount); err != nil {
		t.Fatalf("verify Parent role: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM parent_guardian_links
		 WHERE principal_id=$1 AND revoked_at IS NULL AND link_method='otp_verified'
	`, result.Principal.ID).Scan(&activeLinkCount); err != nil {
		t.Fatalf("verify active guardian links: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE actor_principal_id=$1 AND action='parent.guardian_link_created'
	`, result.Principal.ID).Scan(&auditCount); err != nil {
		t.Fatalf("verify Parent link audit events: %v", err)
	}
	if accountCount != 1 || activeRoleCount != 1 || activeLinkCount != 2 || auditCount != 2 {
		t.Fatalf("account=%d role=%d links=%d audit=%d; want 1/1/2/2",
			accountCount, activeRoleCount, activeLinkCount, auditCount)
	}

	if _, err := repo.ProvisionVerifiedMobileInTx(ctx, tx, mobile); !errors.Is(err, ErrParentAccountAlreadyExists) {
		t.Fatalf("duplicate registration error = %v, want ErrParentAccountAlreadyExists", err)
	}
}
