package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Exercise the production pool-backed path: each report must establish its
// own read-only repeatable-read transaction before resolving permissions or
// reading aggregate counts and page rows.
func TestStaffReportsUsePoolBackedReadOnlySnapshotPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin report authorization fixture: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var actorID, schoolID int64
	if err := tx.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture: %v", err)
	}
	var hasRole bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id WHERE pr.principal_id=$1 AND pr.revoked_at IS NULL AND r.school_id=$2 AND r.role_key='school_admin' AND r.is_system=TRUE)`, actorID, schoolID).Scan(&hasRole); err != nil {
		t.Fatalf("check School Admin fixture role: %v", err)
	}
	if !hasRole {
		var platformAdminID, schoolAdminRoleID int64
		err = tx.QueryRow(ctx, `SELECT id FROM auth_principals WHERE principal_type='platform_admin' AND status='active' ORDER BY id LIMIT 1`).Scan(&platformAdminID)
		if errors.Is(err, pgx.ErrNoRows) { t.Skip("an active Platform Admin is required for this fixture") }
		if err != nil { t.Fatalf("load active Platform Admin: %v", err) }
		if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff' AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`, schoolID).Scan(&schoolAdminRoleID); err != nil {
			t.Fatalf("load School Admin role: %v", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, actorID, schoolAdminRoleID, platformAdminID); err != nil {
			t.Fatalf("assign School Admin fixture role: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil { t.Fatalf("commit report authorization fixture: %v", err) }
	repo := NewUserRepository(pool)
	qualifications, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{
		RecordType: "all",
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("list pool-backed qualifications report: %v", err)
	}
	var qualificationCount int64
	for _, item := range qualifications.Counts {
		qualificationCount += item.Count
	}
	if qualificationCount != qualifications.Total || int64(len(qualifications.Rows)) > qualifications.Total || len(qualifications.Rows) > 1 {
		t.Fatalf("qualifications report totals are inconsistent: %+v", qualifications)
	}

	experience, err := repo.ListStaffExperienceReport(ctx, actorID, schoolID, StaffExperienceReportFilter{
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("list pool-backed experience report: %v", err)
	}
	var experienceCount int64
	for _, item := range experience.Counts {
		experienceCount += item.Count
	}
	if experienceCount != experience.Total || int64(len(experience.Rows)) > experience.Total || len(experience.Rows) > 1 {
		t.Fatalf("experience report totals are inconsistent: %+v", experience)
	}
}
