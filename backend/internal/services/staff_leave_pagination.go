package services

import (
	"context"
	"time"

	"school-management-system/internal/repository"
)

type staffLeavePaginatedRepository interface {
	ListRequestsPage(context.Context, int64, int64, string, int64, string, string, string, int, int) (repository.StaffLeaveRequestPage, error)
}

func (s *StaffLeaveService) ListRequestsPage(ctx context.Context, actor, school int64, status string, staffID int64, onDate, fromDate, toDate string, page, pageSize int) (repository.StaffLeaveRequestPage, error) {
	empty := repository.StaffLeaveRequestPage{Items: []repository.StaffLeaveRequest{}}
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || staffID < 0 || page < 1 || pageSize < 1 || pageSize > 100 || page-1 > int(^uint(0)>>1)/pageSize {
		return empty, ErrInvalidStaffLeave
	}
	switch status { case "", "pending", "approved", "rejected", "cancelled": default: return empty, ErrInvalidStaffLeave }
	if onDate != "" && (fromDate != "" || toDate != "") { return empty, ErrInvalidStaffLeave }
	if onDate != "" { if _, err := parseStaffManagementDate(onDate); err != nil { return empty, ErrInvalidStaffLeave } }
	if (fromDate == "") != (toDate == "") { return empty, ErrInvalidStaffLeave }
	if fromDate != "" {
		start, err1 := parseStaffManagementDate(fromDate); end, err2 := parseStaffManagementDate(toDate)
		if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > 366*24*time.Hour { return empty, ErrInvalidStaffLeave }
	}
	repo, ok := s.repo.(staffLeavePaginatedRepository)
	if !ok { return empty, ErrInvalidStaffLeave }
	return repo.ListRequestsPage(ctx, actor, school, status, staffID, onDate, fromDate, toDate, page, pageSize)
}
