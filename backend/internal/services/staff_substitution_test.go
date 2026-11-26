package services

import (
	"context"
	"testing"

	"school-management-system/internal/repository"
)

type fakeStaffSubstitutionRepository struct { listCalls,createCalls,cancelCalls,listPageCalls int; pageFrom,pageTo string; pageStaffID int64; pageNumber,pageSize int }
func(f *fakeStaffSubstitutionRepository) ListStaffSubstitutions(context.Context,int64,int64,string,string,int64)([]repository.StaffSubstitution,error){f.listCalls++;return []repository.StaffSubstitution{},nil}
func(f *fakeStaffSubstitutionRepository) ListStaffSubstitutionsPage(_ context.Context,_,_ int64,from,to string,staffID int64,page,size int)(repository.StaffSubstitutionPage,error){f.listPageCalls++;f.pageFrom,f.pageTo,f.pageStaffID,f.pageNumber,f.pageSize=from,to,staffID,page,size;return repository.StaffSubstitutionPage{Items:[]repository.StaffSubstitution{}},nil}
func(f *fakeStaffSubstitutionRepository) CreateStaffSubstitution(context.Context,int64,int64,int64,int64,int64,string,*int64,string)(*repository.StaffSubstitution,error){f.createCalls++;return &repository.StaffSubstitution{ID:1},nil}
func(f *fakeStaffSubstitutionRepository) CancelStaffSubstitution(context.Context,int64,int64,int64)error{f.cancelCalls++;return nil}

func TestStaffSubstitutionServiceRejectsInvalidDatesAndScopeBeforeRepository(t *testing.T) {
	repo:=&fakeStaffSubstitutionRepository{}
	svc:=NewStaffSubstitutionService(repo)
	if _,err:=svc.List(context.Background(),1,1,"2026-11-23","2026-11-22",0);err==nil{t.Fatal("expected reversed date range rejection")}
	if _,err:=svc.List(context.Background(),1,1,"2026-01-01","2028-01-01",0);err==nil{t.Fatal("expected oversized date range rejection")}
	if _,err:=svc.List(context.Background(),1,1,"0000-01-01","0000-01-02",0);err==nil{t.Fatal("expected year-zero date rejection")}
	if _,err:=svc.Create(context.Background(),1,1,3,4,4,"2026-11-23",nil,"cover");err==nil{t.Fatal("expected original and substitute teacher to differ")}
	if _,err:=svc.Create(context.Background(),1,1,3,4,5,"2026-11-23",nil,"  ");err==nil{t.Fatal("expected non-empty reason")}
	if _,err:=svc.Create(context.Background(),1,1,3,4,5,"0000-01-01",nil,"cover");err==nil{t.Fatal("expected year-zero cover date rejection")}
	if repo.listCalls!=0||repo.createCalls!=0{t.Fatalf("invalid input reached repository: list=%d create=%d",repo.listCalls,repo.createCalls)}
}

func TestStaffSubstitutionServiceAcceptsValidDatedCoverage(t *testing.T) {
	repo:=&fakeStaffSubstitutionRepository{}
	svc:=NewStaffSubstitutionService(repo)
	item,err:=svc.Create(context.Background(),1,2,3,4,5,"2026-11-23",nil,"Teacher on approved leave")
	if err!=nil{t.Fatalf("create valid substitution: %v",err)}
	if item.ID!=1||repo.createCalls!=1{t.Fatalf("unexpected repository result: item=%+v calls=%d",item,repo.createCalls)}
	if err:=svc.Cancel(context.Background(),1,2,item.ID);err!=nil{t.Fatalf("cancel substitution: %v",err)}
	if repo.cancelCalls!=1{t.Fatalf("expected one cancellation call, got %d",repo.cancelCalls)}
}

func TestStaffSubstitutionPaginationRejectsOffsetOverflowBeforeRepository(t *testing.T) {
	repo := &fakeStaffSubstitutionRepository{}
	svc := NewStaffSubstitutionService(repo)
	maxInt := int(^uint(0) >> 1)
	if _, err := svc.ListPage(context.Background(), 1, 1, "2026-11-01", "2026-11-30", 0, maxInt, 100); err == nil {
		t.Fatal("expected oversized page to be rejected")
	}
	if repo.listPageCalls != 0 {
		t.Fatalf("oversized page reached repository %d times", repo.listPageCalls)
	}
}


func TestStaffSubstitutionPaginationValidatesDateRangeAndForwardsPage(t *testing.T) {
	repo := &fakeStaffSubstitutionRepository{}
	svc := NewStaffSubstitutionService(repo)
	tests := []struct {
		name, from, to string
		staffID int64
		page, size int
		wantErr bool
	}{
		{name: "valid page", from: "2026-11-01", to: "2026-11-30", staffID: 9, page: 3, size: 20},
		{name: "missing start", to: "2026-11-30", page: 1, size: 50, wantErr: true},
		{name: "missing end", from: "2026-11-01", page: 1, size: 50, wantErr: true},
		{name: "reversed range", from: "2026-11-30", to: "2026-11-01", page: 1, size: 50, wantErr: true},
		{name: "range too long", from: "2026-01-01", to: "2027-01-03", page: 1, size: 50, wantErr: true},
		{name: "negative staff ID", from: "2026-11-01", to: "2026-11-30", staffID: -1, page: 1, size: 50, wantErr: true},
		{name: "zero page", from: "2026-11-01", to: "2026-11-30", page: 0, size: 50, wantErr: true},
		{name: "page size above maximum", from: "2026-11-01", to: "2026-11-30", page: 1, size: 101, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := repo.listPageCalls
			_, err := svc.ListPage(context.Background(), 1, 2, tt.from, tt.to, tt.staffID, tt.page, tt.size)
			if tt.wantErr {
				if err == nil { t.Fatal("invalid page/filter was accepted") }
				if repo.listPageCalls != before { t.Fatal("invalid page/filter reached repository") }
				return
			}
			if err != nil { t.Fatalf("valid page rejected: %v", err) }
			if repo.listPageCalls != before+1 || repo.pageFrom != tt.from || repo.pageTo != tt.to || repo.pageStaffID != tt.staffID || repo.pageNumber != tt.page || repo.pageSize != tt.size {
				t.Fatalf("page forwarding mismatch: calls=%d from=%q to=%q staff=%d page=%d size=%d", repo.listPageCalls, repo.pageFrom, repo.pageTo, repo.pageStaffID, repo.pageNumber, repo.pageSize)
			}
		})
	}
}
