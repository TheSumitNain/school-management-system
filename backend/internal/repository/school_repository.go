package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSchoolNotFound = errors.New("school not found")

type School struct {
	ID     int64
	Name   string
	Code   string
	Status string
}

type SchoolRepository struct {
	db *pgxpool.Pool
}

func NewSchoolRepository(db *pgxpool.Pool) *SchoolRepository {
	return &SchoolRepository{db: db}
}

func (r *SchoolRepository) GetByID(
	ctx context.Context,
	schoolID int64,
) (*School, error) {
	if schoolID <= 0 {
		return nil, ErrSchoolNotFound
	}

	const query = `
		SELECT id, name, code, status
		FROM schools
		WHERE id = $1
	`

	var school School

	err := r.db.QueryRow(ctx, query, schoolID).Scan(
		&school.ID,
		&school.Name,
		&school.Code,
		&school.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get school by ID: %w", err)
	}

	return &school, nil
}
