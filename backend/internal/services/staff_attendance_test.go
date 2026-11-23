package services

import (
    "context"
    "errors"
    "strings"
    "testing"
    "time"

    "school-management-system/internal/repository"
)

type fakeStaffAttendanceRepository struct {
    listDailyCalls int
    listMonthlyCalls int
    markCalls int
    updateCalls int
    lastStatus string
}

func (f *fakeStaffAttendanceRepository) ListDaily(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceRecord, error) {
    f.listDailyCalls++
    return []repository.StaffAttendanceRecord{}, nil
}
func (f *fakeStaffAttendanceRepository) ListMonthly(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceMonthlySummary, error) {
    f.listMonthlyCalls++
    return []repository.StaffAttendanceMonthlySummary{}, nil
}
func (f *fakeStaffAttendanceRepository) Mark(_ context.Context, _, _, _ int64, _ time.Time, status string, _ *string) (*repository.StaffAttendanceRecord, error) {
    f.markCalls++
    f.lastStatus = status
    return &repository.StaffAttendanceRecord{Status: &status}, nil
}
func (f *fakeStaffAttendanceRepository) MarkBulk(_ context.Context, _, _ int64, staffIDs []int64, _ time.Time, status string, _ *string) (int, error) {
    f.markCalls++
    f.lastStatus = status
    return len(staffIDs), nil
}
func (f *fakeStaffAttendanceRepository) Update(_ context.Context, _, _, _ int64, status string, _ *string) (*repository.StaffAttendanceRecord, error) {
    f.updateCalls++
    f.lastStatus = status
    return &repository.StaffAttendanceRecord{Status: &status}, nil
}

func TestStaffAttendanceServiceRejectsInvalidInputBeforeRepository(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

    if _, err := service.Mark(context.Background(), 1, 2, 3, day, "holiday", nil); !errors.Is(err, ErrInvalidStaffAttendance) {
        t.Fatalf("invalid status error=%v want ErrInvalidStaffAttendance", err)
    }
    longRemarks := strings.Repeat("x", 501)
    if _, err := service.Mark(context.Background(), 1, 2, 3, day, "present", &longRemarks); !errors.Is(err, ErrInvalidStaffAttendance) {
        t.Fatalf("long remarks error=%v want ErrInvalidStaffAttendance", err)
    }
    if fake.markCalls != 0 { t.Fatalf("repository mark calls=%d want=0", fake.markCalls) }
}

func TestStaffAttendanceServiceDelegatesValidOperations(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

    if _, err := service.ListDaily(context.Background(), 1, 2, day); err != nil { t.Fatalf("ListDaily: %v", err) }
    if _, err := service.Mark(context.Background(), 1, 2, 3, day, "late", nil); err != nil { t.Fatalf("Mark: %v", err) }
    if _, err := service.Update(context.Background(), 1, 2, 4, "half_day", nil); err != nil { t.Fatalf("Update: %v", err) }
    if fake.listDailyCalls != 1 || fake.markCalls != 1 || fake.updateCalls != 1 {
        t.Fatalf("repository calls list=%d mark=%d update=%d", fake.listDailyCalls, fake.markCalls, fake.updateCalls)
    }
}

func TestStaffAttendanceServiceRejectsInvalidBulkTargets(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
    cases := [][]int64{nil, {}, {1, 1}, {1, 0}}
    for _, ids := range cases {
        if _, err := service.MarkBulk(context.Background(), 1, 2, ids, day, "present", nil); !errors.Is(err, ErrInvalidStaffAttendance) {
            t.Fatalf("invalid bulk target list %v error=%v want ErrInvalidStaffAttendance", ids, err)
        }
    }
    if fake.markCalls != 0 { t.Fatalf("repository bulk calls=%d want=0", fake.markCalls) }
}

func TestStaffAttendanceServiceDelegatesValidBulkMark(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
    count, err := service.MarkBulk(context.Background(), 1, 2, []int64{3, 4}, day, "present", nil)
    if err != nil || count != 2 || fake.markCalls != 1 || fake.lastStatus != "present" {
        t.Fatalf("bulk result count=%d calls=%d status=%q err=%v", count, fake.markCalls, fake.lastStatus, err)
    }
}

func TestStaffAttendanceServiceAcceptsYearOneCalendarDates(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    firstDate := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)

    if _, err := service.ListDaily(context.Background(), 1, 2, firstDate); err != nil {
        t.Fatalf("ListDaily year-one date: %v", err)
    }
    if _, err := service.ListMonthly(context.Background(), 1, 2, firstDate); err != nil {
        t.Fatalf("ListMonthly year-one month: %v", err)
    }
    if _, err := service.Mark(context.Background(), 1, 2, 3, firstDate, "present", nil); err != nil {
        t.Fatalf("Mark year-one date: %v", err)
    }
    if _, err := service.MarkBulk(context.Background(), 1, 2, []int64{3}, firstDate, "present", nil); err != nil {
        t.Fatalf("MarkBulk year-one date: %v", err)
    }
    if fake.listDailyCalls != 1 || fake.listMonthlyCalls != 1 || fake.markCalls != 2 {
        t.Fatalf("repository calls daily=%d monthly=%d mark=%d", fake.listDailyCalls, fake.listMonthlyCalls, fake.markCalls)
    }
}
