package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"school-management-system/internal/repository"
)

var (
	ErrInvalidPlatformAdminAccount = errors.New("invalid Platform Admin account")
	ErrPlatformAdminLifecycleForbidden = repository.ErrPlatformAdminActorForbidden
	ErrPlatformAdminSelfDeactivation = repository.ErrPlatformAdminSelfDeactivation
	ErrPlatformAdminTargetNotFound = repository.ErrPlatformAdminTargetNotFound
	ErrPlatformAdminLastAccount = repository.ErrPlatformAdminLastAccount
	ErrPlatformAdminEmailTaken = repository.ErrPlatformAdminEmailTaken
)

type platformAdminLifecycleStore interface {
	List(context.Context,int64) ([]repository.PlatformAdminAccount,error)
	Create(context.Context,int64,string,string) (*repository.PlatformAdminAccount,error)
	Deactivate(context.Context,int64,int64) error
}

type PlatformAdminLifecycleService struct {
	repository platformAdminLifecycleStore
	breachChecker PasswordBreachChecker
}

func NewPlatformAdminLifecycleService(store platformAdminLifecycleStore, checkers ...PasswordBreachChecker) *PlatformAdminLifecycleService {
	checker:=defaultPasswordBreachChecker
	if len(checkers)>0 { checker=checkers[0] }
	return &PlatformAdminLifecycleService{repository:store,breachChecker:checker}
}

func (s *PlatformAdminLifecycleService) List(ctx context.Context,actorID int64) ([]repository.PlatformAdminAccount,error) {
	if s==nil || s.repository==nil || actorID<=0 { return nil,ErrInvalidPlatformAdminAccount }
	return s.repository.List(ctx,actorID)
}

func (s *PlatformAdminLifecycleService) Create(ctx context.Context,actorID int64,email,password string) (*repository.PlatformAdminAccount,error) {
	if s==nil || s.repository==nil || actorID<=0 { return nil,ErrInvalidPlatformAdminAccount }
	email=strings.ToLower(strings.TrimSpace(email))
	parsed,err:=mail.ParseAddress(email)
	if err!=nil || parsed.Address!=email || len(email)>254 { return nil,ErrInvalidPlatformAdminAccount }
	hash,err:=hashPasswordWithChecker(ctx,password,s.breachChecker)
	if err!=nil { return nil,err }
	return s.repository.Create(ctx,actorID,email,hash)
}

func (s *PlatformAdminLifecycleService) Deactivate(ctx context.Context,actorID,targetID int64) error {
	if s==nil || s.repository==nil || actorID<=0 || targetID<=0 { return ErrInvalidPlatformAdminAccount }
	if actorID==targetID { return ErrPlatformAdminSelfDeactivation }
	if err:=s.repository.Deactivate(ctx,actorID,targetID); err!=nil { return fmt.Errorf("deactivate Platform Admin: %w",err) }
	return nil
}
