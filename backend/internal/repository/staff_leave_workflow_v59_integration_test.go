package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type staffLeaveIntegrationFixture struct {
	pool *pgxpool.Pool
	repo *StaffLeaveRepository
	ctx context.Context
	schoolID int64
	adminPrincipalID int64
	staffPrincipalID int64
	leaveTypeID int64
}

func newStaffLeaveIntegrationFixture(t *testing.T) *staffLeaveIntegrationFixture {
	t.Helper()
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	var adminPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id,school_id FROM users WHERE id=$1`, userID).Scan(&adminPrincipalID,&schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v",err)
	}
	platformAdminID := ensurePlatformAdminLifecycleActor(t,pool)
	var roleID int64
	if err := pool.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND role_key='school_admin' AND principal_type='staff' AND scope_type='school' AND is_system=TRUE`,schoolID).Scan(&roleID); err != nil {
		t.Fatalf("load School Admin role: %v",err)
	}
	if _,err := pool.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`,adminPrincipalID,roleID,platformAdminID);err!=nil{
		t.Fatalf("grant School Admin fixture role: %v",err)
	}
	suffix:=fmt.Sprint(time.Now().UnixNano())
	if _,err:=pool.Exec(ctx,`INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number,employment_status) VALUES($1,$2,$3,$4,'School Administrator','9876543210','active')`,adminPrincipalID,schoolID,"LEAVE-ADMIN-"+suffix,"Leave Test Admin");err!=nil{
		t.Fatalf("create admin employee profile: %v",err)
	}
	var staffPrincipalID int64
	if err:=pool.QueryRow(ctx,`INSERT INTO users(school_id,username,password_hash,status) VALUES($1,$2,'integration-password-hash','active') RETURNING principal_id`,schoolID,"leave-staff-"+suffix).Scan(&staffPrincipalID);err!=nil{
		t.Fatalf("create leave applicant account: %v",err)
	}
	if _,err:=pool.Exec(ctx,`INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number,employment_status) VALUES($1,$2,$3,'Leave Applicant','Teacher','9876543211','active')`,staffPrincipalID,schoolID,"LEAVE-STAFF-"+suffix);err!=nil{
		t.Fatalf("create leave applicant profile: %v",err)
	}
	repo:=NewStaffLeaveRepository(pool)
	leaveType,err:=repo.SaveType(ctx,adminPrincipalID,schoolID,0,StaffLeaveType{Name:"Integration Leave "+suffix,IsPaid:true,RequiresApproval:true,IsActive:true})
	if err!=nil{t.Fatalf("create leave type: %v",err)}
	return &staffLeaveIntegrationFixture{pool:pool,repo:repo,ctx:ctx,schoolID:schoolID,adminPrincipalID:adminPrincipalID,staffPrincipalID:staffPrincipalID,leaveTypeID:leaveType.ID}
}

func (f *staffLeaveIntegrationFixture) request(t *testing.T,staffID int64,start,end string) *StaffLeaveRequest {
	t.Helper()
	item,err:=f.repo.CreateRequest(f.ctx,f.adminPrincipalID,f.schoolID,StaffLeaveRequest{StaffPrincipalID:staffID,LeaveTypeID:f.leaveTypeID,StartDate:start,EndDate:end,DurationType:"full_day",Reason:"Integration test leave"})
	if err!=nil{t.Fatalf("create leave request: %v",err)}
	return item
}

func TestStaffLeaveWorkflowPostgresIntegration(t *testing.T) {
	f:=newStaffLeaveIntegrationFixture(t)

	t.Run("concurrent overlapping requests serialize and only one succeeds",func(t *testing.T){
		start:=time.Now().AddDate(0,0,30).Format("2006-01-02")
		end:=time.Now().AddDate(0,0,31).Format("2006-01-02")
		type result struct{err error}
		results:=make(chan result,2)
		startGate:=make(chan struct{})
		// Run two independent pool transactions at the same instant.
		for i:=0;i<2;i++{
			go func(){
				<-startGate
				_,err:=f.repo.CreateRequest(f.ctx,f.adminPrincipalID,f.schoolID,StaffLeaveRequest{StaffPrincipalID:f.staffPrincipalID,LeaveTypeID:f.leaveTypeID,StartDate:start,EndDate:end,DurationType:"full_day",Reason:"Concurrent overlap"})
				results<-result{err:err}
			}()
		}
		close(startGate)
		var succeeded,conflicted int
		for i:=0;i<2;i++{
			got:=<-results
			switch{
			case got.err==nil:succeeded++
			case errors.Is(got.err,ErrStaffLeaveConflict):conflicted++
			default:t.Fatalf("unexpected concurrent leave result: %v",got.err)
			}
		}
		if succeeded!=1||conflicted!=1{t.Fatalf("concurrent outcomes success=%d conflict=%d; want 1 each",succeeded,conflicted)}
	})

	t.Run("decision permission self approval and audit",func(t *testing.T){
		request:=f.request(t,f.staffPrincipalID,time.Now().AddDate(0,0,40).Format("2006-01-02"),time.Now().AddDate(0,0,40).Format("2006-01-02"))
		if err:=f.repo.Decide(f.ctx,f.staffPrincipalID,f.schoolID,request.ID,true,nil);!errors.Is(err,ErrStaffLeaveAccessDenied){
			t.Fatalf("unauthorized staff decision error=%v, want access denied",err)
		}
		if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,true,nil);err!=nil{t.Fatalf("School Admin approval failed: %v",err)}
		var status string
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1 AND school_id=$2`,request.ID,f.schoolID).Scan(&status);err!=nil{t.Fatalf("read approved leave status: %v",err)}
		if status!="approved"{t.Fatalf("leave status=%q want approved",status)}
		var auditCount int
		if err:=f.pool.QueryRow(f.ctx,`SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='staff.leave.approved' AND resource_id=$3`,f.schoolID,f.adminPrincipalID,fmt.Sprint(request.ID)).Scan(&auditCount);err!=nil{t.Fatalf("count approval audit: %v",err)}
		if auditCount!=1{t.Fatalf("approval audit count=%d want=1",auditCount)}
	})

	t.Run("self approval is denied",func(t *testing.T){
		request:=f.request(t,f.adminPrincipalID,time.Now().AddDate(0,0,45).Format("2006-01-02"),time.Now().AddDate(0,0,45).Format("2006-01-02"))
		if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,true,nil);!errors.Is(err,ErrStaffLeaveAccessDenied){
			t.Fatalf("self-approval error=%v, want access denied",err)
		}
		var status string
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1`,request.ID).Scan(&status);err!=nil{t.Fatalf("read self-approval status: %v",err)}
		if status!="pending"{t.Fatalf("self-approved request status=%q want pending",status)}
	})

	t.Run("audit failure rolls back leave decision",func(t *testing.T){
		request:=f.request(t,f.staffPrincipalID,time.Now().AddDate(0,0,50).Format("2006-01-02"),time.Now().AddDate(0,0,50).Format("2006-01-02"))
		const triggerName="trg_test_staff_leave_reject_audit_failure"
		const functionName="test_staff_leave_reject_audit_failure"
		if _,err:=f.pool.Exec(f.ctx,`CREATE OR REPLACE FUNCTION public.test_staff_leave_reject_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='staff.leave.rejected' THEN RAISE EXCEPTION 'forced audit failure'; END IF; RETURN NEW; END $$`);err!=nil{t.Fatalf("create audit failure function: %v",err)}
		if _,err:=f.pool.Exec(f.ctx,`CREATE TRIGGER trg_test_staff_leave_reject_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_staff_leave_reject_audit_failure()`);err!=nil{t.Fatalf("create audit failure trigger: %v",err)}
		t.Cleanup(func(){_,_=f.pool.Exec(context.Background(),`DROP TRIGGER IF EXISTS trg_test_staff_leave_reject_audit_failure ON public.audit_events`);_,_=f.pool.Exec(context.Background(),`DROP FUNCTION IF EXISTS public.test_staff_leave_reject_audit_failure()`)})
		comment:="Rejected in integration test"
		if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,false,&comment);err==nil{t.Fatal("decision unexpectedly succeeded when audit insert failed")}
		var status string
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1`,request.ID).Scan(&status);err!=nil{t.Fatalf("read rolled-back request: %v",err)}
		if status!="pending"{t.Fatalf("request status=%q after audit failure; want pending",status)}
		_,_=f.pool.Exec(f.ctx,fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON public.audit_events`,triggerName))
		_,_=f.pool.Exec(f.ctx,fmt.Sprintf(`DROP FUNCTION IF EXISTS public.%s()`,functionName))
	})

	t.Run("school scope blocks cross-school cancellation",func(t *testing.T){
		request:=f.request(t,f.staffPrincipalID,time.Now().AddDate(0,0,55).Format("2006-01-02"),time.Now().AddDate(0,0,55).Format("2006-01-02"))
		if err:=f.repo.Cancel(f.ctx,f.adminPrincipalID,f.schoolID+999999,request.ID);!errors.Is(err,ErrStaffLeaveAccessDenied){
			t.Fatalf("cross-school cancel error=%v, want access denied",err)
		}
		var status string
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1 AND school_id=$2`,request.ID,f.schoolID).Scan(&status);err!=nil{t.Fatalf("read cross-school request: %v",err)}
		if status!="pending"{t.Fatalf("cross-school operation changed request status to %q",status)}
	})
}

