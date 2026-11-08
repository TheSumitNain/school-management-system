package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestSchoolOnboardingAuditMetadataPostgresIntegration guards the explicit
// PostgreSQL parameter types used by the school-onboarding audit JSON.
// PostgreSQL cannot infer the types of uncast parameters passed to
// jsonb_build_object when the arguments have no other type context.
func TestSchoolOnboardingAuditMetadataPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var metadata string
	if err := pool.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'school_code', $1::text,
			'school_admin_principal_id', $2::bigint
		)::text
	`, "ONBOARDING-TEST", int64(12345)).Scan(&metadata); err != nil {
		t.Fatalf("build school onboarding audit metadata: %v", err)
	}

	if !strings.Contains(metadata, `"school_code": "ONBOARDING-TEST"`) ||
		!strings.Contains(metadata, `"school_admin_principal_id": 12345`) {
		t.Fatalf("unexpected school onboarding audit metadata: %s", metadata)
	}
}
