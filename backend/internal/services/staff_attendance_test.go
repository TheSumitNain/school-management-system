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
    listDailyPageCalls int
    listMonthlyPageCalls int
    lastLimit int
    lastOffset int
    lastDailyFilter repository.StaffAttendanceDailyFilter
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
func (f *fakeStaffAttendanceRepository) ListDailyPage(_ context.Context,_,_ int64,_ time.Time,limit,offset int)([]repository.StaffAttendanceRecord,int,error){f.listDailyPageCalls++;f.lastLimit=limit;f.lastOffset=offset;return []repository.StaffAttendanceRecord{},0,nil}
func (f *fakeStaffAttendanceRepository) ListDailyFilteredPage(_ context.Context, _, _ int64, _ time.Time, filter repository.StaffAttendanceDailyFilter, limit, offset int) ([]repository.StaffAttendanceRecord, int, error) {
    f.listDailyPageCalls++
    f.lastDailyFilter = filter
    f.lastLimit = limit
    f.lastOffset = offset
    return []repository.StaffAttendanceRecord{}, 0, nil
}
func (f *fakeStaffAttendanceRepository) ListMonthlyPage(_ context.Context,_,_ int64,_ time.Time,limit,offset int)([]repository.StaffAttendanceMonthlySummary,int,error){f.listMonthlyPageCalls++;f.lastLimit=limit;f.lastOffset=offset;return []repository.StaffAttendanceMonthlySummary{},0,nil}
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

func TestStaffAttendanceServiceEnforcesBulkBatchLimit(t *testing.T) {
	day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	t.Run("accepts maximum batch", func(t *testing.T) {
		fake := &fakeStaffAttendanceRepository{}
		service := NewStaffAttendanceService(fake)
		ids := make([]int64, 1000)
		for i := range ids { ids[i] = int64(i + 1) }
		count, err := service.MarkBulk(context.Background(), 1, 2, ids, day, "present", nil)
		if err != nil || count != 1000 || fake.markCalls != 1 {
			t.Fatalf("count=%d calls=%d err=%v; want 1000, 1, nil", count, fake.markCalls, err)
		}
	})
	t.Run("rejects oversized batch before repository", func(t *testing.T) {
		fake := &fakeStaffAttendanceRepository{}
		service := NewStaffAttendanceService(fake)
		ids := make([]int64, 1001)
		for i := range ids { ids[i] = int64(i + 1) }
		if _, err := service.MarkBulk(context.Background(), 1, 2, ids, day, "present", nil); !errors.Is(err, ErrInvalidStaffAttendance) {
			t.Fatalf("oversized batch error=%v want ErrInvalidStaffAttendance", err)
		}
		if fake.markCalls != 0 { t.Fatalf("repository calls=%d want=0", fake.markCalls) }
	})
}

func TestStaffAttendanceServiceRejectsInvalidBulkStatusAndRemarks(t *testing.T) {
	fake := &fakeStaffAttendanceRepository{}
	service := NewStaffAttendanceService(fake)
	day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	longRemarks := strings.Repeat("x", 501)
	cases := []struct {
		name string
		actor int64
		school int64
		status string
		remarks *string
	}{
		{name: "zero actor", actor: 0, school: 2, status: "present"},
		{name: "zero school", actor: 1, school: 0, status: "present"},
		{name: "invalid status", actor: 1, school: 2, status: "holiday"},
		{name: "oversized remarks", actor: 1, school: 2, status: "present", remarks: &longRemarks},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.MarkBulk(context.Background(), tc.actor, tc.school, []int64{3}, day, tc.status, tc.remarks); !errors.Is(err, ErrInvalidStaffAttendance) {
				t.Fatalf("error=%v want ErrInvalidStaffAttendance", err)
			}
		})
	}
	if fake.markCalls != 0 { t.Fatalf("repository calls=%d want=0", fake.markCalls) }
}

func TestStaffAttendanceServicePaginationBoundsAndDelegation(t *testing.T) {
    fake:=&fakeStaffAttendanceRepository{}; service:=NewStaffAttendanceService(fake)
    day:=time.Date(2026,11,22,0,0,0,0,time.UTC)
    if _,total,err:=service.ListDailyPage(context.Background(),1,2,day,50,100);err!=nil||total!=0||fake.listDailyPageCalls!=1||fake.lastOffset!=100 { t.Fatalf("daily pagination total=%d calls=%d offset=%d err=%v",total,fake.listDailyPageCalls,fake.lastOffset,err) }
    if _,total,err:=service.ListMonthlyPage(context.Background(),1,2,day,100,200);err!=nil||total!=0||fake.listMonthlyPageCalls!=1||fake.lastLimit!=100 { t.Fatalf("monthly pagination total=%d calls=%d limit=%d err=%v",total,fake.listMonthlyPageCalls,fake.lastLimit,err) }
    for _,v:=range [][2]int{{0,0},{101,0},{50,-1}} {
        if _,_,err:=service.ListDailyPage(context.Background(),1,2,day,v[0],v[1]);!errors.Is(err,ErrInvalidStaffAttendance){t.Fatalf("daily invalid pagination %v: %v",v,err)}
        if _,_,err:=service.ListMonthlyPage(context.Background(),1,2,day,v[0],v[1]);!errors.Is(err,ErrInvalidStaffAttendance){t.Fatalf("monthly invalid pagination %v: %v",v,err)}
    }
}

func TestStaffAttendanceServiceValidatesAndDelegatesDailyFilters(t *testing.T) {
    fake := &fakeStaffAttendanceRepository{}
    service := NewStaffAttendanceService(fake)
    day := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)

    filter := repository.StaffAttendanceDailyFilter{Search: "  Alice  ", Status: "unmarked"}
    if _, _, err := service.ListDailyFilteredPage(context.Background(), 1, 2, day, filter, 25, 50); err != nil {
        t.Fatalf("valid daily filters: %v", err)
    }
    if fake.listDailyPageCalls != 1 || fake.lastDailyFilter.Search != "Alice" || fake.lastDailyFilter.Status != "unmarked" || fake.lastLimit != 25 || fake.lastOffset != 50 {
        t.Fatalf("daily filter delegation mismatch: calls=%d filter=%+v limit=%d offset=%d", fake.listDailyPageCalls, fake.lastDailyFilter, fake.lastLimit, fake.lastOffset)
    }

    invalid := []repository.StaffAttendanceDailyFilter{
        {Search: strings.Repeat("x", 101), Status: "all"},
        {Search: "Alice", Status: "unknown"},
    }
    for _, candidate := range invalid {
        if _, _, err := service.ListDailyFilteredPage(context.Background(), 1, 2, day, candidate, 25, 0); !errors.Is(err, ErrInvalidStaffAttendance) {
            t.Fatalf("invalid daily filter %+v error=%v want ErrInvalidStaffAttendance", candidate, err)
        }
    }
    if fake.listDailyPageCalls != 1 {
        t.Fatalf("invalid filters reached repository: calls=%d want=1", fake.listDailyPageCalls)
    }
}
