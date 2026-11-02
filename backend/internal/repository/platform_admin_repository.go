package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPlatformAdminNotFound = errors.New("Platform Admin credentials not found")

type PlatformAdminCredential struct {
	Principal *models.AuthPrincipal
	PasswordHash string
	Email string
}

type PlatformAdminRepository struct { db *pgxpool.Pool }

func NewPlatformAdminRepository(db *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{db:db}
}

func (r *PlatformAdminRepository) GetByEmail(ctx context.Context,email string) (*PlatformAdminCredential,error) {
	email=strings.TrimSpace(strings.ToLower(email))
	if email=="" { return nil,ErrPlatformAdminNotFound }
	const q=`SELECT ap.id,ap.principal_type,ap.status,ap.token_version,
pa.email,pa.password_hash
FROM platform_admin_accounts pa
JOIN auth_principals ap ON ap.id=pa.principal_id
WHERE LOWER(BTRIM(pa.email))=$1`
	var result PlatformAdminCredential
	var principal models.AuthPrincipal
	err:=r.db.QueryRow(ctx,q,email).Scan(&principal.ID,&principal.Type,&principal.Status,&principal.TokenVersion,&result.Email,&result.PasswordHash)
	if errors.Is(err,pgx.ErrNoRows) { return nil,ErrPlatformAdminNotFound }
	if err!=nil { return nil,fmt.Errorf("load Platform Admin credentials: %w",err) }
	if principal.Type!="platform_admin" || principal.Status!="active" || principal.TokenVersion<1 {
		return nil,ErrPlatformAdminNotFound
	}
	result.Principal=&principal
	return &result,nil
}
