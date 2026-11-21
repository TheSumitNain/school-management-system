package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPlatformAuditRepositoryPostgresIntegration(t *testing.T) {
	_, pool, staffUserID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	actorID := ensurePlatformAdminLifecycleActor(t, pool)
	repo := NewPlatformAuditRepository(pool)

	var staffPrincipalID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id FROM users WHERE id=$1`, staffUserID).Scan(&staffPrincipalID); err != nil {
		t.Fatalf("load staff principal: %v", err)
	}
	if _, err := repo.List(ctx, staffPrincipalID, 50, 0); !errors.Is(err, ErrPlatformAuditForbidden) {
		t.Fatalf("non-Platform Admin audit access error = %v, want forbidden", err)
	}
	if _, err := repo.List(ctx, actorID, 0, 0); !errors.Is(err, ErrInvalidPlatformAuditQuery) {
		t.Fatalf("invalid page size error = %v, want invalid query", err)
	}

	resourceA := fmt.Sprintf("audit-test-a-%d", time.Now().UnixNano())
	resourceB := resourceA + "-b"
	if _, err := pool.Exec(ctx, `
INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata)
VALUES ($1,'platform.audit.test','test_resource',$2,'success','{"sensitive":"must-not-be-returned"}'::jsonb),
       ($1,'platform.audit.test','test_resource',$3,'failure','{}'::jsonb)`, actorID, resourceA, resourceB); err != nil {
		t.Fatalf("insert platform audit fixtures: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome)
VALUES ($1,'school.student.profile.read','student',$2,'success')`, actorID, resourceA); err != nil {
		t.Fatalf("insert school audit fixture: %v", err)
	}
	page, err := repo.List(ctx, actorID, 100, 0)
	if err != nil {
		t.Fatalf("list platform audit events: %v", err)
	}
	found := map[string]PlatformAuditEvent{}
	for _, event := range page.Events {
		if event.ResourceID != nil {
			found[*event.ResourceID] = event
		}
		if len(event.Action) < len("platform.") || event.Action[:len("platform.")] != "platform." {
			t.Fatalf("non-platform audit action leaked into platform history: %q", event.Action)
		}
	}
	for _, id := range []string{resourceA, resourceB} {
		if _, ok := found[id]; !ok {
			t.Fatalf("platform audit event %q missing from list", id)
		}
	}
	if found[resourceA].ActorEmail == nil {
		t.Fatal("Platform Admin actor email should be included")
	}
	if _, err := repo.List(ctx, actorID, 1, 0); err != nil {
		t.Fatalf("first audit page failed: %v", err)
	}
}
