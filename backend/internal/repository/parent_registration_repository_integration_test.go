package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
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

	var mobile, suffix string
	for attempt := 0; attempt < 5; attempt++ {
		randomSuffix, err := rand.Int(rand.Reader, big.NewInt(1_000_000_000))
		if err != nil {
			t.Fatalf("generate unique Parent registration fixture mobile: %v", err)
		}
		suffix = fmt.Sprintf("%09d", randomSuffix.Int64())
		mobile = "+919" + suffix

		var available bool
		if err := pool.QueryRow(ctx, `
			SELECT NOT EXISTS (
				SELECT 1 FROM parent_accounts WHERE mobile_number = $1
			) AND NOT EXISTS (
				SELECT 1 FROM guardians WHERE mobile_number_e164 = $1
			)
		`, mobile).Scan(&available); err != nil {
			t.Fatalf("check Parent registration fixture mobile availability: %v", err)
		}
		if available {
			break
		}
		mobile = ""
	}
	if mobile == "" {
		t.Fatal("could not generate an unused Parent registration fixture mobile after 5 attempts")
	}
	var otherSchoolID, otherSchoolStaffPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active') RETURNING id
	`, "Parent Registration Cross School "+suffix,
		"PRCS-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second school fixture: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, otherSchoolID, "preg-staff-"+suffix).Scan(&otherSchoolStaffPrincipalID); err != nil {
		t.Fatalf("create second-school staff fixture: %v", err)
	}

	guardianSchoolIDs := []int64{schoolID, schoolID, otherSchoolID}
	guardianActorPrincipalIDs := []int64{staffPrincipalID, staffPrincipalID, otherSchoolStaffPrincipalID}
	var guardianIDs [3]int64
	var studentIDs [3]int64

	for i := range guardianIDs {
		if err := tx.QueryRow(ctx, `
			INSERT INTO guardians
				(school_id, full_name, mobile_number, mobile_number_e164,
				 parent_portal_enabled_at, parent_portal_enabled_by)
			VALUES ($1, $2, $3, $3, NOW(), $4)
			RETURNING id
		`, guardianSchoolIDs[i], "Registration Guardian "+suffix+"-"+strconv.Itoa(i),
			mobile, guardianActorPrincipalIDs[i]).Scan(&guardianIDs[i]); err != nil {
			t.Fatalf("create portal-enabled guardian %d: %v", i, err)
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO students (school_id, admission_number, full_name)
			VALUES ($1, $2, $3) RETURNING id
		`, guardianSchoolIDs[i], "PREG-"+suffix+"-"+strconv.Itoa(i),
			"Registration Student "+strconv.Itoa(i)).Scan(&studentIDs[i]); err != nil {
			t.Fatalf("create student %d: %v", i, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_guardians
				(school_id, student_id, guardian_id, relationship_type)
			VALUES ($1, $2, $3, 'guardian')
		`, guardianSchoolIDs[i], studentIDs[i], guardianIDs[i]); err != nil {
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
		result.LinkedGuardianCount != 3 {
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
	if accountCount != 1 || activeRoleCount != 1 || activeLinkCount != 3 || auditCount != 3 {
		t.Fatalf("account=%d role=%d links=%d audit=%d; want 1/1/3/3",
			accountCount, activeRoleCount, activeLinkCount, auditCount)
	}

	var linkedSchoolCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(DISTINCT school_id) FROM parent_guardian_links
		 WHERE principal_id=$1 AND revoked_at IS NULL
	`, result.Principal.ID).Scan(&linkedSchoolCount); err != nil {
		t.Fatalf("verify cross-school Parent links: %v", err)
	}
	if linkedSchoolCount != 2 {
		t.Fatalf("Parent linked schools=%d; want 2", linkedSchoolCount)
	}

	if _, err := repo.ProvisionVerifiedMobileInTx(ctx, tx, mobile); !errors.Is(err, ErrParentAccountAlreadyExists) {
		t.Fatalf("duplicate registration error = %v, want ErrParentAccountAlreadyExists", err)
	}
}


