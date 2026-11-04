package services

import (
	"context"
	"time"

	"school-management-system/internal/repository"
)

type attendanceRepository interface {
	ListAttendance(context.Context, int64, int64, int64, int64, time.Time) ([]repository.AttendanceRecord, error)
	MarkAttendance(context.Context, int64, int64, int64, time.Time, string, *string) (*repository.AttendanceRecord, error)
	UpdateAttendance(context.Context, int64, int64, int64, string, *string) (*repository.AttendanceRecord, error)
}

type AttendanceService struct { repository attendanceRepository }

func NewAttendanceService(repo attendanceRepository) *AttendanceService { return &AttendanceService{repository: repo} }

func (s *AttendanceService) List(ctx context.Context, actorID, schoolID, yearID, sectionID int64, day time.Time) ([]repository.AttendanceRecord, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAttendanceAccessDenied }
	return s.repository.ListAttendance(ctx, actorID, schoolID, yearID, sectionID, day)
}
func (s *AttendanceService) Mark(ctx context.Context, actorID, schoolID, enrollmentID int64, day time.Time, status string, remarks *string) (*repository.AttendanceRecord, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAttendanceAccessDenied }
	return s.repository.MarkAttendance(ctx, actorID, schoolID, enrollmentID, day, status, remarks)
}
func (s *AttendanceService) Update(ctx context.Context, actorID, schoolID, recordID int64, status string, remarks *string) (*repository.AttendanceRecord, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAttendanceAccessDenied }
	return s.repository.UpdateAttendance(ctx, actorID, schoolID, recordID, status, remarks)
}
