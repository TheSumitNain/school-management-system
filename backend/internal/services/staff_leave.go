package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

var ErrInvalidStaffLeave = errors.New("invalid staff leave input")

func parseStaffManagementDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidStaffLeave
	}
	return parsed, nil
}

type staffLeaveRepository interface {
	ListTypes(context.Context, int64, int64) ([]repository.StaffLeaveType, error)
	SaveType(context.Context, int64, int64, int64, repository.StaffLeaveType) (*repository.StaffLeaveType, error)
	ListRequests(context.Context, int64, int64, string, int64, string, string, string) ([]repository.StaffLeaveRequest, error)
	CreateRequest(context.Context, int64, int64, repository.StaffLeaveRequest) (*repository.StaffLeaveRequest, error)
	Decide(context.Context, int64, int64, int64, bool, *string) error
	Cancel(context.Context, int64, int64, int64) error
	ListHolidays(context.Context, int64, int64, string, string, bool) ([]repository.SchoolHoliday, error)
	SaveHoliday(context.Context, int64, int64, int64, repository.SchoolHoliday) (*repository.SchoolHoliday, error)
}

type StaffLeaveService struct {
	repo staffLeaveRepository
}

func NewStaffLeaveService(repo staffLeaveRepository) *StaffLeaveService {
	return &StaffLeaveService{repo: repo}
}

func (s *StaffLeaveService) ListTypes(ctx context.Context, actor, school int64) ([]repository.StaffLeaveType, error) {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 {
		return nil, ErrInvalidStaffLeave
	}
	return s.repo.ListTypes(ctx, actor, school)
}

func (s *StaffLeaveService) SaveType(ctx context.Context, actor, school, id int64, item repository.StaffLeaveType) (*repository.StaffLeaveType, error) {
	item.Name = strings.TrimSpace(item.Name)
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || id < 0 ||
		len([]rune(item.Name)) < 2 || len([]rune(item.Name)) > 100 {
		return nil, ErrInvalidStaffLeave
	}
	return s.repo.SaveType(ctx, actor, school, id, item)
}

func (s *StaffLeaveService) ListRequests(ctx context.Context, actor, school int64, status string, staffID int64, onDate, fromDate, toDate string) ([]repository.StaffLeaveRequest, error) {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || staffID < 0 {
		return nil, ErrInvalidStaffLeave
	}
	status = strings.TrimSpace(status)
	switch status {
	case "", "pending", "approved", "rejected", "cancelled":
	default:
		return nil, ErrInvalidStaffLeave
	}
	if onDate != "" && (fromDate != "" || toDate != "") {
		return nil, ErrInvalidStaffLeave
	}
	if onDate != "" {
		if _, err := parseStaffManagementDate(onDate); err != nil {
			return nil, ErrInvalidStaffLeave
		}
	}
	if (fromDate == "") != (toDate == "") {
		return nil, ErrInvalidStaffLeave
	}
	if fromDate != "" {
		start, startErr := parseStaffManagementDate(fromDate)
		end, endErr := parseStaffManagementDate(toDate)
		if startErr != nil || endErr != nil || end.Before(start) || end.Sub(start) > 366*24*time.Hour {
			return nil, ErrInvalidStaffLeave
		}
	}
	return s.repo.ListRequests(ctx, actor, school, status, staffID, onDate, fromDate, toDate)
}

func (s *StaffLeaveService) CreateRequest(ctx context.Context, actor, school int64, item repository.StaffLeaveRequest) (*repository.StaffLeaveRequest, error) {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || item.StaffPrincipalID <= 0 || item.LeaveTypeID <= 0 {
		return nil, ErrInvalidStaffLeave
	}
	start, err := parseStaffManagementDate(item.StartDate)
	if err != nil {
		return nil, ErrInvalidStaffLeave
	}
	end, err := parseStaffManagementDate(item.EndDate)
	if err != nil || end.Before(start) {
		return nil, ErrInvalidStaffLeave
	}
	item.DurationType = strings.TrimSpace(item.DurationType)
	item.Reason = strings.TrimSpace(item.Reason)
	if item.Reason == "" || len([]rune(item.Reason)) > 1000 {
		return nil, ErrInvalidStaffLeave
	}
	switch item.DurationType {
	case "full_day":
		item.HalfDaySession = nil
	case "half_day":
		if !start.Equal(end) || item.HalfDaySession == nil {
			return nil, ErrInvalidStaffLeave
		}
		session := strings.TrimSpace(*item.HalfDaySession)
		if session != "first_half" && session != "second_half" {
			return nil, ErrInvalidStaffLeave
		}
		item.HalfDaySession = &session
	default:
		return nil, ErrInvalidStaffLeave
	}
	return s.repo.CreateRequest(ctx, actor, school, item)
}

func (s *StaffLeaveService) Decide(ctx context.Context, actor, school, requestID int64, approve bool, comment *string) error {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || requestID <= 0 {
		return ErrInvalidStaffLeave
	}
	if comment != nil {
		value := strings.TrimSpace(*comment)
		if len([]rune(value)) > 1000 {
			return ErrInvalidStaffLeave
		}
		comment = &value
	}
	if !approve && (comment == nil || *comment == "") {
		return ErrInvalidStaffLeave
	}
	return s.repo.Decide(ctx, actor, school, requestID, approve, comment)
}

func (s *StaffLeaveService) Cancel(ctx context.Context, actor, school, requestID int64) error {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || requestID <= 0 {
		return ErrInvalidStaffLeave
	}
	return s.repo.Cancel(ctx, actor, school, requestID)
}

func (s *StaffLeaveService) ListHolidays(ctx context.Context, actor, school int64, from, to string, includeInactive bool) ([]repository.SchoolHoliday, error) {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 {
		return nil, ErrInvalidStaffLeave
	}
	start, err := parseStaffManagementDate(from)
	if err != nil {
		return nil, ErrInvalidStaffLeave
	}
	end, err := parseStaffManagementDate(to)
	if err != nil || end.Before(start) || end.Sub(start) > 366*24*time.Hour {
		return nil, ErrInvalidStaffLeave
	}
	return s.repo.ListHolidays(ctx, actor, school, from, to, includeInactive)
}

func (s *StaffLeaveService) SaveHoliday(ctx context.Context, actor, school, id int64, item repository.SchoolHoliday) (*repository.SchoolHoliday, error) {
	if s == nil || s.repo == nil || actor <= 0 || school <= 0 || id < 0 {
		return nil, ErrInvalidStaffLeave
	}
	if _, err := parseStaffManagementDate(item.HolidayDate); err != nil {
		return nil, ErrInvalidStaffLeave
	}
	item.Name = strings.TrimSpace(item.Name)
	if len([]rune(item.Name)) < 2 || len([]rune(item.Name)) > 160 {
		return nil, ErrInvalidStaffLeave
	}
	if item.Description != nil {
		description := strings.TrimSpace(*item.Description)
		if len([]rune(description)) > 1000 {
			return nil, ErrInvalidStaffLeave
		}
		if description == "" {
			item.Description = nil
		} else {
			item.Description = &description
		}
	}
	return s.repo.SaveHoliday(ctx, actor, school, id, item)
}
