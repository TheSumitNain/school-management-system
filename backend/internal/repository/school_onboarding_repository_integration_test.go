package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

// TestListPlatformSchoolsPostgresIntegration verifies the repository-level
// Platform Admin permission check and the school directory query against the
// migrated integration database. It is read-only and creates no fixtures.
func TestListPlatformSchoolsPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var platformAdminPrincipalID int64
	err := pool.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND r.is_system = TRUE
		   AND p.permission_key = 'platform.schools.read'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Skip("no active Platform Admin with platform.schools.read exists in the integration database")
	}
	if err != nil {
		t.Fatalf("find authorized Platform Admin fixture: %v", err)
	}

	var fixtureSchoolID, staffPrincipalID int64
	if err := pool.QueryRow(ctx, `
		SELECT school_id, principal_id
		  FROM users
		 WHERE id = $1
	`, actorUserID).Scan(&fixtureSchoolID, &staffPrincipalID); err != nil {
		t.Fatalf("load existing school/staff fixture: %v", err)
	}

	repo := NewSchoolOnboardingRepository(pool)
	schools, err := repo.ListPlatformSchools(ctx, platformAdminPrincipalID)
	if err != nil {
		t.Fatalf("authorized Platform Admin school listing failed: %v", err)
	}
	foundFixtureSchool := false
	for _, school := range schools {
		if school.ID == fixtureSchoolID {
			foundFixtureSchool = true
			if school.CreatedAt.IsZero() {
				t.Fatal("school listing omitted created_at")
			}
			break
		}
	}
	if !foundFixtureSchool {
		t.Fatalf("school directory did not include fixture school %d", fixtureSchoolID)
	}

	if _, err := repo.ListPlatformSchools(ctx, staffPrincipalID); !errors.Is(err, ErrPlatformSchoolListDenied) {
		t.Fatalf("School staff listing error = %v, want ErrPlatformSchoolListDenied", err)
	}
}
