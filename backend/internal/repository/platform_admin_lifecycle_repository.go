package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlatformAdminActorForbidden = errors.New("active Platform Admin with lifecycle permission required")
	ErrPlatformAdminSelfDeactivation = errors.New("Platform Admin cannot deactivate their own account")
	ErrPlatformAdminTargetNotFound = errors.New("active Platform Admin target not found")
	ErrPlatformAdminLastAccount = errors.New("cannot deactivate the last active Platform Admin")
)

type PlatformAdminAccount struct {
	PrincipalID int64 `json:"principal_id"`
	Email string `json:"email"`
	Status string `json:"status"`
	MFAEnrolled bool `json:"mfa_enrolled"`
	CreatedAt time.Time `json:"created_at"`
}

type PlatformAdminLifecycleRepository struct { db *pgxpool.Pool }

func NewPlatformAdminLifecycleRepository(db *pgxpool.Pool) *PlatformAdminLifecycleRepository {
	return &PlatformAdminLifecycleRepository{db: db}
}

const platformAdminLifecycleActorQuery = `
SELECT EXISTS (
 SELECT 1 FROM auth_principals ap
 JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
 JOIN roles r ON r.id=pr.role_id
 JOIN role_permissions rp ON rp.role_id=r.id
 JOIN permissions p ON p.id=rp.permission_id
 WHERE ap.id=$1 AND ap.principal_type='platform_admin' AND ap.status='active'
 AND r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin'
 AND p.permission_key='platform.admins.manage' AND p.scope_type='platform' AND p.is_active=TRUE
)`

func (r *PlatformAdminLifecycleRepository) List(ctx context.Context, actorID int64) ([]PlatformAdminAccount, error) {
	if r == nil || r.db == nil || actorID <= 0 { return nil, ErrPlatformAdminActorForbidden }
	var allowed bool
	if err := r.db.QueryRow(ctx, platformAdminLifecycleActorQuery, actorID).Scan(&allowed); err != nil { return nil, fmt.Errorf("authorize Platform Admin listing: %w", err) }
	if !allowed { return nil, ErrPlatformAdminActorForbidden }
	rows, err := r.db.Query(ctx, `
SELECT ap.id, pa.email, ap.status,
 EXISTS (SELECT 1 FROM principal_mfa_credentials mc WHERE mc.principal_id=ap.id AND mc.method='totp' AND mc.verified_at IS NOT NULL),
 pa.created_at
FROM platform_admin_accounts pa JOIN auth_principals ap ON ap.id=pa.principal_id
ORDER BY pa.created_at, ap.id`)
	if err != nil { return nil, fmt.Errorf("list Platform Admin accounts: %w", err) }
	defer rows.Close()
	accounts := make([]PlatformAdminAccount, 0)
	for rows.Next() {
		var account PlatformAdminAccount
		if err := rows.Scan(&account.PrincipalID,&account.Email,&account.Status,&account.MFAEnrolled,&account.CreatedAt); err != nil { return nil, fmt.Errorf("scan Platform Admin account: %w", err) }
		accounts=append(accounts,account)
	}
	if err:=rows.Err(); err!=nil { return nil, fmt.Errorf("iterate Platform Admin accounts: %w",err) }
	return accounts,nil
}

