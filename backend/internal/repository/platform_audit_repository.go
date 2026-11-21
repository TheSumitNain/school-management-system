package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlatformAuditForbidden = errors.New("Platform Admin audit access is not authorized")
	ErrInvalidPlatformAuditQuery = errors.New("invalid Platform Admin audit query")
)

type PlatformAuditEvent struct {
	ID int64 `json:"id"`
	ActorEmail *string `json:"actor_email,omitempty"`
	SchoolID *int64 `json:"school_id,omitempty"`
	SchoolName *string `json:"school_name,omitempty"`
	SchoolCode *string `json:"school_code,omitempty"`
	Action string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID *string `json:"resource_id,omitempty"`
	Outcome string `json:"outcome"`
	CreatedAt time.Time `json:"created_at"`
}

type PlatformAuditPage struct {
	Events []PlatformAuditEvent `json:"events"`
	Total int64 `json:"total"`
}

type PlatformAuditRepository struct { db *pgxpool.Pool }

func NewPlatformAuditRepository(db *pgxpool.Pool) *PlatformAuditRepository {
	return &PlatformAuditRepository{db: db}
}

const platformAuditActorQuery = `
SELECT EXISTS (
 SELECT 1
 FROM auth_principals ap
 JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
 JOIN roles r ON r.id=pr.role_id
 JOIN role_permissions rp ON rp.role_id=r.id
 JOIN permissions p ON p.id=rp.permission_id
 WHERE ap.id=$1 AND ap.principal_type='platform_admin' AND ap.status='active'
   AND r.principal_type='platform_admin' AND r.scope_type='platform'
   AND r.role_key='platform_admin' AND r.is_system=TRUE
   AND p.permission_key='platform.audit.read' AND p.scope_type='platform' AND p.is_active=TRUE
)`

func (r *PlatformAuditRepository) List(ctx context.Context, actorID int64, limit, offset int) (*PlatformAuditPage, error) {
	if r == nil || r.db == nil || actorID <= 0 {
		return nil, ErrPlatformAuditForbidden
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, ErrInvalidPlatformAuditQuery
	}
	var allowed bool
	if err := r.db.QueryRow(ctx, platformAuditActorQuery, actorID).Scan(&allowed); err != nil {
		return nil, fmt.Errorf("authorize Platform Admin audit listing: %w", err)
	}
	if !allowed {
		return nil, ErrPlatformAuditForbidden
	}

	const filter = `ae.action LIKE 'platform.%'
 AND (ae.actor_principal_id IS NULL OR EXISTS (
   SELECT 1 FROM auth_principals actor
   WHERE actor.id=ae.actor_principal_id AND actor.principal_type='platform_admin'
 ))`
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_events ae WHERE "+filter).Scan(&total); err != nil {
		return nil, fmt.Errorf("count Platform Admin audit events: %w", err)
	}
	rows, err := r.db.Query(ctx, `
SELECT ae.id, pa.email, ae.school_id, s.name, s.code,
       ae.action, ae.resource_type, ae.resource_id, ae.outcome, ae.created_at
FROM audit_events ae
LEFT JOIN platform_admin_accounts pa ON pa.principal_id=ae.actor_principal_id
LEFT JOIN schools s ON s.id=ae.school_id
WHERE `+filter+`
ORDER BY ae.created_at DESC, ae.id DESC
LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list Platform Admin audit events: %w", err)
	}
	defer rows.Close()
	page := &PlatformAuditPage{Events: make([]PlatformAuditEvent, 0), Total: total}
	for rows.Next() {
		var event PlatformAuditEvent
		var actorEmail, schoolName, schoolCode, resourceID pgtype.Text
		var schoolID pgtype.Int8
		if err := rows.Scan(&event.ID, &actorEmail, &schoolID, &schoolName, &schoolCode,
			&event.Action, &event.ResourceType, &resourceID, &event.Outcome, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Platform Admin audit event: %w", err)
		}
		if actorEmail.Valid { event.ActorEmail = &actorEmail.String }
		if schoolID.Valid { event.SchoolID = &schoolID.Int64 }
		if schoolName.Valid { event.SchoolName = &schoolName.String }
		if schoolCode.Valid { event.SchoolCode = &schoolCode.String }
		if resourceID.Valid { event.ResourceID = &resourceID.String }
		page.Events = append(page.Events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Platform Admin audit events: %w", err)
	}
	return page, nil
}

var _ = pgx.ErrNoRows
