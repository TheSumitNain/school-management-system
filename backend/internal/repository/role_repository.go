package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRoleNotFound = errors.New("school role not found")
	ErrRoleNameTaken = errors.New("school role key already exists")
	ErrRolePermissionDenied = errors.New("school role operation is not authorized")
	ErrDelegationNotFound = errors.New("school permission delegation not found")
	ErrDelegationAlreadyActive = errors.New("an active delegation already exists for this school permission")
	ErrRolePermissionAlreadyGranted = errors.New("permission is already granted to this custom role")
)

type SchoolRole struct {
	ID        int64     `json:"id"`
	RoleKey   string    `json:"role_key"`
	Name      string    `json:"name"`
	IsSystem  bool      `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
}

type SchoolPermission struct {
	Key         string `json:"permission_key"`
	Description string `json:"description"`
	Delegable   bool   `json:"is_delegable"`
}

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) ListSchoolRoles(ctx context.Context, actorPrincipalID, schoolID int64) ([]SchoolRole, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrRolePermissionDenied
	}
	var active bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			WHERE ap.id = $1
			  AND ap.principal_type = 'staff'
			  AND ap.status = 'active'
			  AND u.status = 'active'
			  AND u.school_id = $2
			  AND s.status = 'active'
		)
	`, actorPrincipalID, schoolID).Scan(&active); err != nil {
		return nil, fmt.Errorf("validate school role list actor: %w", err)
	}
	if !active {
		return nil, ErrRolePermissionDenied
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, role_key, name, is_system, created_at
		FROM roles
		WHERE school_id = $1
		  AND scope_type = 'school'
		  AND principal_type = 'staff'
		ORDER BY is_system DESC, LOWER(name), id
	`, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list school roles: %w", err)
	}
	defer rows.Close()

	roles := make([]SchoolRole, 0)
	for rows.Next() {
		var role SchoolRole
		if err := rows.Scan(&role.ID, &role.RoleKey, &role.Name, &role.IsSystem, &role.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan school role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate school roles: %w", err)
	}
	return roles, nil
}

func (r *RoleRepository) ListSchoolPermissions(ctx context.Context) ([]SchoolPermission, error) {
	if r == nil || r.db == nil {
		return nil, ErrRolePermissionDenied
	}
	rows, err := r.db.Query(ctx, `
		SELECT permission_key, description, is_delegable
		FROM permissions
		WHERE scope_type = 'school' AND is_active = TRUE
		ORDER BY permission_key
	`)
	if err != nil {
		return nil, fmt.Errorf("list school permission catalogue: %w", err)
	}
	defer rows.Close()

	permissions := make([]SchoolPermission, 0)
	for rows.Next() {
		var permission SchoolPermission
		if err := rows.Scan(&permission.Key, &permission.Description, &permission.Delegable); err != nil {
			return nil, fmt.Errorf("scan school permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate school permission catalogue: %w", err)
	}
	return permissions, nil
}

func (r *RoleRepository) CreateCustomRole(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	roleKey, name string,
) (*SchoolRole, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		strings.TrimSpace(roleKey) == "" || strings.TrimSpace(name) == "" {
		return nil, ErrRolePermissionDenied
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin custom role creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1
			  AND ap.principal_type = 'staff'
			  AND ap.status = 'active'
			  AND u.status = 'active'
			  AND u.school_id = $2
			  AND s.status = 'active'
			  AND r.school_id = $2
			  AND r.scope_type = 'school'
			  AND r.principal_type = 'staff'
			  AND r.role_key = 'school_admin'
			  AND r.is_system = TRUE
			  AND p.permission_key = 'roles.create_custom'
			  AND p.scope_type = 'school'
			  AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1
			      FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id
			        AND denied.school_id = $2
			        AND denied.permission_id = p.id
			        AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return nil, fmt.Errorf("authorize custom role creation: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}

	var role SchoolRole
	err = tx.QueryRow(ctx, `
		INSERT INTO roles (school_id, principal_type, scope_type, role_key, name, is_system, created_by)
		VALUES ($1, 'staff', 'school', $2, $3, FALSE, $4)
		RETURNING id, role_key, name, is_system, created_at
	`, schoolID, roleKey, name, actorPrincipalID).Scan(
		&role.ID, &role.RoleKey, &role.Name, &role.IsSystem, &role.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_roles_school_key" {
			return nil, ErrRoleNameTaken
		}
		return nil, fmt.Errorf("insert custom school role: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.role.created', 'school_role', $3, 'success',
		        jsonb_build_object('role_key', $4, 'name', $5))
	`, actorPrincipalID, schoolID, fmt.Sprint(role.ID), role.RoleKey, role.Name); err != nil {
		return nil, fmt.Errorf("audit custom school role creation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit custom school role creation: %w", err)
	}
	return &role, nil
}

