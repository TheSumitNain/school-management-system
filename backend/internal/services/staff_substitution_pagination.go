package services

import (
	"context"
	"time"

	"school-management-system/internal/repository"
)

type staffSubstitutionPaginatedRepository interface {
	ListStaffSubstitutionsPage(context.Context, int64, int64, string, string, int64, int, int) (repository.StaffSubstitutionPage, error)
}

func (s *StaffSubstitutionService) ListPage(ctx context.Context, actor, school int64, from, to string, staffID int64, page, pageSize int) (repository.StaffSubstitutionPage, error) {
	empty := repository.StaffSubstitutionPage{Items: []repository.StaffSubstitution{}}
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || staffID < 0 || page < 1 || pageSize < 1 || pageSize > 100 { return empty, ErrInvalidStaffSubstitution }
	start, err := parseStaffManagementDate(from); if err != nil { return empty, ErrInvalidStaffSubstitution }
	end, err := parseStaffManagementDate(to); if err != nil || end.Before(start) || end.Sub(start) > 366*24*time.Hour { return empty, ErrInvalidStaffSubstitution }
	repo, ok := s.repo.(staffSubstitutionPaginatedRepository); if !ok { return empty, ErrInvalidStaffSubstitution }
	return repo.ListStaffSubstitutionsPage(ctx, actor, school, from, to, staffID, page, pageSize)
}
