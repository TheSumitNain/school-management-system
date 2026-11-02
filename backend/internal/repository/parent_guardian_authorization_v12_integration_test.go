package repository

import (
	"context"
	"strconv"
	"testing"
	"time"
)

// TestParentGuardianAuthorizationV12PostgresIntegration verifies the V12
// tenant-bound parent links, OTP pre-account challenges, and revocation history.
func TestParentGuardianAuthorizationV12PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV12 bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.parent_guardian_links') IS NOT NULL
		   AND to_regclass('public.parent_otp_rate_limits') IS NOT NULL
	`).Scan(&hasV12); err != nil {
		t.Fatalf("detect V12 schema: %v", err)
	}
	if !hasV12 {
		t.Fatal("V12 parent/guardian schema is missing; apply migrations to school_management_test first")
	}

	var schoolID, staffPrincipalID int64
	if err := pool.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &staffPrincipalID); err != nil {
		t.Fatalf("load integration school and staff principal: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin V12 integration transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	expectFailure := func(name string, operation func() error) {
		t.Helper()
		if _, err := tx.Exec(ctx, "SAVEPOINT v12_expected_failure"); err != nil {
			t.Fatalf("%s: create savepoint: %v", name, err)
		}
		operationErr := operation()
		if operationErr == nil {
			_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT v12_expected_failure")
			_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT v12_expected_failure")
			t.Fatalf("%s: expected database rejection", name)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT v12_expected_failure"); err != nil {
			t.Fatalf("%s: rollback expected failure: %v", name, err)
		}
		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT v12_expected_failure"); err != nil {
			t.Fatalf("%s: release savepoint: %v", name, err)
		}
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobile := "+9198765" + suffix[len(suffix)-5:]
	var guardianID, studentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO guardians
			(school_id, full_name, mobile_number, mobile_number_e164,
			 parent_portal_enabled_at, parent_portal_enabled_by)
		VALUES ($1, $2, $3, $3, NOW(), $4)
		RETURNING id
	`, schoolID, "V12 Guardian "+suffix, mobile, staffPrincipalID).Scan(&guardianID); err != nil {
		t.Fatalf("create portal-enabled guardian: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "V12-"+suffix, "V12 Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create student: %v", err)
	}

	var parentPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type, status)
		VALUES ('parent', 'active') RETURNING id
	`).Scan(&parentPrincipalID); err != nil {
		t.Fatalf("create parent principal: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		VALUES ($1, $2, NOW())
	`, parentPrincipalID, mobile); err != nil {
		t.Fatalf("create verified parent account: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("create official student-guardian relationship: %v", err)
	}

	// Stage B matching must use the exact portal-enabled guardian mobile and
	// only count active official student relationships in the same school.
	matches, err := findPortalEligibleGuardiansByMobile(ctx, tx, mobile)
	if err != nil {
		t.Fatalf("match official guardian mobile: %v", err)
	}
	if len(matches) != 1 || matches[0].GuardianID != guardianID ||
		matches[0].SchoolID != schoolID || matches[0].GuardianName != "V12 Guardian "+suffix ||
		matches[0].ActiveStudentCount != 1 {
		t.Fatalf("unexpected official guardian matches: %#v", matches)
	}
	otherMobileMatches, err := findPortalEligibleGuardiansByMobile(ctx, tx, mobile+"9")
	if err != nil {
		t.Fatalf("query non-matching guardian mobile: %v", err)
	}
	if len(otherMobileMatches) != 0 {
		t.Fatalf("expected exact mobile matching, got %#v", otherMobileMatches)
	}

	var parentLinkID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO parent_guardian_links (principal_id, guardian_id, school_id, link_method)
		VALUES ($1, $2, $3, 'otp_verified') RETURNING id
	`, parentPrincipalID, guardianID, schoolID).Scan(&parentLinkID); err != nil {
		t.Fatalf("create verified parent-guardian link: %v", err)
	}

	expectFailure("change Parent mobile while guardian link is active", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE parent_accounts SET mobile_number = $1 WHERE principal_id = $2
		`, mobile+"1", parentPrincipalID)
		return err
	})

	// A guardian contact can have only one active parent link.
	expectFailure("duplicate active parent guardian link", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO parent_guardian_links
				(principal_id, guardian_id, school_id, link_method)
			VALUES ($1, $2, $3, 'school_approved')
		`, parentPrincipalID, guardianID, schoolID)
		return err
	})

	// Parent can access only students connected through both active link layers.
	var visibleChildren int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(DISTINCT sg.student_id)
		FROM parent_guardian_links pgl
		JOIN guardians g ON g.id = pgl.guardian_id AND g.school_id = pgl.school_id
		JOIN student_guardians sg ON sg.guardian_id = g.id AND sg.school_id = g.school_id
		JOIN students s ON s.id = sg.student_id AND s.school_id = sg.school_id
		WHERE pgl.principal_id = $1
		  AND pgl.revoked_at IS NULL
		  AND sg.revoked_at IS NULL
		  AND g.status = 'active'
		  AND s.status = 'active'
	`, parentPrincipalID).Scan(&visibleChildren); err != nil {
		t.Fatalf("query linked children: %v", err)
	}
	if visibleChildren != 1 {
		t.Fatalf("expected one linked child, got %d", visibleChildren)
	}

	// Revoke the official relationship and prove the historical pair can be
	// re-established as a new row without reactivating old history.
	if _, err := tx.Exec(ctx, `
		UPDATE student_guardians
		   SET revoked_at = NOW(), revoked_by_principal_id = $1,
		       revocation_reason = 'V12 integration test'
		 WHERE school_id = $2 AND student_id = $3 AND guardian_id = $4
	`, staffPrincipalID, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("revoke official student-guardian relationship: %v", err)
	}

	if err := tx.QueryRow(ctx, `
		SELECT COUNT(DISTINCT sg.student_id)
		FROM parent_guardian_links pgl
		JOIN student_guardians sg ON sg.guardian_id = pgl.guardian_id AND sg.school_id = pgl.school_id
		WHERE pgl.principal_id = $1 AND pgl.revoked_at IS NULL AND sg.revoked_at IS NULL
	`, parentPrincipalID).Scan(&visibleChildren); err != nil {
		t.Fatalf("recheck children after official relationship revocation: %v", err)
	}
	if visibleChildren != 0 {
		t.Fatalf("expected revoked official relationship to hide child immediately, got %d", visibleChildren)
	}

	matches, err = findPortalEligibleGuardiansByMobile(ctx, tx, mobile)
	if err != nil {
		t.Fatalf("rematch guardian after official relationship revocation: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("revoked official relationship remained eligible for matching: %#v", matches)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("re-establish relationship as new history row: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE parent_guardian_links
		   SET revoked_at = NOW(), revoked_by_principal_id = $1,
		       revocation_reason = 'V12 integration test'
		 WHERE id = $2
	`, parentPrincipalID, parentLinkID); err != nil {
		t.Fatalf("revoke parent-guardian link: %v", err)
	}
	expectFailure("reactivate revoked parent guardian link", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE parent_guardian_links
			   SET revoked_at = NULL, revoked_by_principal_id = NULL, revocation_reason = NULL
			 WHERE id = $1
		`, parentLinkID)
		return err
	})

	// First-time OTP challenges are mobile-HMAC-bound without a provisional
	// principal. Only the HMAC and OTP hash are persisted.
	mobileHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	otpHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_otp_challenges (principal_id, mobile_lookup_hash, otp_hash, expires_at)
		VALUES (NULL, $1, $2, NOW() + INTERVAL '5 minutes')
	`, mobileHash, otpHash); err != nil {
		t.Fatalf("create pre-account parent OTP challenge: %v", err)
	}
	expectFailure("duplicate active pre-account OTP challenge", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO parent_otp_challenges (principal_id, mobile_lookup_hash, otp_hash, expires_at)
			VALUES (NULL, $1, $2, NOW() + INTERVAL '5 minutes')
		`, mobileHash, otpHash)
		return err
	})
	expectFailure("mutate pre-account OTP mobile HMAC", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE parent_otp_challenges SET mobile_lookup_hash = $1
			WHERE mobile_lookup_hash = $2
		`, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", mobileHash)
		return err
	})

	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_otp_rate_limits
			(subject_type, subject_hash, window_started_at, send_count, last_sent_at)
		VALUES ('mobile', $1, date_trunc('minute', NOW()), 1, NOW())
	`, mobileHash); err != nil {
		t.Fatalf("insert persistent OTP rate-limit bucket: %v", err)
	}
}