type SchoolPermissionDelegation struct {
	ID                 int64     `json:"id"`
	SchoolID           int64     `json:"school_id"`
	PermissionKey      string    `json:"permission_key"`
	CanAssignToRoles   bool      `json:"can_assign_to_roles"`
	CanOverrideUsers   bool      `json:"can_override_users"`
	GrantedBy          int64     `json:"granted_by"`
	CreatedAt          time.Time `json:"created_at"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
}

// GrantSchoolPermissionDelegation gives a Platform Admin an audited way to
// delegate only explicitly delegable school permissions. The exact inserted
// delegation ID must accompany any later custom-role or ALLOW grant.
func (r *RoleRepository) GrantSchoolPermissionDelegation(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	permissionKey string,
	canAssignToRoles, canOverrideUsers bool,
) (*SchoolPermissionDelegation, error) {
	permissionKey = strings.TrimSpace(permissionKey)
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		permissionKey == "" || (!canAssignToRoles && !canOverrideUsers) {
		return nil, ErrRolePermissionDenied
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin permission delegation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1
			  AND ap.principal_type = 'platform_admin'
			  AND ap.status = 'active'
			  AND r.scope_type = 'platform'
			  AND r.school_id IS NULL
			  AND r.principal_type = 'platform_admin'
			  AND r.role_key = 'platform_admin'
			  AND p.permission_key = 'platform.permissions.delegate'
			  AND p.scope_type = 'platform'
			  AND p.is_active = TRUE
		)
	`, actorPrincipalID).Scan(&authorized); err != nil {
		return nil, fmt.Errorf("authorize permission delegation: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}

	var delegation SchoolPermissionDelegation
	err = tx.QueryRow(ctx, `
		INSERT INTO role_permission_delegations
		    (school_id, permission_id, can_assign_to_roles, can_override_users, granted_by)
		SELECT s.id, p.id, $3, $4, $5
		FROM schools s
		JOIN permissions p ON p.permission_key = $2
		WHERE s.id = $1
		  AND s.status = 'active'
		  AND p.scope_type = 'school'
		  AND p.is_active = TRUE
		  AND p.is_delegable = TRUE
		RETURNING id, school_id, $2::text, can_assign_to_roles, can_override_users, granted_by, created_at, revoked_at
	`, schoolID, permissionKey, canAssignToRoles, canOverrideUsers, actorPrincipalID).Scan(
		&delegation.ID, &delegation.SchoolID, &delegation.PermissionKey,
		&delegation.CanAssignToRoles, &delegation.CanOverrideUsers,
		&delegation.GrantedBy, &delegation.CreatedAt, &delegation.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_role_permission_delegations_active" {
			return nil, ErrDelegationAlreadyActive
		}
		return nil, fmt.Errorf("insert permission delegation: %w", err)
	}
	if delegation.ID <= 0 {
		return nil, ErrRoleNotFound
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'platform.permission_delegation.granted', 'permission_delegation', $3, 'success',
		        jsonb_build_object('permission_key', $4, 'can_assign_to_roles', $5, 'can_override_users', $6))
	`, actorPrincipalID, schoolID, fmt.Sprint(delegation.ID), permissionKey, canAssignToRoles, canOverrideUsers); err != nil {
		return nil, fmt.Errorf("audit permission delegation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit permission delegation: %w", err)
	}
	return &delegation, nil
}

func (r *RoleRepository) RevokeSchoolPermissionDelegation(
	ctx context.Context,
	actorPrincipalID, delegationID int64,
) error {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || delegationID <= 0 {
		return ErrRolePermissionDenied
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delegation revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1
			  AND ap.principal_type = 'platform_admin'
			  AND ap.status = 'active'
			  AND r.scope_type = 'platform'
			  AND r.school_id IS NULL
			  AND r.principal_type = 'platform_admin'
			  AND r.role_key = 'platform_admin'
			  AND p.permission_key = 'platform.permissions.delegate'
			  AND p.scope_type = 'platform'
			  AND p.is_active = TRUE
		)
	`, actorPrincipalID).Scan(&authorized); err != nil {
		return fmt.Errorf("authorize delegation revocation: %w", err)
	}
	if !authorized {
		return ErrRolePermissionDenied
	}

	var schoolID int64
	var permissionKey string
	err = tx.QueryRow(ctx, `
		UPDATE role_permission_delegations d
		SET revoked_at = NOW()
		FROM permissions p
		WHERE d.id = $1
		  AND d.revoked_at IS NULL
		  AND p.id = d.permission_id
		RETURNING d.school_id, p.permission_key
	`, delegationID).Scan(&schoolID, &permissionKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDelegationNotFound
	}
	if err != nil {
		return fmt.Errorf("revoke permission delegation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'platform.permission_delegation.revoked', 'permission_delegation', $3, 'success',
		        jsonb_build_object('permission_key', $4))
	`, actorPrincipalID, schoolID, fmt.Sprint(delegationID), permissionKey); err != nil {
		return fmt.Errorf("audit permission delegation revocation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit permission delegation revocation: %w", err)
	}
	return nil
}

func (r *RoleRepository) ListSchoolPermissionDelegations(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
) ([]SchoolPermissionDelegation, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrRolePermissionDenied
	}
	var authorized bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff'
			  AND ap.status = 'active' AND u.status = 'active'
			  AND u.school_id = $2 AND s.id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school'
			  AND r.principal_type = 'staff' AND r.role_key = 'school_admin'
			  AND p.permission_key = 'permissions.assign'
			  AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id
			        AND denied.school_id = $2
			        AND denied.permission_id = p.id
			        AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return nil, fmt.Errorf("authorize school delegation listing: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}

	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.school_id, p.permission_key,
		       d.can_assign_to_roles, d.can_override_users,
		       d.granted_by, d.created_at, d.revoked_at
		FROM role_permission_delegations d
		JOIN permissions p ON p.id = d.permission_id
		WHERE d.school_id = $1 AND d.revoked_at IS NULL
		ORDER BY p.permission_key, d.id
	`, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list school permission delegations: %w", err)
	}
	defer rows.Close()
	delegations := make([]SchoolPermissionDelegation, 0)
	for rows.Next() {
		var item SchoolPermissionDelegation
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.PermissionKey,
			&item.CanAssignToRoles, &item.CanOverrideUsers, &item.GrantedBy,
			&item.CreatedAt, &item.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan school permission delegation: %w", err)
		}
		delegations = append(delegations, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate school permission delegations: %w", err)
	}
	return delegations, nil
}

