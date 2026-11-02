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

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobile := "+9198765" + suffix[len(suffix)-5:]
	var guardianID, studentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO guardians
			(school_id, full_name, mobile_number, mobile_number_e164,
			 parent_portal_enabled_at, parent_portal_enabled_by)
		VALUES ($1, $2, $3, $3, NOW(), $4)
		RETURNING id
	`, schoolID, "V12 Guardian "+suffix, mobile, staffPrincipalID).Scan(&guardianID); err != nil {
		t.Fatalf("create portal-enabled guardian: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "V12-"+suffix, "V12 Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create student: %v", err)
	}

	var parentPrincipalID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type, status)
		VALUES ('parent', 'active') RETURNING id
	`).Scan(&parentPrincipalID); err != nil {
		t.Fatalf("create parent principal: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		VALUES ($1, $2, NOW())
	`, parentPrincipalID, mobile); err != nil {
		t.Fatalf("create verified parent account: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("create official student-guardian relationship: %v", err)
	}

	var parentLinkID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO parent_guardian_links (principal_id, guardian_id, school_id, link_method)
		VALUES ($1, $2, $3, 'otp_verified') RETURNING id
	`, parentPrincipalID, guardianID, schoolID).Scan(&parentLinkID); err != nil {
		t.Fatalf("create verified parent-guardian link: %v", err)
	}

	// A guardian contact cannot be actively claimed by a second Parent.
	if _, err := pool.Exec(ctx, `
		INSERT INTO parent_guardian_links
			(principal_id, guardian_id, school_id, link_method)
		VALUES ($1, $2, $3, 'school_approved')
	`, parentPrincipalID+999999, guardianID, schoolID); err == nil {
		t.Fatal("expected duplicate active guardian claim to be rejected")
	}

	// Parent can access only students connected through both active link layers.
	var visibleChildren int
	if err := pool.QueryRow(ctx, `
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
	if _, err := pool.Exec(ctx, `
		UPDATE student_guardians
		   SET revoked_at = NOW(), revoked_by_principal_id = $1,
		       revocation_reason = 'V12 integration test'
		 WHERE school_id = $2 AND student_id = $3 AND guardian_id = $4
	`, staffPrincipalID, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("revoke official student-guardian relationship: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("re-establish relationship as new history row: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE parent_guardian_links
		   SET revoked_at = NOW(), revoked_by_principal_id = $1,
		       revocation_reason = 'V12 integration test'
		 WHERE id = $2
	`, parentPrincipalID, parentLinkID); err != nil {
		t.Fatalf("revoke parent-guardian link: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE parent_guardian_links
		   SET revoked_at = NULL, revoked_by_principal_id = NULL, revocation_reason = NULL
		 WHERE id = $1
	`, parentLinkID); err == nil {
		t.Fatal("expected revoked parent-guardian link to be immutable")
	}

	// First-time OTP challenges are mobile-HMAC-bound without a provisional
	// principal. Only the HMAC and OTP hash are persisted.
	mobileHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	otpHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := pool.Exec(ctx, `
		INSERT INTO parent_otp_challenges (principal_id, mobile_lookup_hash, otp_hash, expires_at)
		VALUES (NULL, $1, $2, NOW() + INTERVAL '5 minutes')
	`, mobileHash, otpHash); err != nil {
		t.Fatalf("create pre-account parent OTP challenge: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO parent_otp_challenges (principal_id, mobile_lookup_hash, otp_hash, expires_at)
		VALUES (NULL, $1, $2, NOW() + INTERVAL '5 minutes')
	`, mobileHash, otpHash); err == nil {
		t.Fatal("expected only one active pre-account OTP challenge per mobile HMAC")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE parent_otp_challenges SET mobile_lookup_hash = $1
		WHERE mobile_lookup_hash = $2
	`, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", mobileHash); err == nil {
		t.Fatal("expected OTP mobile lookup HMAC to be immutable")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO parent_otp_rate_limits
			(subject_type, subject_hash, window_started_at, send_count, last_sent_at)
		VALUES ('mobile', $1, date_trunc('minute', NOW()), 1, NOW())
	`, mobileHash); err != nil {
		t.Fatalf("insert persistent OTP rate-limit bucket: %v", err)
	}
}
