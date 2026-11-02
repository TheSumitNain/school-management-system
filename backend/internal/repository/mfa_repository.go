package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMFACredentialExists = errors.New("verified MFA credential already exists")
	ErrMFAEnrollmentInvalid = errors.New("MFA enrollment challenge is invalid")
	ErrMFARecoveryCodeInvalid = errors.New("MFA recovery code is invalid or already used")
)

type MFACredential struct {
	Ciphertext []byte
	KeyVersion int
	VerifiedAt *time.Time
}

type MFARepository struct { db *pgxpool.Pool }

func NewMFARepository(db *pgxpool.Pool) *MFARepository {
	return &MFARepository{db: db}
}

// IsMandatoryForPrincipal is deliberately based on the canonical system role,
// not a client-supplied role or a JWT claim.
func (r *MFARepository) IsMandatoryForPrincipal(ctx context.Context, principalID int64) (bool, error) {
	if principalID <= 0 { return false, fmt.Errorf("invalid principal ID") }
	const q = `
SELECT CASE
 WHEN ap.principal_type = 'platform_admin' THEN TRUE
 WHEN ap.principal_type = 'staff' THEN EXISTS (
   SELECT 1 FROM principal_roles pr
   JOIN roles ro ON ro.id=pr.role_id
   WHERE pr.principal_id=ap.id AND pr.revoked_at IS NULL
     AND ro.is_system=TRUE AND ro.role_key='school_admin'
     AND ro.principal_type='staff' AND ro.scope_type='school'
     AND ro.school_id=(SELECT school_id FROM users WHERE principal_id=ap.id)
 )
 ELSE FALSE END
FROM auth_principals ap WHERE ap.id=$1 AND ap.status='active'`
	var required bool
	if err:=r.db.QueryRow(ctx,q,principalID).Scan(&required); err!=nil {
		if errors.Is(err,pgx.ErrNoRows) { return false, modelsPrincipalNotFound() }
		return false,fmt.Errorf("load mandatory MFA policy: %w",err)
	}
	return required,nil
}

func (r *MFARepository) GetCredential(ctx context.Context, principalID int64) (*MFACredential,error) {
	if principalID<=0 { return nil,fmt.Errorf("invalid principal ID") }
	var c MFACredential
	err:=r.db.QueryRow(ctx,`SELECT secret_ciphertext,encryption_key_version,verified_at
FROM principal_mfa_credentials WHERE principal_id=$1 AND method='totp'`,principalID).Scan(&c.Ciphertext,&c.KeyVersion,&c.VerifiedAt)
	if errors.Is(err,pgx.ErrNoRows) { return nil,nil }
	if err!=nil { return nil,fmt.Errorf("load MFA credential: %w",err) }
	return &c,nil
}

// SavePendingCredential never replaces an already verified authenticator.
func (r *MFARepository) SavePendingCredential(ctx context.Context, principalID int64, ciphertext []byte, keyVersion int) error {
	if principalID<=0 || len(ciphertext)<28 || keyVersion<1 { return fmt.Errorf("invalid pending MFA credential") }
	tag,err:=r.db.Exec(ctx,`INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
VALUES ($1,'totp',$2,$3,NULL)
ON CONFLICT (principal_id) DO UPDATE SET
 secret_ciphertext=EXCLUDED.secret_ciphertext,
 encryption_key_version=EXCLUDED.encryption_key_version,
 updated_at=NOW()
WHERE principal_mfa_credentials.verified_at IS NULL`,principalID,ciphertext,keyVersion)
	if err!=nil { return fmt.Errorf("save pending MFA credential: %w",err) }
	if tag.RowsAffected()!=1 { return ErrMFACredentialExists }
	return nil
}