func (r *PlatformAdminLifecycleRepository) Create(ctx context.Context, actorID int64, email, passwordHash string) (*PlatformAdminAccount,error) {
	email=strings.TrimSpace(strings.ToLower(email))
	if r==nil || r.db==nil || actorID<=0 || email=="" || len(email)>254 || passwordHash=="" { return nil,fmt.Errorf("invalid Platform Admin account input") }
	tx,err:=r.db.Begin(ctx); if err!=nil { return nil,fmt.Errorf("begin Platform Admin creation: %w",err) }
	defer func(){ _=tx.Rollback(ctx) }()
	if _,err=tx.Exec(ctx,`SELECT pg_advisory_xact_lock(7414,2)`); err!=nil { return nil,fmt.Errorf("lock Platform Admin lifecycle: %w",err) }
	var allowed bool
	if err=tx.QueryRow(ctx,platformAdminLifecycleActorQuery,actorID).Scan(&allowed); err!=nil { return nil,fmt.Errorf("authorize Platform Admin creation: %w",err) }
	if !allowed { return nil,ErrPlatformAdminActorForbidden }
	var principalID int64
	if err=tx.QueryRow(ctx,`INSERT INTO auth_principals(principal_type,status) VALUES('platform_admin','active') RETURNING id`).Scan(&principalID); err!=nil { return nil,fmt.Errorf("create Platform Admin principal: %w",err) }
	if _,err=tx.Exec(ctx,`INSERT INTO platform_admin_accounts(principal_id,email,password_hash,must_change_password) VALUES($1,$2,$3,FALSE)`,principalID,email,passwordHash); err!=nil { return nil,mapPlatformAdminLifecycleError(err) }
	var roleID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM roles WHERE principal_type='platform_admin' AND scope_type='platform' AND role_key='platform_admin' AND is_system=TRUE`).Scan(&roleID); err!=nil { return nil,fmt.Errorf("load Platform Admin system role: %w",err) }
	if _,err=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,principalID,roleID,actorID); err!=nil { return nil,fmt.Errorf("assign Platform Admin system role: %w",err) }
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,'platform.admin.created','platform_admin',$2,'success',jsonb_build_object('email',$3::text,'mfa_enrollment_required',TRUE))`,actorID,fmt.Sprint(principalID),email); err!=nil { return nil,fmt.Errorf("audit Platform Admin creation: %w",err) }
	if err=tx.Commit(ctx); err!=nil { return nil,fmt.Errorf("commit Platform Admin creation: %w",err) }
	return &PlatformAdminAccount{PrincipalID:principalID,Email:email,Status:"active",MFAEnrolled:false},nil
}

func (r *PlatformAdminLifecycleRepository) Deactivate(ctx context.Context, actorID,targetID int64) error {
	if r==nil || r.db==nil || actorID<=0 || targetID<=0 { return ErrPlatformAdminTargetNotFound }
	if actorID==targetID { return ErrPlatformAdminSelfDeactivation }
	tx,err:=r.db.Begin(ctx); if err!=nil { return fmt.Errorf("begin Platform Admin deactivation: %w",err) }
	defer func(){ _=tx.Rollback(ctx) }()
	if _,err=tx.Exec(ctx,`SELECT pg_advisory_xact_lock(7414,2)`); err!=nil { return fmt.Errorf("lock Platform Admin lifecycle: %w",err) }
	var allowed bool
	if err=tx.QueryRow(ctx,platformAdminLifecycleActorQuery,actorID).Scan(&allowed); err!=nil { return fmt.Errorf("authorize Platform Admin deactivation: %w",err) }
	if !allowed { return ErrPlatformAdminActorForbidden }
	// Match the principal-level lock used by refresh-session creation and rotation.
	// Acquiring it before touching auth_principals or refresh_sessions prevents a
	// concurrent rotation from issuing a replacement token during deactivation.
	if _,err=tx.Exec(ctx,`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,targetID); err!=nil { return fmt.Errorf("lock Platform Admin refresh sessions: %w",err) }
	tag,err:=tx.Exec(ctx,`UPDATE auth_principals ap SET status='disabled',token_version=token_version+1,updated_at=NOW()
WHERE ap.id=$1 AND ap.principal_type='platform_admin' AND ap.status='active'
AND EXISTS(SELECT 1 FROM platform_admin_accounts pa WHERE pa.principal_id=ap.id)`,targetID)
	if err!=nil {
		var pgErr *pgconn.PgError
		if errors.As(err,&pgErr) && strings.Contains(pgErr.Message,"last active Platform Admin") { return ErrPlatformAdminLastAccount }
		return fmt.Errorf("disable Platform Admin principal: %w",err)
	}
	if tag.RowsAffected()!=1 { return ErrPlatformAdminTargetNotFound }
	if _,err=tx.Exec(ctx,`UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,NOW()) WHERE principal_id=$1 AND revoked_at IS NULL`,targetID); err!=nil { return fmt.Errorf("revoke Platform Admin refresh sessions: %w",err) }
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,'platform.admin.deactivated','platform_admin',$2,'success',jsonb_build_object('sessions_revoked',TRUE))`,actorID,fmt.Sprint(targetID)); err!=nil { return fmt.Errorf("audit Platform Admin deactivation: %w",err) }
	if err=tx.Commit(ctx); err!=nil { return fmt.Errorf("commit Platform Admin deactivation: %w",err) }
	return nil
}

func mapPlatformAdminLifecycleError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err,&pgErr) && pgErr.Code=="23505" && pgErr.ConstraintName=="ux_platform_admin_email" { return ErrPlatformAdminEmailTaken }
	return fmt.Errorf("create Platform Admin account: %w",err)
}
