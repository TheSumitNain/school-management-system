package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlatformAdminNotFound          = errors.New("Platform Admin credentials not found")
	ErrPlatformAdminAlreadyBootstrapped = errors.New("the first Platform Admin has already been bootstrapped")
	ErrPlatformAdminEmailTaken        = errors.New("Platform Admin email already exists")
	ErrPlatformAdminPasswordConflict = errors.New("Platform Admin password changed concurrently")
)

type PlatformAdminCredential struct {
	Principal   *models.AuthPrincipal
	PasswordHash string
	Email        string
}

type PlatformAdminRepository struct {
	db *pgxpool.Pool
}

func NewPlatformAdminRepository(db *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{db: db}
}

func (r *PlatformAdminRepository) GetByEmail(ctx context.Context, email string) (*PlatformAdminCredential, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, ErrPlatformAdminNotFound
	}

	const q = `SELECT ap.id, ap.principal_type, ap.status, ap.token_version,
       pa.email, pa.password_hash
FROM platform_admin_accounts pa
JOIN auth_principals ap ON ap.id = pa.principal_id
WHERE LOWER(BTRIM(pa.email)) = $1`

	var result PlatformAdminCredential
	var principal models.AuthPrincipal
	err := r.db.QueryRow(ctx, q, email).Scan(
		&principal.ID, &principal.Type, &principal.Status, &principal.TokenVersion,
		&result.Email, &result.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlatformAdminNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load Platform Admin credentials: %w", err)
	}
	if principal.Type != "platform_admin" || principal.Status != "active" || principal.TokenVersion < 1 {
		return nil, ErrPlatformAdminNotFound
	}
	result.Principal = &principal
	return &result, nil
}

// BootstrapFirst creates the one initial Platform Admin in a single transaction.
// It is a manually invoked, one-time operation; no HTTP endpoint exposes it.
func (r *PlatformAdminRepository) BootstrapFirst(
	ctx context.Context,
	email string,
	passwordHash string,
) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if r == nil || r.db == nil || email == "" || len(email) > 254 || passwordHash == "" {
		return 0, fmt.Errorf("invalid Platform Admin bootstrap input")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin Platform Admin bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	principalID, err := r.bootstrapFirstInTx(ctx, tx, email, passwordHash)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit initial Platform Admin bootstrap: %w", err)
	}
	return principalID, nil
}

// BootstrapFirstInTx performs the one-time bootstrap inside a caller-owned
// transaction. It is intended for atomic setup workflows and rollback-scoped
// integration tests; callers retain responsibility for committing or rolling back.
func (r *PlatformAdminRepository) BootstrapFirstInTx(
	ctx context.Context,
	tx pgx.Tx,
	email string,
	passwordHash string,
) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if r == nil || r.db == nil || tx == nil || email == "" || len(email) > 254 || passwordHash == "" {
		return 0, fmt.Errorf("invalid Platform Admin bootstrap input")
	}
	return r.bootstrapFirstInTx(ctx, tx, email, passwordHash)
}

