package services

import (
    "context"
    "errors"
    "strings"
    "time"

    "school-management-system/internal/repository"
)

var ErrInvalidStaffAttendance = errors.New("invalid staff attendance input")

type staffAttendanceRepository interface {
    ListDaily(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceRecord, error)
    ListMonthly(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceMonthlySummary, error)
    Mark(context.Context, int64, int64, int64, time.Time, string, *string) (*repository.StaffAttendanceRecord, error)
    MarkBulk(context.Context, int64, int64, []int64, time.Time, string, *string) (int, error)
    Update(context.Context, int64, int64, int64, string, *string) (*repository.StaffAttendanceRecord, error)
}

type StaffAttendanceService struct { repository staffAttendanceRepository }

func NewStaffAttendanceService(repo staffAttendanceRepository) *StaffAttendanceService {
    return &StaffAttendanceService{repository: repo}
}

func (s *StaffAttendanceService) ListDaily(ctx context.Context, actor, school int64, day time.Time) ([]repository.StaffAttendanceRecord, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || day.Year() < 1 { return nil, ErrInvalidStaffAttendance }
    return s.repository.ListDaily(ctx, actor, school, day)
}

func (s *StaffAttendanceService) ListMonthly(ctx context.Context, actor, school int64, month time.Time) ([]repository.StaffAttendanceMonthlySummary, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || month.Year() < 1 { return nil, ErrInvalidStaffAttendance }
    return s.repository.ListMonthly(ctx, actor, school, month)
}

func (s *StaffAttendanceService) Mark(ctx context.Context, actor, school, staff int64, day time.Time, status string, remarks *string) (*repository.StaffAttendanceRecord, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || staff <= 0 || day.Year() < 1 { return nil, ErrInvalidStaffAttendance }
    status = strings.TrimSpace(status)
    if status != "present" && status != "absent" && status != "late" && status != "half_day" { return nil, ErrInvalidStaffAttendance }
    if remarks != nil && len([]rune(strings.TrimSpace(*remarks))) > 500 { return nil, ErrInvalidStaffAttendance }
    return s.repository.Mark(ctx, actor, school, staff, day, status, remarks)
}

func (s *StaffAttendanceService) Update(ctx context.Context, actor, school, record int64, status string, remarks *string) (*repository.StaffAttendanceRecord, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || record <= 0 { return nil, ErrInvalidStaffAttendance }
    status = strings.TrimSpace(status)
    if status != "present" && status != "absent" && status != "late" && status != "half_day" { return nil, ErrInvalidStaffAttendance }
    if remarks != nil && len([]rune(strings.TrimSpace(*remarks))) > 500 { return nil, ErrInvalidStaffAttendance }
    return s.repository.Update(ctx, actor, school, record, status, remarks)
}

func (s *StaffAttendanceService) MarkBulk(ctx context.Context, actor, school int64, staffIDs []int64, day time.Time, status string, remarks *string) (int, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || len(staffIDs) == 0 || len(staffIDs) > 1000 || day.Year() < 1 {
        return 0, ErrInvalidStaffAttendance
    }
    status = strings.TrimSpace(status)
    if status != "present" && status != "absent" && status != "late" && status != "half_day" {
        return 0, ErrInvalidStaffAttendance
    }
    if remarks != nil && len([]rune(strings.TrimSpace(*remarks))) > 500 {
        return 0, ErrInvalidStaffAttendance
    }
    seen := make(map[int64]struct{}, len(staffIDs))
    for _, staffID := range staffIDs {
        if staffID <= 0 { return 0, ErrInvalidStaffAttendance }
        if _, exists := seen[staffID]; exists { return 0, ErrInvalidStaffAttendance }
        seen[staffID] = struct{}{}
    }
    return s.repository.MarkBulk(ctx, actor, school, staffIDs, day, status, remarks)
}