func TestStaffLeaveDecisionRejectRequiresReasonAndPersistsAuditPostgresIntegration(t *testing.T) {
	f:=newStaffLeaveIntegrationFixture(t)
	request:=f.request(t,f.staffPrincipalID,time.Now().AddDate(0,0,60).Format("2006-01-02"),time.Now().AddDate(0,0,60).Format("2006-01-02"))
	if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,false,nil);!errors.Is(err,ErrInvalidStaffLeave){
		t.Fatalf("rejection without comment error=%v, want invalid leave",err)
	}
	comment:="Insufficient staffing"
	if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,false,&comment);err!=nil{t.Fatalf("reject leave with reason: %v",err)}
	var status,decisionComment string
	if err:=f.pool.QueryRow(f.ctx,`SELECT status,decision_comment FROM staff_leave_requests WHERE id=$1`,request.ID).Scan(&status,&decisionComment);err!=nil{t.Fatalf("read rejected leave: %v",err)}
	if status!="rejected"||decisionComment!=comment{t.Fatalf("rejected request status=%q comment=%q",status,decisionComment)}
	var auditCount int
	if err:=f.pool.QueryRow(f.ctx,`SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='staff.leave.rejected' AND resource_id=$3`,f.schoolID,f.adminPrincipalID,fmt.Sprint(request.ID)).Scan(&auditCount);err!=nil{t.Fatalf("count rejection audit: %v",err)}
	if auditCount!=1{t.Fatalf("rejection audit count=%d want=1",auditCount)}
	if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,request.ID,true,nil);!errors.Is(err,ErrStaffLeaveNotFound){
		t.Fatalf("second decision error=%v, want request not found",err)
	}
}