// bootstrapFirstInTx performs the complete bootstrap inside a caller-owned
// transaction. Keeping transaction ownership outside permits rollback-based
// PostgreSQL integration tests without leaving a permanent Platform Admin.
func (r *PlatformAdminRepository) bootstrapFirstInTx(
	ctx context.Context,
	tx pgx.Tx,
	email string,
	passwordHash string,
) (int64, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7414, 1)`); err != nil {
		return 0, fmt.Errorf("lock Platform Admin bootstrap: %w", err)
	}

	var alreadyExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
		    OR EXISTS (
		        SELECT 1
		          FROM principal_roles pr
		          JOIN roles r ON r.id = pr.role_id
		         WHERE r.principal_type = 'platform_admin'
		           AND r.scope_type = 'platform'
		           AND r.role_key = 'platform_admin'
		    )
	`).Scan(&alreadyExists); err != nil {
		return 0, fmt.Errorf("check existing Platform Admin bootstrap: %w", err)
	}
	if alreadyExists {
		return 0, ErrPlatformAdminAlreadyBootstrapped
	}

	var principalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type, status)
		VALUES ('platform_admin', 'active')
		RETURNING id
	`).Scan(&principalID); err != nil {
		return 0, fmt.Errorf("create initial Platform Admin principal: %w", err)
	}

	// The operator supplies the permanent initial password. The password policy
	// is enforced before this repository method is called; no password is returned.
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform_admin_accounts
		    (principal_id, email, password_hash, must_change_password)
		VALUES ($1, $2, $3, FALSE)
	`, principalID, email, passwordHash); err != nil {
		return 0, mapPlatformAdminBootstrapError(err)
	}

	var roleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		  FROM roles
		 WHERE principal_type = 'platform_admin'
		   AND scope_type = 'platform'
		   AND role_key = 'platform_admin'
		   AND is_system = TRUE
	`).Scan(&roleID); err != nil {
		return 0, fmt.Errorf("load Platform Admin system role: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, NULL)
	`, principalID, roleID); err != nil {
		return 0, fmt.Errorf("assign initial Platform Admin system role: %w", err)
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, action, resource_type, resource_id, outcome, metadata)
		VALUES (NULL, 'platform.admin.bootstrapped', 'platform_admin', $1, 'success',
		        jsonb_build_object('email', $2::text, 'bootstrap_method', 'one_time_cli'))
	`, strconv.FormatInt(principalID, 10), email)
	if err != nil {
		return 0, fmt.Errorf("audit initial Platform Admin bootstrap: %w", err)
	}
	return principalID, nil
}

func mapPlatformAdminBootstrapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "ux_platform_admin_email" {
		return ErrPlatformAdminEmailTaken
	}
	return fmt.Errorf("create initial Platform Admin account: %w", err)
}


// GetPasswordHashByPrincipalID returns the active Platform Admin's stored
// password hash for an authenticated self-service password change.
func (r *PlatformAdminRepository) GetPasswordHashByPrincipalID(ctx context.Context, principalID int64) (string, error) {
	if r == nil || r.db == nil || principalID <= 0 {
		return "", ErrPlatformAdminNotFound
	}
	var hash string
	err := r.db.QueryRow(ctx, `
		SELECT pa.password_hash
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		 WHERE pa.principal_id = $1
		   AND ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
	`, principalID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPlatformAdminNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load Platform Admin password credential: %w", err)
	}
	return hash, nil
}

// ChangePassword atomically updates the Platform Admin credential, invalidates
// all existing access/refresh sessions, and appends an immutable audit event.
// The expected hash prevents a stale verification from overwriting a concurrent change.
func (r *PlatformAdminRepository) ChangePassword(ctx context.Context, principalID int64, expectedHash, newHash string) error {
	if r == nil || r.db == nil || principalID <= 0 || expectedHash == "" || newHash == "" {
		return ErrPlatformAdminNotFound
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin Platform Admin password change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		principalID,
	); err != nil {
		return fmt.Errorf("lock Platform Admin password change: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE platform_admin_accounts
		   SET password_hash = $3
		 WHERE principal_id = $1
		   AND password_hash = $2
		   AND EXISTS (
		       SELECT 1 FROM auth_principals ap
		        WHERE ap.id = platform_admin_accounts.principal_id
		          AND ap.principal_type = 'platform_admin'
		          AND ap.status = 'active'
		   )
	`, principalID, expectedHash, newHash)
	if err != nil {
		return fmt.Errorf("update Platform Admin password: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPlatformAdminPasswordConflict
	}

	tag, err = tx.Exec(ctx, `
		UPDATE auth_principals
		   SET token_version = token_version + 1,
		       updated_at = NOW()
		 WHERE id = $1
		   AND principal_type = 'platform_admin'
		   AND status = 'active'
	`, principalID)
	if err != nil {
		return fmt.Errorf("invalidate Platform Admin access tokens: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrPlatformAdminNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions
		   SET revoked_at = COALESCE(revoked_at, NOW())
		 WHERE principal_id = $1
		   AND revoked_at IS NULL
	`, principalID); err != nil {
		return fmt.Errorf("revoke Platform Admin refresh sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, 'platform.admin.password.changed', 'platform_admin', $2, 'success',
		        jsonb_build_object('sessions_revoked', TRUE))
	`, principalID, strconv.FormatInt(principalID, 10)); err != nil {
		return fmt.Errorf("audit Platform Admin password change: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Platform Admin password change: %w", err)
	}
	return nil
}
