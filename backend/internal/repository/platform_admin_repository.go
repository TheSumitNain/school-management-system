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

	// The same transaction-scoped lock is enforced by the V14 role-assignment
	// trigger, so concurrent bootstrap attempts cannot both pass the first-admin check.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7414, 1)`); err != nil {
		return 0, fmt.Errorf("lock Platform Admin bootstrap: %w", err)
	}

	var alreadyExists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
		    OR EXISTS (
		        SELECT 1
		          FROM principal_roles pr
		          JOIN roles r ON r.id = pr.role_id
		         WHERE r.principal_type = 'platform_admin'
		           AND r.scope_type = 'platform'
		           AND r.role_key = 'platform_admin'
		    )
	`).Scan(&alreadyExists)
	if err != nil {
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

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events
		    (actor_principal_id, action, resource_type, resource_id, outcome, metadata)
		VALUES (NULL, 'platform.admin.bootstrapped', 'platform_admin', $1, 'success',
		        jsonb_build_object('email', $2, 'bootstrap_method', 'one_time_cli'))
	`, strconv.FormatInt(principalID, 10), email)
	if err != nil {
		return 0, fmt.Errorf("audit initial Platform Admin bootstrap: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit initial Platform Admin bootstrap: %w", err)
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
