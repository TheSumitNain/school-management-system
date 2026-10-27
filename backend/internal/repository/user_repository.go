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
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already exists in this school")
	ErrInvalidUser   = errors.New("invalid user data")
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
		SELECT id, school_id, username, password_hash,
		       status, created_at, updated_at
		FROM users
		WHERE school_id = $1
		  AND LOWER(username) = LOWER($2)
	`

	var user models.User

	err := r.db.QueryRow(ctx, query, schoolID, strings.TrimSpace(username)).Scan(
		&user.ID,
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
		RETURNING id, school_id, username, password_hash,
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
			pgErr.ConstraintName == "users_school_username_unique" {
			return nil, ErrUsernameTaken
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}