// ActivateEnrollment consumes the challenge, verifies its principal and
// purpose, activates the pending credential, invalidates existing sessions,
// and records the security event in one transaction.
func (r *MFARepository) ActivateEnrollment(ctx context.Context, principalID int64, challengeHash string, acceptedStep int64, recoveryHashes []string) error {
	if principalID<=0 || !isSHA256Hash(challengeHash) || acceptedStep < 0 || !validRecoveryCodeHashes(recoveryHashes) { return ErrMFAEnrollmentInvalid }
	tx,err:=r.db.Begin(ctx); if err!=nil { return fmt.Errorf("begin MFA enrollment: %w",err) }
	defer func(){ _=tx.Rollback(ctx) }()
	var challengePrincipal int64
	var purpose string
	err=tx.QueryRow(ctx,`SELECT principal_id,purpose FROM auth_mfa_challenges
WHERE challenge_hash=$1 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5
FOR UPDATE`,challengeHash).Scan(&challengePrincipal,&purpose)
	if errors.Is(err,pgx.ErrNoRows) { return ErrMFAEnrollmentInvalid }
	if err!=nil { return fmt.Errorf("lock MFA enrollment challenge: %w",err) }
	if challengePrincipal!=principalID || purpose!="enrollment" { return ErrMFAEnrollmentInvalid }
	tag,err:=tx.Exec(ctx,`UPDATE principal_mfa_credentials SET verified_at=NOW(),last_accepted_step=$2,updated_at=NOW()
WHERE principal_id=$1 AND method='totp' AND verified_at IS NULL`,principalID,acceptedStep)
	if err!=nil { return fmt.Errorf("activate MFA credential: %w",err) }
	if tag.RowsAffected()!=1 { return ErrMFAEnrollmentInvalid }
	tag,err=tx.Exec(ctx,`UPDATE auth_mfa_challenges SET consumed_at=NOW()
WHERE challenge_hash=$1 AND consumed_at IS NULL`,challengeHash)
	if err!=nil { return fmt.Errorf("consume MFA enrollment challenge: %w",err) }
	if tag.RowsAffected()!=1 { return ErrMFAEnrollmentInvalid }
	if _,err=tx.Exec(ctx,`UPDATE platform_admin_accounts SET mfa_enrolled_at=NOW(),updated_at=NOW() WHERE principal_id=$1`,principalID); err!=nil {
		return fmt.Errorf("mark Platform Admin MFA enrollment: %w",err)
	}
	if _,err=tx.Exec(ctx,`UPDATE auth_principals SET token_version=token_version+1,updated_at=NOW() WHERE id=$1 AND status='active'`,principalID); err!=nil {
		return fmt.Errorf("invalidate pre-MFA access tokens: %w",err)
	}
	if _,err=tx.Exec(ctx,`UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,NOW())
WHERE principal_id=$1 AND revoked_at IS NULL`,principalID); err!=nil {
		return fmt.Errorf("revoke pre-MFA refresh sessions: %w",err)
	}
	if err = insertRecoveryCodeHashes(ctx, tx, principalID, recoveryHashes); err != nil {
		return fmt.Errorf("store initial recovery codes: %w", err)
	}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata)
VALUES($1,'auth.mfa.enrollment_completed','principal',$1::text,'success','{}'::jsonb)`,principalID); err!=nil {
		return fmt.Errorf("audit MFA enrollment: %w",err)
	}
	if err=tx.Commit(ctx); err!=nil { return fmt.Errorf("commit MFA enrollment: %w",err) }
	return nil
}

// ConsumeLoginChallenge atomically consumes the challenge and the matched
// TOTP time-step. The conditional counter update prevents replay across distinct
// challenges and remains safe when requests race.
func (r *MFARepository) ConsumeLoginChallenge(ctx context.Context, principalID int64, challengeHash string, acceptedStep int64) error {
	if principalID <= 0 || !isSHA256Hash(challengeHash) || acceptedStep < 0 {
		return ErrMFAEnrollmentInvalid
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin MFA login verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var challengePrincipal int64
	var purpose string
	err = tx.QueryRow(ctx, `SELECT principal_id,purpose FROM auth_mfa_challenges
