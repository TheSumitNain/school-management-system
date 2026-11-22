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
    Update(context.Context, int64, int64, int64, string, *string) (*repository.StaffAttendanceRecord, error)
}

type StaffAttendanceService struct { repository staffAttendanceRepository }

func NewStaffAttendanceService(repo staffAttendanceRepository) *StaffAttendanceService {
    return &StaffAttendanceService{repository: repo}
}

func (s *StaffAttendanceService) ListDaily(ctx context.Context, actor, school int64, day time.Time) ([]repository.StaffAttendanceRecord, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || day.IsZero() { return nil, ErrInvalidStaffAttendance }
    return s.repository.ListDaily(ctx, actor, school, day)
}

func (s *StaffAttendanceService) ListMonthly(ctx context.Context, actor, school int64, month time.Time) ([]repository.StaffAttendanceMonthlySummary, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || month.IsZero() { return nil, ErrInvalidStaffAttendance }
    return s.repository.ListMonthly(ctx, actor, school, month)
}

func (s *StaffAttendanceService) Mark(ctx context.Context, actor, school, staff int64, day time.Time, status string, remarks *string) (*repository.StaffAttendanceRecord, error) {
    if s == nil || s.repository == nil || actor <= 0 || school <= 0 || staff <= 0 || day.IsZero() { return nil, ErrInvalidStaffAttendance }
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
