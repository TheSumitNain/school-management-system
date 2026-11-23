package repository

import (
	"context"
	"errors"
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
