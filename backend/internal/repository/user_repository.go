package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrStaffPasswordConflict = errors.New("staff password changed concurrently")
	ErrUserNotFound   = errors.New("user not found")
	ErrUsernameTaken  = errors.New("username already exists")
	ErrInvalidUser    = errors.New("invalid user data")
	ErrStaffInactive  = errors.New("staff account is inactive")
	ErrSchoolInactive = errors.New("school is inactive")
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetUserByUsername(
	ctx context.Context,
	schoolID int64,
	username string,
) (*models.User, error) {
	if schoolID <= 0 || strings.TrimSpace(username) == "" {
		return nil, ErrInvalidUser
	}

	const query = `
		SELECT id, principal_id, school_id, username, password_hash,
		       status, must_change_password, created_at, updated_at
		FROM users
		WHERE school_id = $1
		  AND LOWER(username) = LOWER($2)
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, schoolID, strings.TrimSpace(username)).Scan(
		&user.ID,
		&user.PrincipalID,
		&user.SchoolID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	schoolID int64,
	username string,
	passwordHash string,
) (*models.User, error) {
	username = strings.TrimSpace(username)

	if schoolID <= 0 || username == "" || passwordHash == "" {
		return nil, ErrInvalidUser
	}

	const query = `
		INSERT INTO users (school_id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, principal_id, school_id, username, password_hash,
		          		          status, must_change_password, created_at, updated_at
	`

	var user models.User

	err := r.db.QueryRow(
		ctx,
		query,
		schoolID,
		username,
		passwordHash,
	).Scan(
		&user.ID,
		&user.PrincipalID,
		&user.SchoolID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
		&user.MustChangePassword,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "ux_users_username_global" {
			return nil, ErrUsernameTaken
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

// GetStaffPrincipalByUsername resolves a globally unique staff username to its
// canonical V4 principal and verifies the associated school is active.
func (r *UserRepository) GetStaffPrincipalByUsername(
	ctx context.Context,
	username string,
) (*models.AuthPrincipal, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrInvalidUser
	}

	const query = `
SELECT ap.id, ap.principal_type, ap.status, ap.token_version,
       u.school_id, u.username, u.status, s.status
FROM users u
JOIN auth_principals ap ON ap.id = u.principal_id
JOIN schools s ON s.id = u.school_id
WHERE LOWER(u.username) = LOWER($1)
  AND ap.principal_type = 'staff'
`

	var principal models.AuthPrincipal
	var userStatus string
	var schoolStatus string
	if err := r.db.QueryRow(ctx, query, username).Scan(
		&principal.ID,
		&principal.Type,
		&principal.Status,
		&principal.TokenVersion,
		&principal.SchoolID,
		&principal.Username,
		&userStatus,
		&schoolStatus,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get staff principal by username: %w", err)
	}

	if principal.Status != "active" || userStatus != "active" {
		return nil, ErrStaffInactive
	}
	if schoolStatus != "active" {
		return nil, ErrSchoolInactive
	}
	return &principal, nil
}


// GetStaffPasswordCredential returns the password hash for a canonical staff
// principal. Password hashes are only consumed by the password service.
func (r *UserRepository) GetStaffPasswordCredential(ctx context.Context, principalID int64) (int64, string, error) {
	if principalID <= 0 {
		return 0, "", ErrInvalidUser
	}
	var userID int64
	var passwordHash string
	err := r.db.QueryRow(ctx, `
		SELECT id, password_hash
		FROM users
		WHERE principal_id = $1
		  AND EXISTS (
		      SELECT 1 FROM auth_principals ap
		      WHERE ap.id = users.principal_id
		        AND ap.principal_type = 'staff'
		        AND ap.status = 'active'
		  )
	`, principalID).Scan(&userID, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrUserNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("load staff password credential: %w", err)
	}
	return userID, passwordHash, nil
}

// ChangeStaffPassword atomically changes the credential, clears the temporary
// password flag, increments token_version, revokes all refresh sessions, and
// appends an audit event. The expected hash prevents a stale password check
// from overwriting a concurrent credential change.
func (r *UserRepository) ChangeStaffPassword(
	ctx context.Context,
	principalID int64,
	expectedPasswordHash string,
	newPasswordHash string,
) error {
	if principalID <= 0 || expectedPasswordHash == "" || newPasswordHash == "" {
		return ErrInvalidUser
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin staff password change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		principalID,
	); err != nil {
		return fmt.Errorf("lock staff principal password change: %w", err)
	}

	var userID, schoolID int64
	err = tx.QueryRow(ctx, `
		UPDATE users
		   SET password_hash = $3,
		       must_change_password = FALSE,
		       updated_at = NOW()
		 WHERE principal_id = $1
		   AND password_hash = $2
		RETURNING id, school_id
	`, principalID, expectedPasswordHash, newPasswordHash).Scan(&userID, &schoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStaffPasswordConflict
	}
	if err != nil {
		return fmt.Errorf("update staff password: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE auth_principals
		   SET token_version = token_version + 1,
		       updated_at = NOW()
		 WHERE id = $1
		   AND principal_type = 'staff'
		   AND status = 'active'
	`, principalID)
	if err != nil {
		return fmt.Errorf("invalidate staff access tokens: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions
		   SET revoked_at = COALESCE(revoked_at, NOW())
		 WHERE principal_id = $1
		   AND revoked_at IS NULL
	`, principalID); err != nil {
		return fmt.Errorf("revoke staff refresh sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.password.changed', 'staff_user', $3, 'success',
		        jsonb_build_object('password_change_required_cleared', TRUE))
	`, principalID, schoolID, fmt.Sprint(userID)); err != nil {
		return fmt.Errorf("audit staff password change: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit staff password change: %w", err)
	}
	return nil
}


// ResetStaffPassword permits an active same-school School Admin with the
// users.password_reset grant to set another staff member's temporary password.
// It independently enforces school scope and authorization in the transaction.
func (r *UserRepository) ResetStaffPassword(
	ctx context.Context,
	actorPrincipalID int64,
	schoolID int64,
	targetPrincipalID int64,
	newPasswordHash string,
) error {
	if actorPrincipalID <= 0 || schoolID <= 0 || targetPrincipalID <= 0 ||
		actorPrincipalID == targetPrincipalID || newPasswordHash == "" {
		return ErrInvalidUser
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin staff password reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		targetPrincipalID,
	); err != nil {
		return fmt.Errorf("lock target staff principal password reset: %w", err)
	}

	var targetUserID int64
	err = tx.QueryRow(ctx, `
		SELECT target.id
		FROM users target
		JOIN auth_principals target_principal
		  ON target_principal.id = target.principal_id
		WHERE target.principal_id = $1
		  AND target.school_id = $2
		  AND target_principal.principal_type = 'staff'
		  AND target_principal.status = 'active'
		  AND target.status = 'active'
		  AND EXISTS (
		      SELECT 1
		      FROM users actor
		      JOIN auth_principals actor_principal
		        ON actor_principal.id = actor.principal_id
		      JOIN principal_roles pr
		        ON pr.principal_id = actor.principal_id
		       AND pr.revoked_at IS NULL
		      JOIN roles role
		        ON role.id = pr.role_id
		       AND role.school_id = actor.school_id
		       AND role.scope_type = 'school'
		       AND role.principal_type = 'staff'
		       AND role.role_key = 'school_admin'
		       AND role.is_system = TRUE
		      JOIN role_permissions rp ON rp.role_id = role.id
		      JOIN permissions permission ON permission.id = rp.permission_id
		      WHERE actor.principal_id = $3
		        AND actor.school_id = $2
		        AND actor.status = 'active'
		        AND actor_principal.status = 'active'
		        AND actor_principal.principal_type = 'staff'
		        AND permission.permission_key = 'users.password_reset'
		        AND permission.scope_type = 'school'
		        AND permission.is_active = TRUE
		        AND NOT EXISTS (
		            SELECT 1
		            FROM user_permission_overrides override
		            WHERE override.principal_id = actor.principal_id
		              AND override.school_id = $2
		              AND override.permission_id = permission.id
		              AND override.effect = 'deny'
		        )
		  )
		FOR UPDATE OF target
	`, targetPrincipalID, schoolID, actorPrincipalID).Scan(&targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("authorize same-school staff password reset: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE users
		   SET password_hash = $2,
		       must_change_password = TRUE,
		       updated_at = NOW()
		 WHERE id = $1
	`, targetUserID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("update reset staff password: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	tag, err = tx.Exec(ctx, `
		UPDATE auth_principals
		   SET token_version = token_version + 1,
		       updated_at = NOW()
		 WHERE id = $1
		   AND principal_type = 'staff'
		   AND status = 'active'
	`, targetPrincipalID)
	if err != nil {
		return fmt.Errorf("invalidate reset staff access tokens: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions
		   SET revoked_at = COALESCE(revoked_at, NOW())
		 WHERE principal_id = $1
		   AND revoked_at IS NULL
	`, targetPrincipalID); err != nil {
		return fmt.Errorf("revoke reset staff refresh sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.password.reset', 'staff_user', $3, 'success',
		        jsonb_build_object('must_change_password', TRUE))
	`, actorPrincipalID, schoolID, fmt.Sprint(targetUserID)); err != nil {
		return fmt.Errorf("audit staff password reset: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit staff password reset: %w", err)
	}
	return nil
}
