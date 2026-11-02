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
		       status, created_at, updated_at
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
		          status, created_at, updated_at
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
		&user.SchoolID,
		&user.Username,
		&user.PasswordHash,
		&user.Status,
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
