package repository

import (
	"context"
	"testing"
	"time"
)

// Exercise the production pool-backed path: each report must establish its
// own read-only repeatable-read transaction before resolving permissions or
// reading aggregate counts and page rows.
func TestStaffReportsUsePoolBackedReadOnlySnapshotPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var actorID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture: %v", err)
	}

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
