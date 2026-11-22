package services

import (
	"context"
	"testing"

	"school-management-system/internal/repository"
)

type fakeStaffSubstitutionRepository struct { listCalls,createCalls,cancelCalls int }
func(f *fakeStaffSubstitutionRepository) ListStaffSubstitutions(context.Context,int64,int64,string,string,int64)([]repository.StaffSubstitution,error){f.listCalls++;return []repository.StaffSubstitution{},nil}
func(f *fakeStaffSubstitutionRepository) CreateStaffSubstitution(context.Context,int64,int64,int64,int64,int64,string,*int64,string)(*repository.StaffSubstitution,error){f.createCalls++;return &repository.StaffSubstitution{ID:1},nil}
func(f *fakeStaffSubstitutionRepository) CancelStaffSubstitution(context.Context,int64,int64,int64)error{f.cancelCalls++;return nil}

func TestStaffSubstitutionServiceRejectsInvalidDatesAndScopeBeforeRepository(t *testing.T) {
	repo:=&fakeStaffSubstitutionRepository{}
	svc:=NewStaffSubstitutionService(repo)
	if _,err:=svc.List(context.Background(),1,1,"2026-11-23","2026-11-22",0);err==nil{t.Fatal("expected reversed date range rejection")}
	if _,err:=svc.List(context.Background(),1,1,"2026-01-01","2028-01-01",0);err==nil{t.Fatal("expected oversized date range rejection")}
	if _,err:=svc.Create(context.Background(),1,1,3,4,4,"2026-11-23",nil,"cover");err==nil{t.Fatal("expected original and substitute teacher to differ")}
	if _,err:=svc.Create(context.Background(),1,1,3,4,5,"2026-11-23",nil,"  ");err==nil{t.Fatal("expected non-empty reason")}
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