func (r *RoleRepository) GrantCustomRolePermission(
	ctx context.Context,
	actorPrincipalID, schoolID, roleID, delegationID int64,
	permissionKey string,
) error {
	permissionKey = strings.TrimSpace(permissionKey)
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		roleID <= 0 || delegationID <= 0 || permissionKey == "" {
		return ErrRolePermissionDenied
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin custom role permission grant: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff'
			  AND ap.status = 'active' AND u.status = 'active'
			  AND u.school_id = $2 AND s.id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school'
			  AND r.principal_type = 'staff' AND r.role_key = 'school_admin'
			  AND p.permission_key = 'permissions.assign'
			  AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id
			        AND denied.school_id = $2
			        AND denied.permission_id = p.id
			        AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return fmt.Errorf("authorize custom role permission grant: %w", err)
	}
	if !authorized {
		return ErrRolePermissionDenied
	}

	var insertedRoleID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, delegation_id)
		SELECT target.id, p.id, d.id
		FROM roles target
		JOIN permissions p ON p.permission_key = $4
		JOIN role_permission_delegations d
		  ON d.id = $3
		 AND d.school_id = $2
		 AND d.permission_id = p.id
		 AND d.can_assign_to_roles = TRUE
		 AND d.revoked_at IS NULL
		WHERE target.id = $1
		  AND target.school_id = $2
		  AND target.scope_type = 'school'
		  AND target.principal_type = 'staff'
		  AND target.is_system = FALSE
		  AND p.scope_type = 'school'
		  AND p.is_active = TRUE
		  AND p.is_delegable = TRUE
		RETURNING role_id
	`, roleID, schoolID, delegationID, permissionKey).Scan(&insertedRoleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDelegationNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrRolePermissionAlreadyGranted
		}
		return fmt.Errorf("insert custom role permission grant: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.custom_role.permission_granted', 'school_role', $3, 'success',
		        jsonb_build_object('permission_key', $4, 'delegation_id', $5))
	`, actorPrincipalID, schoolID, fmt.Sprint(insertedRoleID), permissionKey, delegationID); err != nil {
		return fmt.Errorf("audit custom role permission grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit custom role permission grant: %w", err)
	}
	return nil
}

func (r *RoleRepository) RevokeCustomRolePermission(
	ctx context.Context,
	actorPrincipalID, schoolID, roleID int64,
	permissionKey string,
) error {
	permissionKey = strings.TrimSpace(permissionKey)
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 || roleID <= 0 || permissionKey == "" {
		return ErrRolePermissionDenied
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin custom role permission revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff'
			  AND ap.status = 'active' AND u.status = 'active'
			  AND u.school_id = $2 AND s.id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school'
			  AND r.principal_type = 'staff' AND r.role_key = 'school_admin'
			  AND p.permission_key = 'permissions.assign'
			  AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id
			        AND denied.school_id = $2
			        AND denied.permission_id = p.id
			        AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return fmt.Errorf("authorize custom role permission revocation: %w", err)
	}
	if !authorized {
		return ErrRolePermissionDenied
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM role_permissions rp
		USING roles target, permissions p
		WHERE rp.role_id = target.id
		  AND rp.permission_id = p.id
		  AND target.id = $1
		  AND target.school_id = $2
		  AND target.scope_type = 'school'
		  AND target.principal_type = 'staff'
		  AND target.is_system = FALSE
		  AND p.permission_key = $3
		  AND rp.delegation_id IS NOT NULL
	`, roleID, schoolID, permissionKey)
	if err != nil {
		return fmt.Errorf("delete custom role permission grant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDelegationNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.custom_role.permission_revoked', 'school_role', $3, 'success',
		        jsonb_build_object('permission_key', $4))
	`, actorPrincipalID, schoolID, fmt.Sprint(roleID), permissionKey); err != nil {
		return fmt.Errorf("audit custom role permission revocation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit custom role permission revocation: %w", err)
	}
	return nil
}
