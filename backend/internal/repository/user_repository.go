package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrStaffPasswordConflict = errors.New("staff password changed concurrently")
	ErrUserNotFound   = errors.New("user not found")
	ErrUsernameTaken  = errors.New("username already exists")
	ErrInvalidUser    = errors.New("invalid user data")
	ErrStaffInactive  = errors.New("staff account is inactive")
	ErrSchoolInactive = errors.New("school is inactive")
    ErrEmployeeCodeTaken = errors.New("employee code already exists in this school")
)

type UserRepository struct {
	db repositoryDB
}

func NewUserRepository(db repositoryDB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) ListStaffAccounts(ctx context.Context, actorPrincipalID, schoolID int64) ([]models.User, error) {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 {
		return nil, ErrRolePermissionDenied
	}
	var authorized bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users actor ON actor.principal_id = ap.id
			JOIN schools s ON s.id = actor.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff' AND ap.status = 'active'
			  AND actor.status = 'active' AND actor.school_id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school' AND r.principal_type = 'staff'
			  AND r.role_key = 'school_admin' AND r.is_system = TRUE
			  AND p.permission_key = 'users.read' AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id AND denied.school_id = $2
			        AND denied.permission_id = p.id AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return nil, fmt.Errorf("authorize staff account listing: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.principal_id, u.school_id, u.username, u.status,
		       u.must_change_password, u.created_at, u.updated_at,
               COALESCE(sp.employee_code, ''), COALESCE(sp.full_name, ''),
               COALESCE(sp.designation, ''), COALESCE(sp.mobile_number, ''),
               COALESCE(sp.email, ''), sp.joining_date,
               COALESCE(sp.employment_status, ''), (sp.principal_id IS NOT NULL)
		FROM users u
        LEFT JOIN staff_profiles sp ON sp.principal_id = u.principal_id AND sp.school_id = u.school_id
		WHERE u.school_id = $1
		ORDER BY LOWER(u.username), u.id
	`, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list school staff accounts: %w", err)
	}
	defer rows.Close()
	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
        var joiningDate *time.Time
		if err := rows.Scan(&user.ID, &user.PrincipalID, &user.SchoolID, &user.Username,
			&user.Status, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt,
            &user.EmployeeCode, &user.FullName, &user.Designation, &user.MobileNumber,
            &user.Email, &joiningDate, &user.EmploymentStatus, &user.ProfileComplete); err != nil {
			return nil, fmt.Errorf("scan school staff account: %w", err)
		}
        user.JoiningDate = joiningDate
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate school staff accounts: %w", err)
	}
	return users, nil
}

func (r *UserRepository) CreateStaffAccount(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	username, passwordHash string,
) (*models.User, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 || username == "" || passwordHash == "" {
		return nil, ErrInvalidUser
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin staff account creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users actor ON actor.principal_id = ap.id
			JOIN schools s ON s.id = actor.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff' AND ap.status = 'active'
			  AND actor.status = 'active' AND actor.school_id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school'
			  AND r.principal_type = 'staff' AND r.role_key = 'school_admin' AND r.is_system = TRUE
			  AND p.permission_key = 'users.create' AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id AND denied.school_id = $2
			        AND denied.permission_id = p.id AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return nil, fmt.Errorf("authorize staff account creation: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}

	var user models.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id, school_id, username, status,
		          must_change_password, created_at, updated_at
	`, schoolID, username, passwordHash).Scan(
		&user.ID, &user.PrincipalID, &user.SchoolID, &user.Username,
		&user.Status, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_users_username_global" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("insert staff account: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.account.created', 'staff_user', $3, 'success',
		        jsonb_build_object('principal_id', $4::bigint, 'username', $5::text, 'must_change_password', $6::boolean))
	`, actorPrincipalID, schoolID, fmt.Sprint(user.ID), user.PrincipalID, user.Username, user.MustChangePassword); err != nil {
		return nil, fmt.Errorf("audit staff account creation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit staff account creation: %w", err)
	}
	return &user, nil
}


func (r *UserRepository) CreateStaffAccountWithProfile(
	ctx context.Context,
	actorPrincipalID, schoolID int64,
	username, passwordHash string,
	profile models.StaffProfile,
) (*models.User, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 || username == "" || passwordHash == "" {
		return nil, ErrInvalidUser
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin staff account creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var authorized bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users actor ON actor.principal_id = ap.id
			JOIN schools s ON s.id = actor.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff' AND ap.status = 'active'
			  AND actor.status = 'active' AND actor.school_id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school'
			  AND r.principal_type = 'staff' AND r.role_key = 'school_admin' AND r.is_system = TRUE
			  AND p.permission_key = 'users.create' AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id AND denied.school_id = $2
			        AND denied.permission_id = p.id AND denied.effect = 'deny'
			  )
		)
	`, actorPrincipalID, schoolID).Scan(&authorized)
	if err != nil {
		return nil, fmt.Errorf("authorize staff account creation: %w", err)
	}
	if !authorized {
		return nil, ErrRolePermissionDenied
	}

	var user models.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id, school_id, username, status,
		          must_change_password, created_at, updated_at
	`, schoolID, username, passwordHash).Scan(
		&user.ID, &user.PrincipalID, &user.SchoolID, &user.Username,
		&user.Status, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_users_username_global" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("insert staff account: %w", err)
	}

    if _, err := tx.Exec(ctx, `
        INSERT INTO staff_profiles
            (principal_id, school_id, employee_code, full_name, designation, mobile_number, email, joining_date, employment_status)
        VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, 'active')
    `, user.PrincipalID, schoolID, profile.EmployeeCode, profile.FullName, profile.Designation, profile.MobileNumber, profile.Email, profile.JoiningDate); err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_staff_profiles_school_employee_code" {
            return nil, ErrEmployeeCodeTaken
        }
        return nil, fmt.Errorf("insert staff employee profile: %w", err)
    }
    user.EmployeeCode = profile.EmployeeCode
    user.FullName = profile.FullName
    user.Designation = profile.Designation
    user.MobileNumber = profile.MobileNumber
    user.Email = profile.Email
    user.JoiningDate = profile.JoiningDate
    user.EmploymentStatus = "active"
    user.ProfileComplete = true

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.account.created', 'staff_user', $3, 'success',
		        jsonb_build_object('principal_id', $4::bigint, 'username', $5::text, 'must_change_password', $6::boolean))
	`, actorPrincipalID, schoolID, fmt.Sprint(user.ID), user.PrincipalID, user.Username, user.MustChangePassword); err != nil {
		return nil, fmt.Errorf("audit staff account creation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit staff account creation: %w", err)
	}
	return &user, nil
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
		  AND status = 'active'
		  AND EXISTS (
		      SELECT 1 FROM auth_principals ap
		      WHERE ap.id = users.principal_id
		        AND ap.principal_type = 'staff'
		        AND ap.status = 'active'
		  )
		  AND EXISTS (
		      SELECT 1 FROM schools s
		      WHERE s.id = users.school_id
		        AND s.status = 'active'
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
		   AND status = 'active'
		   AND EXISTS (
		       SELECT 1
		         FROM auth_principals ap
		        WHERE ap.id = users.principal_id
		          AND ap.principal_type = 'staff'
		          AND ap.status = 'active'
		   )
		   AND EXISTS (
		       SELECT 1
		         FROM schools s
		        WHERE s.id = users.school_id
		          AND s.status = 'active'
		   )
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
		  AND EXISTS (
		      SELECT 1 FROM schools target_school
		      WHERE target_school.id = $2 AND target_school.status = 'active'
		  )
		  AND target_principal.principal_type = 'staff'
		  AND target_principal.status = 'active'
		  AND target.status = 'active'
		  AND NOT EXISTS (
		      SELECT 1
		        FROM principal_roles target_roles
		        JOIN roles target_role ON target_role.id = target_roles.role_id
		       WHERE target_roles.principal_id = target.principal_id
		         AND target_roles.revoked_at IS NULL
		         AND target_role.school_id = $2
		         AND target_role.scope_type = 'school'
		         AND target_role.principal_type = 'staff'
		         AND target_role.role_key = 'school_admin'
		         AND target_role.is_system = TRUE
		  )
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
		      JOIN permissions perm ON perm.id = rp.permission_id
		      JOIN schools actor_school
		        ON actor_school.id = actor.school_id
		       AND actor_school.status = 'active'
		      WHERE actor.principal_id = $3
		        AND actor.school_id = $2
		        AND actor.status = 'active'
		        AND actor_principal.status = 'active'
		        AND actor_principal.principal_type = 'staff'
		        AND perm.permission_key = 'users.password_reset'
		        AND perm.scope_type = 'school'
		        AND perm.is_active = TRUE
		        AND NOT EXISTS (
		            SELECT 1
		            FROM user_permission_overrides actor_override
		            WHERE actor_override.principal_id = actor.principal_id
		              AND actor_override.school_id = $2
		              AND actor_override.permission_id = perm.id
		              AND actor_override.effect = 'deny'
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

func (r *UserRepository) DisableStaffAccount(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) error {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || actorPrincipalID == targetPrincipalID {
		return ErrInvalidUser
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin staff account disable: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		targetPrincipalID,
	); err != nil {
		return fmt.Errorf("lock target staff principal disable: %w", err)
	}

	var targetUserID int64
	err = tx.QueryRow(ctx, `
		SELECT target.id
		FROM users target
		JOIN auth_principals target_principal ON target_principal.id = target.principal_id
		WHERE target.principal_id = $1
		  AND target.school_id = $2
		  AND target.status = 'active'
		  AND target_principal.principal_type = 'staff'
		  AND target_principal.status = 'active'
		  AND NOT EXISTS (
		      SELECT 1
		      FROM principal_roles target_roles
		      JOIN roles target_role ON target_role.id = target_roles.role_id
		      WHERE target_roles.principal_id = target.principal_id
		        AND target_roles.revoked_at IS NULL
		        AND target_role.school_id = $2
		        AND target_role.scope_type = 'school'
		        AND target_role.principal_type = 'staff'
		        AND target_role.role_key = 'school_admin'
		        AND target_role.is_system = TRUE
		  )
		  AND EXISTS (
		      SELECT 1
		      FROM users actor
		      JOIN auth_principals actor_principal ON actor_principal.id = actor.principal_id
		      JOIN principal_roles pr ON pr.principal_id = actor.principal_id AND pr.revoked_at IS NULL
		      JOIN roles role ON role.id = pr.role_id
		      JOIN role_permissions rp ON rp.role_id = role.id
		      JOIN permissions perm ON perm.id = rp.permission_id
		      JOIN schools actor_school ON actor_school.id = actor.school_id AND actor_school.status = 'active'
		      WHERE actor.principal_id = $3 AND actor.school_id = $2
		        AND actor.status = 'active'
		        AND actor_principal.principal_type = 'staff' AND actor_principal.status = 'active'
		        AND role.school_id = $2 AND role.scope_type = 'school'
		        AND role.principal_type = 'staff' AND role.role_key = 'school_admin' AND role.is_system = TRUE
		        AND perm.permission_key = 'users.disable'
		        AND perm.scope_type = 'school' AND perm.is_active = TRUE
		        AND NOT EXISTS (
		            SELECT 1 FROM user_permission_overrides denied
		            WHERE denied.principal_id = actor.principal_id
		              AND denied.school_id = $2
		              AND denied.permission_id = perm.id
		              AND denied.effect = 'deny'
		        )
		  )
		FOR UPDATE OF target, target_principal
	`, targetPrincipalID, schoolID, actorPrincipalID).Scan(&targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("authorize same-school staff account disable: %w", err)
	}
	// auth_principals is canonical in V4. Its status trigger synchronizes
	// users.status in the same transaction; updating both rows independently
	// would make the second status transition a zero-row update.
	tag, err := tx.Exec(ctx, `
		UPDATE auth_principals
		SET status = 'disabled', token_version = token_version + 1, updated_at = NOW()
		WHERE id = $1 AND principal_type = 'staff' AND status = 'active'
	`, targetPrincipalID)
	if err != nil {
		return fmt.Errorf("disable staff principal: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE principal_id = $1 AND revoked_at IS NULL
	`, targetPrincipalID); err != nil {
		return fmt.Errorf("revoke disabled staff refresh sessions: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.account.disabled', 'staff_user', $3, 'success',
		        jsonb_build_object('target_principal_id', $4::bigint, 'sessions_revoked', TRUE))
	`, actorPrincipalID, schoolID, fmt.Sprint(targetUserID), targetPrincipalID); err != nil {
		return fmt.Errorf("audit staff account disable: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit staff account disable: %w", err)
	}
	return nil
}


// UpdateStaffAccount changes a same-school staff username. The operation is
// restricted to School Admins with users.update and invalidates every existing
// access/refresh credential because the login identity has changed.
func (r *UserRepository) UpdateStaffAccount(
	ctx context.Context,
	actorPrincipalID, schoolID, targetPrincipalID int64,
	username string,
) (*models.User, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || username == "" {
		return nil, ErrInvalidUser
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin staff account update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		targetPrincipalID,
	); err != nil {
		return nil, fmt.Errorf("lock target staff principal update: %w", err)
	}

	var user models.User
	err = tx.QueryRow(ctx, `
		SELECT target.id, target.principal_id, target.school_id, target.username,
		       target.status, target.must_change_password, target.created_at, target.updated_at
		  FROM users target
		  JOIN auth_principals target_principal ON target_principal.id = target.principal_id
		 WHERE target.principal_id = $1
		   AND target.school_id = $2
		   AND target.status = 'active'
		   AND target_principal.principal_type = 'staff'
		   AND target_principal.status = 'active'
		   AND NOT EXISTS (
		       SELECT 1
		         FROM principal_roles target_roles
		         JOIN roles target_role ON target_role.id = target_roles.role_id
		        WHERE target_roles.principal_id = target.principal_id
		          AND target_roles.revoked_at IS NULL
		          AND target_role.school_id = $2
		          AND target_role.scope_type = 'school'
		          AND target_role.principal_type = 'staff'
		          AND target_role.role_key = 'school_admin'
		          AND target_role.is_system = TRUE
		   )
		   AND EXISTS (
		       SELECT 1
		         FROM users actor
		         JOIN auth_principals actor_principal ON actor_principal.id = actor.principal_id
		         JOIN principal_roles pr ON pr.principal_id = actor.principal_id AND pr.revoked_at IS NULL
		         JOIN roles role ON role.id = pr.role_id
		         JOIN role_permissions rp ON rp.role_id = role.id
		         JOIN permissions perm ON perm.id = rp.permission_id
		         JOIN schools actor_school ON actor_school.id = actor.school_id AND actor_school.status = 'active'
		        WHERE actor.principal_id = $3
		          AND actor.school_id = $2
		          AND actor.status = 'active'
		          AND actor_principal.principal_type = 'staff'
		          AND actor_principal.status = 'active'
		          AND role.school_id = $2
		          AND role.scope_type = 'school'
		          AND role.principal_type = 'staff'
		          AND role.role_key = 'school_admin'
		          AND role.is_system = TRUE
		          AND perm.permission_key = 'users.update'
		          AND perm.scope_type = 'school'
		          AND perm.is_active = TRUE
		          AND NOT EXISTS (
		              SELECT 1 FROM user_permission_overrides denied
		               WHERE denied.principal_id = actor.principal_id
		                 AND denied.school_id = $2
		                 AND denied.permission_id = perm.id
		                 AND denied.effect = 'deny'
		          )
		   )
		 FOR UPDATE OF target, target_principal
	`, targetPrincipalID, schoolID, actorPrincipalID).Scan(
		&user.ID, &user.PrincipalID, &user.SchoolID, &user.Username,
		&user.Status, &user.MustChangePassword, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("authorize same-school staff account update: %w", err)
	}
	oldUsername := user.Username

	err = tx.QueryRow(ctx, `
		UPDATE users
		   SET username = $2,
		       updated_at = NOW()
		 WHERE id = $1
		RETURNING username, updated_at
	`, user.ID, username).Scan(&user.Username, &user.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			pgErr.ConstraintName == "ux_users_username_global" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("update staff username: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE auth_principals
		   SET token_version = token_version + 1,
		       updated_at = NOW()
		 WHERE id = $1
		   AND principal_type = 'staff'
		   AND status = 'active'
	`, targetPrincipalID)
	if err != nil {
		return nil, fmt.Errorf("invalidate updated staff access tokens: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions
		   SET revoked_at = COALESCE(revoked_at, NOW())
		 WHERE principal_id = $1
		   AND revoked_at IS NULL
	`, targetPrincipalID); err != nil {
		return nil, fmt.Errorf("revoke updated staff refresh sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.account.updated', 'staff_user', $3, 'success',
		        jsonb_build_object('old_username', $4::text, 'username', $5::text, 'sessions_revoked', TRUE))
	`, actorPrincipalID, schoolID, fmt.Sprint(user.ID), oldUsername, user.Username); err != nil {
		return nil, fmt.Errorf("audit staff account update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit staff account update: %w", err)
	}
	return &user, nil
}

// EnableStaffAccount reactivates a disabled same-school staff account only
// when a School Admin supplies a new temporary password. It forces a password
// change at next login and invalidates all pre-disable credentials.
func (r *UserRepository) EnableStaffAccount(
	ctx context.Context,
	actorPrincipalID, schoolID, targetPrincipalID int64,
	newPasswordHash string,
) error {
	if r == nil || r.db == nil || actorPrincipalID <= 0 || schoolID <= 0 ||
		targetPrincipalID <= 0 || actorPrincipalID == targetPrincipalID || newPasswordHash == "" {
		return ErrInvalidUser
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin staff account re-enable: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		targetPrincipalID,
	); err != nil {
		return fmt.Errorf("lock target staff principal re-enable: %w", err)
	}

	var targetUserID int64
	err = tx.QueryRow(ctx, `
		SELECT target.id
		  FROM users target
		  JOIN auth_principals target_principal ON target_principal.id = target.principal_id
		 WHERE target.principal_id = $1
		   AND target.school_id = $2
		   AND target.status = 'disabled'
		   AND target_principal.principal_type = 'staff'
		   AND target_principal.status = 'disabled'
		   AND NOT EXISTS (
		       SELECT 1
		         FROM principal_roles target_roles
		         JOIN roles target_role ON target_role.id = target_roles.role_id
		        WHERE target_roles.principal_id = target.principal_id
		          AND target_roles.revoked_at IS NULL
		          AND target_role.school_id = $2
		          AND target_role.scope_type = 'school'
		          AND target_role.principal_type = 'staff'
		          AND target_role.role_key = 'school_admin'
		          AND target_role.is_system = TRUE
		   )
		   AND EXISTS (
		       SELECT 1
		         FROM users actor
		         JOIN auth_principals actor_principal ON actor_principal.id = actor.principal_id
		         JOIN principal_roles pr ON pr.principal_id = actor.principal_id AND pr.revoked_at IS NULL
		         JOIN roles role ON role.id = pr.role_id
		         JOIN role_permissions rp ON rp.role_id = role.id
		         JOIN permissions perm ON perm.id = rp.permission_id
		         JOIN schools actor_school ON actor_school.id = actor.school_id AND actor_school.status = 'active'
		        WHERE actor.principal_id = $3
		          AND actor.school_id = $2
		          AND actor.status = 'active'
		          AND actor_principal.principal_type = 'staff'
		          AND actor_principal.status = 'active'
		          AND role.school_id = $2
		          AND role.scope_type = 'school'
		          AND role.principal_type = 'staff'
		          AND role.role_key = 'school_admin'
		          AND role.is_system = TRUE
		          AND perm.permission_key = 'users.update'
		          AND perm.scope_type = 'school'
		          AND perm.is_active = TRUE
		          AND NOT EXISTS (
		              SELECT 1 FROM user_permission_overrides denied
		               WHERE denied.principal_id = actor.principal_id
		                 AND denied.school_id = $2
		                 AND denied.permission_id = perm.id
		                 AND denied.effect = 'deny'
		          )
		   )
		 FOR UPDATE OF target, target_principal
	`, targetPrincipalID, schoolID, actorPrincipalID).Scan(&targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("authorize same-school staff account re-enable: %w", err)
	}

	// Reactivate the canonical principal first. The V4 status-sync trigger
	// updates users.status atomically; the credential update below then sets
	// the temporary password and forced-change flag without a second status
	// transition.
	tag, err := tx.Exec(ctx, `
		UPDATE auth_principals
		   SET status = 'active',
		       token_version = token_version + 1,
		       updated_at = NOW()
		 WHERE id = $1
		   AND principal_type = 'staff'
		   AND status = 'disabled'
	`, targetPrincipalID)
	if err != nil {
		return fmt.Errorf("reactivate staff principal: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrUserNotFound
	}

	tag, err = tx.Exec(ctx, `
		UPDATE users
		   SET password_hash = $2,
		       must_change_password = TRUE,
		       updated_at = NOW()
		 WHERE id = $1
		   AND status = 'active'
	`, targetUserID, newPasswordHash)
	if err != nil {
		return fmt.Errorf("update re-enabled staff credential: %w", err)
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
		return fmt.Errorf("revoke pre-existing staff refresh sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'staff.account.enabled', 'staff_user', $3, 'success',
		        jsonb_build_object('must_change_password', TRUE, 'sessions_revoked', TRUE))
	`, actorPrincipalID, schoolID, fmt.Sprint(targetUserID)); err != nil {
		return fmt.Errorf("audit staff account re-enable: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit staff account re-enable: %w", err)
	}
	return nil
}