WHERE challenge_hash=$1 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5
FOR UPDATE`, challengeHash).Scan(&challengePrincipal, &purpose)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMFAEnrollmentInvalid
	}
	if err != nil {
		return fmt.Errorf("lock MFA login challenge: %w", err)
	}
	if challengePrincipal != principalID || purpose != "login" {
		return ErrMFAEnrollmentInvalid
	}

	tag, err := tx.Exec(ctx, `UPDATE principal_mfa_credentials
SET last_accepted_step=$2,updated_at=NOW()
WHERE principal_id=$1 AND method='totp' AND verified_at IS NOT NULL
  AND (last_accepted_step IS NULL OR last_accepted_step < $2)`, principalID, acceptedStep)
	if err != nil {
		return fmt.Errorf("record accepted TOTP time-step: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrMFAEnrollmentInvalid
	}
	tag, err = tx.Exec(ctx, `UPDATE auth_mfa_challenges SET consumed_at=NOW()
WHERE challenge_hash=$1 AND principal_id=$2 AND consumed_at IS NULL
  AND expires_at>NOW() AND attempts<5`, challengeHash, principalID)
	if err != nil {
		return fmt.Errorf("consume MFA login challenge: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrMFAEnrollmentInvalid
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit MFA login verification: %w", err)
	}
	return nil
}

// ConsumeLoginRecoveryCode consumes a recovery code and its login challenge atomically.
func (r *MFARepository) ConsumeLoginRecoveryCode(ctx context.Context, principalID int64, challengeHash, codeHash string) error {
	if principalID <= 0 || !isSHA256Hash(challengeHash) || !isSHA256Hash(codeHash) { return ErrMFARecoveryCodeInvalid }
	tx, err := r.db.Begin(ctx)
	if err != nil { return fmt.Errorf("begin recovery-code login: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var challengePrincipal int64
	var purpose string
	err = tx.QueryRow(ctx, `SELECT principal_id,purpose FROM auth_mfa_challenges
WHERE challenge_hash=$1 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5
FOR UPDATE`, challengeHash).Scan(&challengePrincipal, &purpose)
	if errors.Is(err, pgx.ErrNoRows) { return ErrMFAEnrollmentInvalid }
	if err != nil { return fmt.Errorf("lock recovery-code challenge: %w", err) }
	if challengePrincipal != principalID || purpose != "login" { return ErrMFAEnrollmentInvalid }
	tag, err := tx.Exec(ctx, `UPDATE principal_mfa_recovery_codes SET used_at=NOW()
WHERE principal_id=$1 AND code_hash=$2 AND used_at IS NULL`, principalID, strings.ToLower(codeHash))
	if err != nil { return fmt.Errorf("consume recovery code: %w", err) }
	if tag.RowsAffected() != 1 { return ErrMFARecoveryCodeInvalid }
	tag, err = tx.Exec(ctx, `UPDATE auth_mfa_challenges SET consumed_at=NOW()
WHERE challenge_hash=$1 AND principal_id=$2 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5`, challengeHash, principalID)
	if err != nil { return fmt.Errorf("consume recovery-code challenge: %w", err) }
	if tag.RowsAffected() != 1 { return ErrMFAEnrollmentInvalid }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata)
VALUES($1,'auth.mfa.recovery_code_used','principal',$1::text,'success','{}'::jsonb)`, principalID); err != nil {
		return fmt.Errorf("audit recovery-code use: %w", err)
	}
	if err = tx.Commit(ctx); err != nil { return fmt.Errorf("commit recovery-code login: %w", err) }
	return nil
}