func TestStaffLeaveReportDateRangePostgresIntegration(t *testing.T) {
 f:=newStaffLeaveIntegrationFixture(t)
 inside:=f.request(t,f.staffPrincipalID,"2027-03-10","2027-03-11")
 outside:=f.request(t,f.staffPrincipalID,"2027-04-10","2027-04-10")
 if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,inside.ID,true,nil);err!=nil{t.Fatalf("approve in-range leave: %v",err)}
 if err:=f.repo.Decide(f.ctx,f.adminPrincipalID,f.schoolID,outside.ID,true,nil);err!=nil{t.Fatalf("approve out-of-range leave: %v",err)}
 got,err:=f.repo.ListRequests(f.ctx,f.adminPrincipalID,f.schoolID,"approved",0,"","2027-03-01","2027-03-31")
 if err!=nil{t.Fatalf("list approved leave overlapping March range: %v",err)}
 if len(got)!=1||got[0].ID!=inside.ID{t.Fatalf("range returned %+v; want only request %d",got,inside.ID)}
}

func TestSchoolHolidayInactiveListingRequiresManagePermissionPostgresIntegration(t *testing.T) {
	f := newStaffLeaveIntegrationFixture(t)
	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-01-10", Name: "Active integration holiday", IsActive: true,
	}); err != nil {
		t.Fatalf("create active holiday: %v", err)
	}
	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-01-11", Name: "Inactive integration holiday", IsActive: true,
	}); err != nil {
		t.Fatalf("create holiday to deactivate: %v", err)
	}
	var inactiveID int64
	if err := f.pool.QueryRow(f.ctx, `SELECT id FROM school_holidays WHERE school_id=$1 AND name='Inactive integration holiday'`, f.schoolID).Scan(&inactiveID); err != nil {
		t.Fatalf("load holiday to deactivate: %v", err)
	}
	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, inactiveID, SchoolHoliday{
		HolidayDate: "2027-01-11", Name: "Inactive integration holiday", IsActive: false,
	}); err != nil {
		t.Fatalf("deactivate holiday: %v", err)
	}

	activeOnly, err := f.repo.ListHolidays(f.ctx, f.adminPrincipalID, f.schoolID, "2027-01-01", "2027-01-31", false)
	if err != nil {
		t.Fatalf("list active holidays: %v", err)
	}
	if len(activeOnly) != 1 || !activeOnly[0].IsActive {
		t.Fatalf("active-only list = %+v; want only the active holiday", activeOnly)
	}

	all, err := f.repo.ListHolidays(f.ctx, f.adminPrincipalID, f.schoolID, "2027-01-01", "2027-01-31", true)
	if err != nil {
		t.Fatalf("list all holidays as manager: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("manager holiday list count=%d; want active and inactive entries", len(all))
	}
	var foundInactive bool
	for _, item := range all {
		if item.Name == "Inactive integration holiday" && !item.IsActive {
			foundInactive = true
		}
	}
	if !foundInactive {
		t.Fatalf("manager holiday list did not include inactive entry: %+v", all)
	}

	if _, err := f.repo.ListHolidays(f.ctx, f.staffPrincipalID, f.schoolID, "2027-01-01", "2027-01-31", true); !errors.Is(err, ErrStaffLeaveAccessDenied) {
		t.Fatalf("non-manager inactive holiday listing error=%v; want access denied", err)
	}
}