// TestParentRegistrationConcurrentSameMobilePostgresIntegration proves that
// concurrent verified registrations for one mobile produce exactly one
// Parent account and one set of guardian links.
func TestParentRegistrationConcurrentSameMobilePostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

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

	var mobile, suffix string
	for attempt := 0; attempt < 5; attempt++ {
		randomSuffix, err := rand.Int(rand.Reader, big.NewInt(1_000_000_000))
		if err != nil {
			t.Fatalf("generate unique Parent registration fixture mobile: %v", err)
		}
		suffix = fmt.Sprintf("%09d", randomSuffix.Int64())
		mobile = "+919" + suffix

		var available bool
		if err := pool.QueryRow(ctx, `
			SELECT NOT EXISTS (
				SELECT 1 FROM parent_accounts WHERE mobile_number = $1
			) AND NOT EXISTS (
				SELECT 1 FROM guardians WHERE mobile_number_e164 = $1
			)
		`, mobile).Scan(&available); err != nil {
			t.Fatalf("check Parent registration fixture mobile availability: %v", err)
		}
		if available {
			break
		}
		mobile = ""
	}
	if mobile == "" {
		t.Fatal("could not generate an unused Parent registration fixture mobile after 5 attempts")
	}

	fixtureTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin concurrent registration fixture: %v", err)
	}
	defer func() { _ = fixtureTx.Rollback(ctx) }()

	var guardianID, studentID int64
	if err := fixtureTx.QueryRow(ctx, `
		INSERT INTO guardians
			(school_id, full_name, mobile_number, mobile_number_e164,
			 parent_portal_enabled_at, parent_portal_enabled_by)
		VALUES ($1, $2, $3, $3, NOW(), $4)
		RETURNING id
	`, schoolID, "Concurrent Registration Guardian "+suffix,
		mobile, staffPrincipalID).Scan(&guardianID); err != nil {
		t.Fatalf("create portal-enabled guardian: %v", err)
	}
	if err := fixtureTx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "PCR-"+suffix, "Concurrent Registration Student").Scan(&studentID); err != nil {
		t.Fatalf("create student: %v", err)
	}
	if _, err := fixtureTx.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'guardian')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("create official student-guardian relationship: %v", err)
	}
	if err := fixtureTx.Commit(ctx); err != nil {
		t.Fatalf("commit concurrent registration fixture: %v", err)
	}

	repo := NewParentRegistrationRepository(pool)
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, errs[index] = repo.ProvisionVerifiedMobile(ctx, mobile)
		}(i)
	}
	close(start)
	wg.Wait()

	successCount, duplicateCount := 0, 0
	for _, registrationErr := range errs {
		switch {
		case registrationErr == nil:
			successCount++
		case errors.Is(registrationErr, ErrParentAccountAlreadyExists):
			duplicateCount++
		default:
			t.Fatalf("unexpected concurrent registration error: %v", registrationErr)
		}
	}
	if successCount != 1 || duplicateCount != 1 {
		t.Fatalf("concurrent outcomes: successes=%d duplicate_errors=%d; want 1/1 (errors=%v)",
			successCount, duplicateCount, errs)
	}

	var accountCount, activeLinkCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM parent_accounts WHERE mobile_number=$1`, mobile,
	).Scan(&accountCount); err != nil {
		t.Fatalf("verify concurrent Parent account count: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM parent_guardian_links pgl
		  JOIN parent_accounts pa ON pa.principal_id=pgl.principal_id
		 WHERE pa.mobile_number=$1 AND pgl.revoked_at IS NULL
	`, mobile).Scan(&activeLinkCount); err != nil {
		t.Fatalf("verify concurrent Parent guardian links: %v", err)
	}
	if accountCount != 1 || activeLinkCount != 1 {
		t.Fatalf("concurrent registration persisted accounts=%d links=%d; want 1/1",
			accountCount, activeLinkCount)
	}
}