// ReplaceRecoveryCodeHashes invalidates all remaining codes, records the fresh
// TOTP step, inserts the replacement set and audits the change atomically.
func (r *MFARepository) ReplaceRecoveryCodeHashes(ctx context.Context, principalID int64, challengeHash string, acceptedStep int64, hashes []string) error {
	if principalID <= 0 || !isSHA256Hash(challengeHash) || acceptedStep < 0 || !validRecoveryCodeHashes(hashes) { return ErrMFAEnrollmentInvalid }
	tx, err := r.db.Begin(ctx)
	if err != nil { return fmt.Errorf("begin recovery-code regeneration: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var challengePrincipal int64
	var purpose string
	err = tx.QueryRow(ctx, `SELECT principal_id,purpose FROM auth_mfa_challenges
WHERE challenge_hash=$1 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5
FOR UPDATE`, challengeHash).Scan(&challengePrincipal, &purpose)
	if errors.Is(err, pgx.ErrNoRows) { return ErrMFAEnrollmentInvalid }
	if err != nil { return fmt.Errorf("lock recovery regeneration challenge: %w", err) }
	if challengePrincipal != principalID || purpose != "recovery_regeneration" { return ErrMFAEnrollmentInvalid }
	tag, err := tx.Exec(ctx, `UPDATE principal_mfa_credentials SET last_accepted_step=$2,updated_at=NOW()
WHERE principal_id=$1 AND method='totp' AND verified_at IS NOT NULL
AND (last_accepted_step IS NULL OR last_accepted_step<$2)`, principalID, acceptedStep)
	if err != nil { return fmt.Errorf("consume TOTP step for recovery regeneration: %w", err) }
	if tag.RowsAffected() != 1 { return ErrMFAEnrollmentInvalid }
	if _, err = tx.Exec(ctx, `UPDATE principal_mfa_recovery_codes SET used_at=NOW()
WHERE principal_id=$1 AND used_at IS NULL`, principalID); err != nil {
		return fmt.Errorf("invalidate previous recovery codes: %w", err)
	}
	if err = insertRecoveryCodeHashes(ctx, tx, principalID, hashes); err != nil { return fmt.Errorf("store replacement recovery codes: %w", err) }
	tag, err = tx.Exec(ctx, `UPDATE auth_mfa_challenges SET consumed_at=NOW()
WHERE challenge_hash=$1 AND principal_id=$2 AND consumed_at IS NULL AND expires_at>NOW() AND attempts<5`, challengeHash, principalID)
	if err != nil { return fmt.Errorf("consume recovery regeneration challenge: %w", err) }
	if tag.RowsAffected() != 1 { return ErrMFAEnrollmentInvalid }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,action,resource_type,resource_id,outcome,metadata)
VALUES($1,'auth.mfa.recovery_codes_regenerated','principal',$1::text,'success','{}'::jsonb)`, principalID); err != nil {
		return fmt.Errorf("audit recovery-code regeneration: %w", err)
	}
	if err = tx.Commit(ctx); err != nil { return fmt.Errorf("commit recovery-code regeneration: %w", err) }
	return nil
}

func validRecoveryCodeHashes(hashes []string) bool {
	if len(hashes) < 8 || len(hashes) > 12 { return false }
	seen := make(map[string]struct{}, len(hashes))
	for _, hash := range hashes {
		if !isSHA256Hash(hash) { return false }
		hash = strings.ToLower(hash)
		if _, ok := seen[hash]; ok { return false }
		seen[hash] = struct{}{}
	}
	return true
}

func insertRecoveryCodeHashes(ctx context.Context, tx pgx.Tx, principalID int64, hashes []string) error {
	if !validRecoveryCodeHashes(hashes) { return fmt.Errorf("invalid recovery code hash set") }
	for _, hash := range hashes {
		if _, err := tx.Exec(ctx, `INSERT INTO principal_mfa_recovery_codes(principal_id,code_hash) VALUES($1,$2)`, principalID, strings.ToLower(hash)); err != nil {
			return err
		}
	}
	return nil
}

func modelsPrincipalNotFound() error { return errors.New("principal is unavailable") }
