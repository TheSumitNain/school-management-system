package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestStaffSubstitutionCreateAuthorizesBeforeValidatingInputPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).
		Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load ordinary staff actor: %v", err)
	}

	_, err := NewAcademicRepository(pool).CreateStaffSubstitution(
		ctx, actorPrincipalID, schoolID, 0, 0, 0, "not-a-date", nil, "",
	)
	if !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("unauthorized actor should be denied before invalid input is evaluated; got %v", err)
	}
}

func TestStaffSubstitutionCancelAndListAuthorizeBeforeValidatingIDsPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load ordinary staff actor: %v", err)
	}
	repo := NewAcademicRepository(pool)
	if err := repo.CancelStaffSubstitution(ctx, actorPrincipalID, schoolID, 0); !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("unauthorized cancellation with invalid ID should be denied first; got %v", err)
	}
	if _, err := repo.ListStaffSubstitutions(ctx, actorPrincipalID, schoolID, "2028-01-01", "2028-01-31", -1); !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("unauthorized list with invalid staff ID should be denied first; got %v", err)
	}
}

func TestStaffSubstitutionListAuthorizesBeforeValidatingDateFiltersPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).
		Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load ordinary staff actor: %v", err)
	}

	_, err := NewAcademicRepository(pool).ListStaffSubstitutions(
		ctx, actorPrincipalID, schoolID, "not-a-date", "", 0,
	)
	if !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("unauthorized actor should be denied before date filters are validated; got %v", err)
	}
}

func TestStaffSubstitutionOperationsRejectForeignSchoolScopePostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var actorPrincipalID, actorSchoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).
		Scan(&actorPrincipalID, &actorSchoolID); err != nil {
		t.Fatalf("load actor fixture: %v", err)
	}

	var foreignSchoolID int64
	suffix := time.Now().UnixNano()
	if err := pool.QueryRow(ctx, `INSERT INTO schools(name, code, status)
		VALUES($1, $2, 'active') RETURNING id`,
		fmt.Sprintf("Substitution Isolation School %d", suffix),
		fmt.Sprintf("SI-%d", suffix)).Scan(&foreignSchoolID); err != nil {
		t.Fatalf("create second school fixture: %v", err)
	}
	if foreignSchoolID == actorSchoolID {
		t.Fatal("foreign school fixture unexpectedly matches actor school")
	}

	repo := NewAcademicRepository(pool)
	if _, err := repo.CreateStaffSubstitution(ctx, actorPrincipalID, foreignSchoolID,
		1, 1, 2, "2027-08-16", nil, "cross-school attempt"); !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("cross-school create error=%v; want access denied", err)
	}
	if err := repo.CancelStaffSubstitution(ctx, actorPrincipalID, foreignSchoolID, 1); !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("cross-school cancel error=%v; want access denied", err)
	}
	if _, err := repo.ListStaffSubstitutions(ctx, actorPrincipalID, foreignSchoolID,
		"2027-08-16", "2027-08-17", 0); !errors.Is(err, ErrStaffSubstitutionAccessDenied) {
		t.Fatalf("cross-school list error=%v; want access denied", err)
	}
}
