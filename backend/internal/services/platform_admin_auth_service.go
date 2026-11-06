package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

type platformAdminCredentialStore interface {
	GetByEmail(context.Context,string) (*repository.PlatformAdminCredential,error)
}

type platformAdminLoginSecurity interface {
	AcquireAttemptLock(context.Context,string) (func(),error)
	IsLocked(context.Context,string) (bool,error)
	RecordFailure(context.Context,string) error
	RecordPlatformAdminSuccess(context.Context,string,int64) error
}

type PlatformAdminAuthService struct {
	credentials platformAdminCredentialStore
	loginSecurity platformAdminLoginSecurity
}

func NewPlatformAdminAuthService(credentials platformAdminCredentialStore, security platformAdminLoginSecurity) (*PlatformAdminAuthService,error) {
	if credentials==nil || security==nil { return nil,fmt.Errorf("Platform Admin credential and login-security repositories are required") }
	return &PlatformAdminAuthService{credentials:credentials,loginSecurity:security},nil
}

func (s *PlatformAdminAuthService) Authenticate(ctx context.Context,email,password string) (*models.AuthPrincipal,error) {
	email=strings.TrimSpace(strings.ToLower(email))
	if email=="" || password=="" { return nil,ErrInvalidCredentials }
	release,err:=s.loginSecurity.AcquireAttemptLock(ctx,email)
	if err!=nil { return nil,err }
	defer release()
	locked,err:=s.loginSecurity.IsLocked(ctx,email)
	if err!=nil { return nil,err }
	if locked { return nil,ErrLoginLocked }
	record,err:=s.credentials.GetByEmail(ctx,email)
	if err!=nil {
		if errors.Is(err,repository.ErrPlatformAdminNotFound) {
			if recordErr:=s.loginSecurity.RecordFailure(ctx,email); recordErr!=nil { return nil,recordErr }
			return nil,ErrInvalidCredentials
		}
		return nil,err
	}
	if record.Principal==nil || !CheckPassword(password,record.PasswordHash) {
		if recordErr:=s.loginSecurity.RecordFailure(ctx,email); recordErr!=nil { return nil,recordErr }
		return nil,ErrInvalidCredentials
	}
	if err:=s.loginSecurity.RecordPlatformAdminSuccess(ctx,email,record.Principal.ID); err!=nil { return nil,err }
	return record.Principal,nil
}
