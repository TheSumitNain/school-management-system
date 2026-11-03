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
	ErrRoleNotFound = errors.New("school role not found")
	ErrRoleNameTaken = errors.New("school role key already exists")
	ErrRolePermissionDenied = errors.New("school role operation is not authorized")
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
