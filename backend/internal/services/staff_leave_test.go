package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/repository"
)

type staffLeaveRepositoryStub struct {
	createCalls int
	created     repository.StaffLeaveRequest
	saveHolidayCalls int
	savedHoliday repository.SchoolHoliday
}

func (s *staffLeaveRepositoryStub) ListTypes(context.Context, int64, int64) ([]repository.StaffLeaveType, error) {
	return nil, nil
}
func (s *staffLeaveRepositoryStub) SaveType(context.Context, int64, int64, int64, repository.StaffLeaveType) (*repository.StaffLeaveType, error) {
	return nil, nil
}
func (s *staffLeaveRepositoryStub) ListRequests(context.Context, int64, int64, string, int64, string, string, string) ([]repository.StaffLeaveRequest, error) {
	return nil, nil
}
func (s *staffLeaveRepositoryStub) CreateRequest(_ context.Context, _, _ int64, item repository.StaffLeaveRequest) (*repository.StaffLeaveRequest, error) {
	s.createCalls++
	s.created = item
	return &item, nil
}
func (s *staffLeaveRepositoryStub) Decide(context.Context, int64, int64, int64, bool, *string) error {
	return nil
}
func (s *staffLeaveRepositoryStub) Cancel(context.Context, int64, int64, int64) error {
	return nil
}
func (s *staffLeaveRepositoryStub) ListHolidays(context.Context, int64, int64, string, string, bool) ([]repository.SchoolHoliday, error) {
	return nil, nil
}
func (s *staffLeaveRepositoryStub) SaveHoliday(_ context.Context, _, _ , _ int64, item repository.SchoolHoliday) (*repository.SchoolHoliday, error) {
	s.saveHolidayCalls++
	s.savedHoliday = item
	return &item, nil
}

func TestStaffLeaveServiceRejectsInvalidRequestBeforeRepository(t *testing.T) {
	repo := &staffLeaveRepositoryStub{}
	service := NewStaffLeaveService(repo)
	firstHalf := "first_half"

	tests := []struct {
		name string
		item repository.StaffLeaveRequest
	}{
		{"invalid start date", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "22-11-2026", EndDate: "2026-11-22", DurationType: "full_day", Reason: "Personal"}},
		{"end before start", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "2026-11-23", EndDate: "2026-11-22", DurationType: "full_day", Reason: "Personal"}},
		{"missing reason", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "2026-11-22", EndDate: "2026-11-22", DurationType: "full_day"}},
		{"unknown duration", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "2026-11-22", EndDate: "2026-11-22", DurationType: "hourly", Reason: "Personal"}},
		{"half day spans multiple dates", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "2026-11-22", EndDate: "2026-11-23", DurationType: "half_day", HalfDaySession: &firstHalf, Reason: "Personal"}},
		{"half day missing session", repository.StaffLeaveRequest{StaffPrincipalID: 2, LeaveTypeID: 3, StartDate: "2026-11-22", EndDate: "2026-11-22", DurationType: "half_day", Reason: "Personal"}},
	} 

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateRequest(context.Background(), 1, 1, test.item)
			if !errors.Is(err, ErrInvalidStaffLeave) {
				t.Fatalf("expected ErrInvalidStaffLeave, got %v", err)
			}
		})
	}
	if repo.createCalls != 0 {
		t.Fatalf("invalid requests reached repository %d times", repo.createCalls)
	}
}

func TestStaffLeaveServiceNormalizesValidRequest(t *testing.T) {
	repo := &staffLeaveRepositoryStub{}
	service := NewStaffLeaveService(repo)
	session := " first_half "
	item := repository.StaffLeaveRequest{
		StaffPrincipalID: 2, LeaveTypeID: 3,
		StartDate: "2026-11-22", EndDate: "2026-11-22",
		DurationType: "half_day", HalfDaySession: &session, Reason: "  Personal leave  ",
	}
	_, err := service.CreateRequest(context.Background(), 1, 1, item)
	if err != nil {
		t.Fatalf("CreateRequest returned error: %v", err)
	}
	if repo.createCalls != 1 || repo.created.Reason != "Personal leave" || repo.created.HalfDaySession == nil || *repo.created.HalfDaySession != "first_half" {
		t.Fatalf("request was not normalized before repository call: %+v", repo.created)
	}
}

