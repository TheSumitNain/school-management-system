package services

import (
	"context"
	"errors"

	"school-management-system/internal/repository"
)

var (
	ErrPlatformAuditForbidden = repository.ErrPlatformAuditForbidden
	ErrInvalidPlatformAuditQuery = repository.ErrInvalidPlatformAuditQuery
)

type platformAuditStore interface {
	List(context.Context, int64, int, int) (*repository.PlatformAuditPage, error)
}

type PlatformAuditService struct { repository platformAuditStore }

func NewPlatformAuditService(store platformAuditStore) *PlatformAuditService {
	return &PlatformAuditService{repository: store}
}

func (s *PlatformAuditService) List(ctx context.Context, actorID int64, limit, offset int) (*repository.PlatformAuditPage, error) {
	if s == nil || s.repository == nil || actorID <= 0 {
		return nil, errors.New("Platform Admin audit service unavailable")
	}
	return s.repository.List(ctx, actorID, limit, offset)
}
