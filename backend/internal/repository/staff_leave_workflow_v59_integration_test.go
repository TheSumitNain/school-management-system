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
	 t.Run("cancellation audit failure rolls back and successful cancellation is audited once",func(t *testing.T){
		request:=f.request(t,f.staffPrincipalID,time.Now().AddDate(0,0,56).Format("2006-01-02"),time.Now().AddDate(0,0,56).Format("2006-01-02"))
		const triggerName="trg_test_staff_leave_cancel_audit_failure"
		const functionName="test_staff_leave_cancel_audit_failure"
		if _,err:=f.pool.Exec(f.ctx,`CREATE OR REPLACE FUNCTION public.test_staff_leave_cancel_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $ BEGIN IF NEW.action='staff.leave.cancelled' THEN RAISE EXCEPTION 'forced cancellation audit failure'; END IF; RETURN NEW; END $`);err!=nil{t.Fatalf("create cancellation audit failure function: %v",err)}
		if _,err:=f.pool.Exec(f.ctx,`CREATE TRIGGER trg_test_staff_leave_cancel_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_staff_leave_cancel_audit_failure()`);err!=nil{t.Fatalf("create cancellation audit failure trigger: %v",err)}
		t.Cleanup(func(){_,_=f.pool.Exec(context.Background(),`DROP TRIGGER IF EXISTS trg_test_staff_leave_cancel_audit_failure ON public.audit_events`);_,_=f.pool.Exec(context.Background(),`DROP FUNCTION IF EXISTS public.test_staff_leave_cancel_audit_failure()`)})
		if err:=f.repo.Cancel(f.ctx,f.adminPrincipalID,f.schoolID,request.ID);err==nil{t.Fatal("cancellation unexpectedly succeeded when audit insert failed")}
		var status string
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1 AND school_id=$2`,request.ID,f.schoolID).Scan(&status);err!=nil{t.Fatalf("read rolled-back cancellation: %v",err)}
		if status!="pending"{t.Fatalf("request status=%q after cancellation audit failure; want pending",status)}
		if _,err:=f.pool.Exec(f.ctx,`DROP TRIGGER IF EXISTS trg_test_staff_leave_cancel_audit_failure ON public.audit_events`);err!=nil{t.Fatalf("drop cancellation audit failure trigger: %v",err)}
		if _,err:=f.pool.Exec(f.ctx,`DROP FUNCTION IF EXISTS public.test_staff_leave_cancel_audit_failure()`);err!=nil{t.Fatalf("drop cancellation audit failure function: %v",err)}
		if err:=f.repo.Cancel(f.ctx,f.adminPrincipalID,f.schoolID,request.ID);err!=nil{t.Fatalf("cancel leave request after removing forced failure: %v",err)}
		if err:=f.pool.QueryRow(f.ctx,`SELECT status FROM staff_leave_requests WHERE id=$1 AND school_id=$2`,request.ID,f.schoolID).Scan(&status);err!=nil{t.Fatalf("read cancelled request: %v",err)}
		if status!="cancelled"{t.Fatalf("request status=%q after cancellation; want cancelled",status)}
		var auditCount int
		if err:=f.pool.QueryRow(f.ctx,`SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='staff.leave.cancelled' AND resource_id=$3`,f.schoolID,f.adminPrincipalID,fmt.Sprint(request.ID)).Scan(&auditCount);err!=nil{t.Fatalf("count cancellation audit: %v",err)}
		if auditCount!=1{t.Fatalf("cancellation audit count=%d want=1",auditCount)}
		if err:=f.repo.Cancel(f.ctx,f.adminPrincipalID,f.schoolID,request.ID);!errors.Is(err,ErrStaffLeaveNotFound){t.Fatalf("second cancellation error=%v, want not found",err)}
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

func TestSchoolHolidayDuplicateAndConflictingUpdatePostgresIntegration(t *testing.T) {
	f := newStaffLeaveIntegrationFixture(t)
	first, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-01-26", Name: "Republic Day", IsActive: true,
	})
	if err != nil {
		t.Fatalf("create first holiday: %v", err)
	}

	// The same school/date/name is a duplicate even when the name casing or
	// surrounding whitespace differs.
	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-01-26", Name: "  republic day  ", IsActive: true,
	}); !errors.Is(err, ErrStaffLeaveConflict) {
		t.Fatalf("case-insensitive duplicate holiday error=%v; want conflict", err)
	}

	second, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-01-27", Name: "School Foundation Day", IsActive: true,
	})
	if err != nil {
		t.Fatalf("create second holiday: %v", err)
	}

	// Updating another row into the first row's unique key must fail without
	// changing the existing row or writing a success audit event.
	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, second.ID, SchoolHoliday{
		HolidayDate: "2027-01-26", Name: "REPUBLIC DAY", IsActive: true,
	}); !errors.Is(err, ErrStaffLeaveConflict) {
		t.Fatalf("conflicting holiday update error=%v; want conflict", err)
	}

	var date, name string
	if err := f.pool.QueryRow(f.ctx, `SELECT holiday_date::text,name FROM school_holidays WHERE id=$1 AND school_id=$2`, second.ID, f.schoolID).Scan(&date, &name); err != nil {
		t.Fatalf("read second holiday after conflicting update: %v", err)
	}
	if date != "2027-01-27" || name != "School Foundation Day" {
		t.Fatalf("conflicting update changed second holiday to %s %q", date, name)
	}

	var firstAuditCount, secondAuditCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.holiday.created' AND resource_id=$3`, f.schoolID, f.adminPrincipalID, fmt.Sprint(first.ID)).Scan(&firstAuditCount); err != nil {
		t.Fatalf("count first holiday audit: %v", err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND resource_type='school_holiday' AND resource_id=$3`, f.schoolID, f.adminPrincipalID, fmt.Sprint(second.ID)).Scan(&secondAuditCount); err != nil {
		t.Fatalf("count second holiday audit: %v", err)
	}
	if firstAuditCount != 1 || secondAuditCount != 1 {
		t.Fatalf("holiday audit counts first=%d second=%d; want exactly one successful create audit per row", firstAuditCount, secondAuditCount)
	}
}

func TestSchoolHolidayCrossSchoolWriteAndAuditRollbackPostgresIntegration(t *testing.T) {
	f := newStaffLeaveIntegrationFixture(t)

	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID+999999, 0, SchoolHoliday{
		HolidayDate: "2027-08-15", Name: "Cross-school holiday", IsActive: true,
	}); !errors.Is(err, ErrStaffLeaveAccessDenied) {
		t.Fatalf("cross-school holiday create error=%v; want access denied", err)
	}

	const functionName = "test_school_holiday_audit_failure"
	const triggerName = "trg_test_school_holiday_audit_failure"
	if _, err := f.pool.Exec(f.ctx, `CREATE OR REPLACE FUNCTION public.test_school_holiday_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='school.holiday.created' THEN RAISE EXCEPTION 'forced holiday audit failure'; END IF; RETURN NEW; END $$`); err != nil {
		t.Fatalf("create holiday audit failure function: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `CREATE TRIGGER trg_test_school_holiday_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_school_holiday_audit_failure()`); err != nil {
		t.Fatalf("create holiday audit failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS trg_test_school_holiday_audit_failure ON public.audit_events`)
		_, _ = f.pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS public.test_school_holiday_audit_failure()`)
	})

	if _, err := f.repo.SaveHoliday(f.ctx, f.adminPrincipalID, f.schoolID, 0, SchoolHoliday{
		HolidayDate: "2027-08-15", Name: "Audit rollback holiday", IsActive: true,
	}); err == nil {
		t.Fatal("holiday create unexpectedly succeeded when audit insert failed")
	}

	var holidayCount, auditCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM school_holidays WHERE school_id=$1 AND name='Audit rollback holiday'`, f.schoolID).Scan(&holidayCount); err != nil {
		t.Fatalf("count holiday rows after audit failure: %v", err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.holiday.created' AND metadata->>'holiday_date'='2027-08-15'`, f.schoolID, f.adminPrincipalID).Scan(&auditCount); err != nil {
		t.Fatalf("count holiday audit rows after audit failure: %v", err)
	}
	if holidayCount != 0 || auditCount != 0 {
		t.Fatalf("audit failure left partial state: holiday rows=%d audit rows=%d; want both zero", holidayCount, auditCount)
	}
}