func TestStaffLeaveServiceValidatesHolidayInput(t *testing.T) {
	repo := &staffLeaveRepositoryStub{}
	service := NewStaffLeaveService(repo)
	description := "  School annual day  "
	item := repository.SchoolHoliday{HolidayDate: "2026-11-22", Name: "  Annual Day  ", Description: &description}

	_, err := service.SaveHoliday(context.Background(), 1, 1, 0, item)
	if err != nil {
		t.Fatalf("SaveHoliday returned error: %v", err)
	}
	if repo.saveHolidayCalls != 1 || repo.savedHoliday.Name != "Annual Day" || repo.savedHoliday.Description == nil || *repo.savedHoliday.Description != "School annual day" {
		t.Fatalf("holiday was not normalized before repository call: %+v", repo.savedHoliday)
	}

	item.HolidayDate = "not-a-date"
	if _, err := service.SaveHoliday(context.Background(), 1, 1, 0, item); !errors.Is(err, ErrInvalidStaffLeave) {
		t.Fatalf("expected invalid date error, got %v", err)
	}
	if repo.saveHolidayCalls != 1 {
		t.Fatalf("invalid holiday reached repository; calls=%d", repo.saveHolidayCalls)
	}
}

func TestStaffLeaveServiceRequiresCommentForRejection(t *testing.T) {
	repo := &staffLeaveRepositoryStub{}
	service := NewStaffLeaveService(repo)
	if err := service.Decide(context.Background(), 1, 1, 4, false, nil); !errors.Is(err, ErrInvalidStaffLeave) {
		t.Fatalf("expected rejection without comment to fail, got %v", err)
	}
	if err := service.Decide(context.Background(), 1, 1, 4, false, new(string)); !errors.Is(err, ErrInvalidStaffLeave) {
		t.Fatalf("expected blank rejection comment to fail, got %v", err)
	}
}

func TestStaffLeaveServiceValidatesReportDateRange(t *testing.T) {
 repo := &staffLeaveRepositoryStub{}
 service := NewStaffLeaveService(repo)
 tests := []struct{name string; onDate,fromDate,toDate string; wantErr bool}{
  {"valid range","", "2026-11-01","2026-11-30",false},
  {"reversed range","", "2026-11-30","2026-11-01",true},
  {"missing range end","", "2026-11-01","",true},
  {"invalid date","", "2026-11-01","2026-11-31",true},
  {"range mixed with single date","2026-11-15","2026-11-01","2026-11-30",true},
  {"valid single date","2026-11-15","","",false},
 }
 for _,tt:=range tests { t.Run(tt.name,func(t *testing.T){
  callsBefore:=repo.listRequestsCalls
  _,err:=service.ListRequests(context.Background(),1,1,"approved",0,tt.onDate,tt.fromDate,tt.toDate)
  if tt.wantErr&&!errors.Is(err,ErrInvalidStaffLeave){t.Fatalf("error=%v want invalid leave",err)}
  if !tt.wantErr&&err!=nil{t.Fatalf("unexpected error: %v",err)}
  wantCalls:=callsBefore
  if !tt.wantErr { wantCalls++ }
  if repo.listRequestsCalls!=wantCalls { t.Fatalf("repository calls=%d want=%d (invalid filters must be rejected before database access)",repo.listRequestsCalls,wantCalls) }
 })}
}

func TestStaffLeaveServiceRejectsUnknownRequestStatusBeforeRepository(t *testing.T) {
 repo:=&staffLeaveRepositoryStub{}
 service:=NewStaffLeaveService(repo)
 for _,status:=range []string{"PENDING","unknown","pending unexpected"} {
  if _,err:=service.ListRequests(context.Background(),1,1,status,0,"","","");!errors.Is(err,ErrInvalidStaffLeave) {
   t.Errorf("status %q: error=%v want ErrInvalidStaffLeave",status,err)
  }
 }
 if repo.listRequestsCalls!=0 { t.Fatalf("invalid statuses reached repository %d times",repo.listRequestsCalls) }
}
