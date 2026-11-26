package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"
	"school-management-system/internal/schooltime"

	"github.com/jackc/pgx/v5"
)

func TestStaffRecordsRepositoryPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin rollback-scoped staff records test: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var actorID, schoolID int64
	if err := tx.QueryRow(ctx, `SELECT principal_id,school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorID,&schoolID); err != nil {
		t.Fatalf("load School Admin fixture: %v",err)
	}

	var platformAdminID int64
	err = tx.QueryRow(ctx,`
		SELECT ap.id
		FROM platform_admin_accounts pa
		JOIN auth_principals ap ON ap.id=pa.principal_id
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		JOIN role_permissions rp ON rp.role_id=r.id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE ap.principal_type='platform_admin' AND ap.status='active'
		  AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
		  AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
		LIMIT 1
	`).Scan(&platformAdminID)
	if errors.Is(err,pgx.ErrNoRows) {
		var anyBootstrap bool
		if checkErr:=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)`).Scan(&anyBootstrap);checkErr!=nil{t.Fatalf("check Platform Admin bootstrap: %v",checkErr)}
		if anyBootstrap { t.Skip("Platform Admin exists but no active administrator can delegate permissions") }
		platformAdminID,err=NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx,tx,fmt.Sprintf("staff-records-%d@example.invalid",time.Now().UnixNano()),"integration-test-password-hash")
	}
	if err!=nil { t.Fatalf("resolve Platform Admin fixture: %v",err) }

	var schoolAdminRoleID int64
	if err:=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff' AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`,schoolID).Scan(&schoolAdminRoleID);err!=nil{
		t.Fatalf("load School Admin role: %v",err)
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,actorID,schoolAdminRoleID,platformAdminID);err!=nil{
		t.Fatalf("assign School Admin fixture role: %v",err)
	}

	repo:=NewUserRepository(tx)
	suffix:=fmt.Sprintf("%d",time.Now().UnixNano())
	staff,err:=repo.CreateStaffAccountWithProfile(ctx,actorID,schoolID,"staffrecord-"+suffix,"integration-test-password-hash",models.StaffProfile{
		EmployeeCode:"SR-"+suffix,FullName:"Staff Record Integration",Designation:"Teacher",MobileNumber:"9876543210",EmploymentStatus:"active",
	})
	if err!=nil{t.Fatalf("create staff profile fixture: %v",err)}

	year:=2021
	score:=86.25
	qualification,err:=repo.CreateStaffQualification(ctx,actorID,schoolID,staff.PrincipalID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"M.Sc. Mathematics",Institution:"State University",CompletionYear:&year,Score:&score,ScoreScale:"percentage",VerificationStatus:"verified",
	})
	if err!=nil{t.Fatalf("create staff qualification: %v",err)}
	if qualification.ID<=0||qualification.VerificationStatus!="pending"{t.Fatalf("unexpected qualification response: %+v",qualification)}

	certification,err:=repo.CreateStaffCertification(ctx,actorID,schoolID,staff.PrincipalID,StaffCertification{
		CertificateName:"Teacher Training",IssuingOrganization:"State Board",IssueDate:"2022-01-10",ExpiryDate:"2027-01-10",VerificationStatus:"verified",
	})
	if err!=nil{t.Fatalf("create staff certification: %v",err)}
	if certification.ID<=0||certification.VerificationStatus!="pending"{t.Fatalf("unexpected certification response: %+v",certification)}

	experience,err:=repo.CreateStaffExperience(ctx,actorID,schoolID,staff.PrincipalID,StaffExperience{
		EmployerName:"Previous School",Designation:"Mathematics Teacher",EmploymentType:"permanent",StartDate:"2018-04-01",EndDate:"2021-03-31",
	})
	if err!=nil{t.Fatalf("create staff experience: %v",err)}
	if experience.ID<=0{t.Fatalf("unexpected experience response: %+v",experience)}

	updatedQualification,err:=repo.UpdateStaffQualification(ctx,actorID,schoolID,staff.PrincipalID,qualification.ID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"M.Sc. Applied Mathematics",Institution:"State University",CompletionYear:&year,Score:&score,ScoreScale:"percentage",VerificationStatus:"verified",
	})
	if err!=nil||updatedQualification.QualificationName!="M.Sc. Applied Mathematics"||updatedQualification.VerificationStatus!="pending"{t.Fatalf("update qualification must preserve pending verification despite forged input: item=%+v err=%v",updatedQualification,err)}
	updatedCertification,err:=repo.UpdateStaffCertification(ctx,actorID,schoolID,staff.PrincipalID,certification.ID,StaffCertification{
		CertificateName:"Advanced Teacher Training",IssuingOrganization:"State Board",IssueDate:"2022-01-10",ExpiryDate:"2028-01-10",VerificationStatus:"verified",
	})
	if err!=nil||updatedCertification.CertificateName!="Advanced Teacher Training"||updatedCertification.VerificationStatus!="pending"{t.Fatalf("update certification must preserve pending verification despite forged input: item=%+v err=%v",updatedCertification,err)}
	updatedExperience,err:=repo.UpdateStaffExperience(ctx,actorID,schoolID,staff.PrincipalID,experience.ID,StaffExperience{
		EmployerName:"Previous School",Designation:"Senior Mathematics Teacher",EmploymentType:"permanent",StartDate:"2018-04-01",EndDate:"2021-03-31",
	})

    // Force audit inserts to fail and verify all six record write paths roll back.
    auditFailureFunction := "test_staff_record_audit_failure_" + suffix
    auditFailureTrigger := "test_staff_record_audit_failure_trigger_" + suffix
    if _, err := tx.Exec(ctx, fmt.Sprintf("CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $staff_record_audit_failure$ BEGIN IF NEW.action IN ('staff.qualification.created','staff.qualification.updated','staff.certification.created','staff.certification.updated','staff.experience.created','staff.experience.updated') THEN RAISE EXCEPTION 'injected staff record audit failure'; END IF; RETURN NEW; END; $staff_record_audit_failure$; CREATE TRIGGER %s BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION %s();", auditFailureFunction, auditFailureTrigger, auditFailureFunction)); err != nil {
        t.Fatalf("install staff record audit failure trigger: %v", err)
    }
    if item, err := repo.CreateStaffQualification(ctx, actorID, schoolID, staff.PrincipalID, StaffQualification{QualificationType:"diploma",QualificationName:"Must Roll Back",Institution:"Failure Test"}); err == nil || item != nil {
        t.Fatalf("qualification create with audit failure returned item=%+v err=%v", item, err)
    }
    if item, err := repo.CreateStaffCertification(ctx, actorID, schoolID, staff.PrincipalID, StaffCertification{CertificateName:"Must Roll Back",IssuingOrganization:"Failure Test",IssueDate:"2023-01-01"}); err == nil || item != nil {
        t.Fatalf("certification create with audit failure returned item=%+v err=%v", item, err)
    }
    if item, err := repo.CreateStaffExperience(ctx, actorID, schoolID, staff.PrincipalID, StaffExperience{EmployerName:"Must Roll Back",Designation:"Failure Test",EmploymentType:"permanent",StartDate:"2017-01-01"}); err == nil || item != nil {
        t.Fatalf("experience create with audit failure returned item=%+v err=%v", item, err)
    }
    if _, err := repo.UpdateStaffQualification(ctx, actorID, schoolID, staff.PrincipalID, qualification.ID, StaffQualification{QualificationType:"postgraduate",QualificationName:"Must Roll Back",Institution:"Failure Test"}); err == nil {
        t.Fatal("qualification update unexpectedly succeeded when audit insert failed")
    }
    if _, err := repo.UpdateStaffCertification(ctx, actorID, schoolID, staff.PrincipalID, certification.ID, StaffCertification{CertificateName:"Must Roll Back",IssuingOrganization:"Failure Test",IssueDate:"2022-01-10"}); err == nil {
        t.Fatal("certification update unexpectedly succeeded when audit insert failed")
    }
    if _, err := repo.UpdateStaffExperience(ctx, actorID, schoolID, staff.PrincipalID, experience.ID, StaffExperience{EmployerName:"Must Roll Back",Designation:"Must Roll Back",EmploymentType:"permanent",StartDate:"2018-04-01"}); err == nil {
        t.Fatal("experience update unexpectedly succeeded when audit insert failed")
    }
    var qualificationCount, certificationCount, experienceCount int
    if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM staff_qualifications WHERE school_id=$1 AND staff_principal_id=$2", schoolID, staff.PrincipalID).Scan(&qualificationCount); err != nil { t.Fatalf("count qualifications after audit failures: %v", err) }
    if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM staff_certifications WHERE school_id=$1 AND staff_principal_id=$2", schoolID, staff.PrincipalID).Scan(&certificationCount); err != nil { t.Fatalf("count certifications after audit failures: %v", err) }
    if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM staff_experience WHERE school_id=$1 AND staff_principal_id=$2", schoolID, staff.PrincipalID).Scan(&experienceCount); err != nil { t.Fatalf("count experience after audit failures: %v", err) }
    if qualificationCount != 1 || certificationCount != 1 || experienceCount != 1 {
        t.Fatalf("failed creates left partial records: qualifications=%d certifications=%d experience=%d", qualificationCount, certificationCount, experienceCount)
    }
    var qualificationName, certificationName, experienceDesignation string
    if err := tx.QueryRow(ctx, "SELECT qualification_name FROM staff_qualifications WHERE id=$1", qualification.ID).Scan(&qualificationName); err != nil { t.Fatalf("read qualification after audit failure: %v", err) }
    if err := tx.QueryRow(ctx, "SELECT certificate_name FROM staff_certifications WHERE id=$1", certification.ID).Scan(&certificationName); err != nil { t.Fatalf("read certification after audit failure: %v", err) }
    if err := tx.QueryRow(ctx, "SELECT designation FROM staff_experience WHERE id=$1", experience.ID).Scan(&experienceDesignation); err != nil { t.Fatalf("read experience after audit failure: %v", err) }
    if qualificationName != "M.Sc. Applied Mathematics" || certificationName != "Advanced Teacher Training" || experienceDesignation != "Senior Mathematics Teacher" {
        t.Fatalf("audit failure did not roll back updates: qualification=%q certification=%q experience=%q", qualificationName, certificationName, experienceDesignation)
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", auditFailureTrigger, auditFailureFunction)); err != nil {
        t.Fatalf("remove staff record audit failure trigger: %v", err)
    }

    attendanceRepo := NewStaffAttendanceRepository(tx)
    today := schooltime.Today()
    marked, err := attendanceRepo.Mark(ctx, actorID, schoolID, staff.PrincipalID, today, "present", nil)
    if err != nil { t.Fatalf("mark staff attendance: %v", err) }
    if marked.ID == nil || marked.Status == nil || *marked.Status != "present" { t.Fatalf("unexpected staff attendance result: %+v", marked) }
    if _, err := attendanceRepo.Mark(ctx, actorID, schoolID, staff.PrincipalID, today, "absent", nil); !errors.Is(err, ErrStaffAttendanceConflict) {
        t.Fatalf("duplicate same-day staff attendance error=%v want conflict", err)
    }
    daily, err := attendanceRepo.ListDaily(ctx, actorID, schoolID, today)
    if err != nil || len(daily) != 1 || daily[0].Status == nil || *daily[0].Status != "present" {
        t.Fatalf("daily staff attendance: records=%+v err=%v", daily, err)
    }
    updatedAttendance, err := attendanceRepo.Update(ctx, actorID, schoolID, *marked.ID, "late", nil)
    if err != nil || updatedAttendance.Status == nil || *updatedAttendance.Status != "late" {
        t.Fatalf("update same-day staff attendance: record=%+v err=%v", updatedAttendance, err)
    }
    // An audit failure during an attendance correction must preserve the prior status.
    var updateAuditCountBefore int
    if err:=tx.QueryRow(ctx,`SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND action='school.staff_attendance.updated' AND resource_id=$2`,schoolID,fmt.Sprint(*marked.ID)).Scan(&updateAuditCountBefore);err!=nil{t.Fatalf("count attendance update audit events before failure: %v",err)}
    if _,err:=tx.Exec(ctx,`CREATE OR REPLACE FUNCTION public.test_staff_attendance_update_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $body$ BEGIN IF NEW.action='school.staff_attendance.updated' THEN RAISE EXCEPTION 'forced attendance update audit failure'; END IF; RETURN NEW; END; $body$`);err!=nil{t.Fatalf("create attendance update audit failure function: %v",err)}
    if _,err:=tx.Exec(ctx,`CREATE TRIGGER trg_test_staff_attendance_update_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_staff_attendance_update_audit_failure()`);err!=nil{t.Fatalf("create attendance update audit failure trigger: %v",err)}
    if _,err:=attendanceRepo.Update(ctx,actorID,schoolID,*marked.ID,"absent",nil);err==nil{t.Fatal("attendance update succeeded despite forced audit failure")}
    var statusAfterAuditFailure string
    if err:=tx.QueryRow(ctx,`SELECT status FROM staff_attendance_records WHERE id=$1 AND school_id=$2`,*marked.ID,schoolID).Scan(&statusAfterAuditFailure);err!=nil{t.Fatalf("read attendance after update audit failure: %v",err)}
    if statusAfterAuditFailure!="late"{t.Fatalf("attendance status after audit failure=%q want prior status late",statusAfterAuditFailure)}
    var updateAuditCount int
    if err:=tx.QueryRow(ctx,`SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND action='school.staff_attendance.updated' AND resource_id=$2`,schoolID,fmt.Sprint(*marked.ID)).Scan(&updateAuditCount);err!=nil{t.Fatalf("count attendance update audit events: %v",err)}
    if updateAuditCount!=updateAuditCountBefore{t.Fatalf("attendance update audit rows changed after failed update: before=%d after=%d",updateAuditCountBefore,updateAuditCount)}
    if _,err:=tx.Exec(ctx,`DROP TRIGGER trg_test_staff_attendance_update_audit_failure ON public.audit_events`);err!=nil{t.Fatalf("drop attendance update audit failure trigger: %v",err)}
    if _,err:=tx.Exec(ctx,`DROP FUNCTION public.test_staff_attendance_update_audit_failure()`);err!=nil{t.Fatalf("drop attendance update audit failure function: %v",err)}
    yesterday := today.AddDate(0, 0, -1)
    past, err := attendanceRepo.Mark(ctx, actorID, schoolID, staff.PrincipalID, yesterday, "absent", nil)
    if err != nil { t.Fatalf("mark historical staff attendance with correction permission: %v", err) }
    if _, err := attendanceRepo.Mark(ctx, actorID, schoolID, staff.PrincipalID, today.AddDate(0, 0, 1), "present", nil); !errors.Is(err, ErrInvalidStaffAttendance) {
        t.Fatalf("future staff attendance error=%v want invalid input", err)
    }
    if _, err := tx.Exec(ctx, "SAVEPOINT staff_attendance_future_date"); err != nil { t.Fatalf("create future-date savepoint: %v", err) }
    _, databaseFutureErr := tx.Exec(ctx, `
        INSERT INTO staff_attendance_records
            (school_id, staff_principal_id, attendance_date, status, marked_by_principal_id, updated_by_principal_id)
        VALUES ($1,$2,$3,'present',$4,$4)
    `, schoolID, staff.PrincipalID, today.AddDate(0, 0, 1).Format("2006-01-02"), actorID)
    if databaseFutureErr == nil { t.Fatal("database accepted a future staff attendance date") }
    if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT staff_attendance_future_date"); err != nil { t.Fatalf("rollback future-date savepoint: %v", err) }
    if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT staff_attendance_future_date"); err != nil { t.Fatalf("release future-date savepoint: %v", err) }
    monthRows, err := attendanceRepo.ListMonthly(ctx, actorID, schoolID, today)
    if err != nil || len(monthRows) != 1 || monthRows[0].LateDays != 1 || monthRows[0].AbsentDays != 1 || len(monthRows[0].Records) == 0 {
        t.Fatalf("monthly staff attendance: records=%+v err=%v", monthRows, err)
    }
    if _, err := attendanceRepo.ListDaily(ctx, actorID, schoolID+1, today); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
        t.Fatalf("cross-school daily attendance error=%v want access denied", err)
    }
    if _, err := attendanceRepo.Update(ctx, actorID, schoolID, *past.ID, "present", nil); err != nil {
        t.Fatalf("correct historical staff attendance: %v", err)
    }
    secondStaff, err := repo.CreateStaffAccountWithProfile(ctx, actorID, schoolID, "staffbulk-"+suffix, "integration-test-password-hash", models.StaffProfile{
        EmployeeCode:"SB-"+suffix, FullName:"Bulk Staff Integration", Designation:"Teacher", MobileNumber:"9876543211", EmploymentStatus:"active",
    })
    if err != nil { t.Fatalf("create second staff for bulk attendance: %v", err) }

    // A record ID from another employee in the same school must not bypass
    // the staff principal ID supplied by the profile route.
    otherQualifications, err := repo.ListStaffQualifications(ctx, actorID, schoolID, secondStaff.PrincipalID)
    if err != nil || len(otherQualifications) != 0 {
        t.Fatalf("second employee qualifications leaked: items=%+v err=%v", otherQualifications, err)
    }
    otherCertifications, err := repo.ListStaffCertifications(ctx, actorID, schoolID, secondStaff.PrincipalID)
    if err != nil || len(otherCertifications) != 0 {
        t.Fatalf("second employee certifications leaked: items=%+v err=%v", otherCertifications, err)
    }
    otherExperiences, err := repo.ListStaffExperience(ctx, actorID, schoolID, secondStaff.PrincipalID)
    if err != nil || len(otherExperiences) != 0 {
        t.Fatalf("second employee experience leaked: items=%+v err=%v", otherExperiences, err)
    }
    if _, err := repo.UpdateStaffQualification(ctx, actorID, schoolID, secondStaff.PrincipalID, qualification.ID, StaffQualification{
        QualificationType: "postgraduate", QualificationName: "Unauthorized Update", Institution: "Other University", CompletionYear: &year, Score: &score, ScoreScale: "percentage",
    }); !errors.Is(err, ErrInvalidStaffRecord) {
        t.Fatalf("cross-employee qualification update error=%v want ErrInvalidStaffRecord", err)
    }
    if _, err := repo.UpdateStaffCertification(ctx, actorID, schoolID, secondStaff.PrincipalID, certification.ID, StaffCertification{
        CertificateName: "Unauthorized Update", IssuingOrganization: "Other Board",
    }); !errors.Is(err, ErrInvalidStaffRecord) {
        t.Fatalf("cross-employee certification update error=%v want ErrInvalidStaffRecord", err)
    }
    if _, err := repo.UpdateStaffExperience(ctx, actorID, schoolID, secondStaff.PrincipalID, experience.ID, StaffExperience{
        EmployerName: "Other School", Designation: "Unauthorized Update", EmploymentType: "permanent", StartDate: "2018-04-01",
    }); !errors.Is(err, ErrInvalidStaffRecord) {
        t.Fatalf("cross-employee experience update error=%v want ErrInvalidStaffRecord", err)
    }


    if _, err := attendanceRepo.MarkBulk(ctx, actorID, schoolID, []int64{secondStaff.PrincipalID, staff.PrincipalID}, today, "absent", nil); !errors.Is(err, ErrStaffAttendanceConflict) {
        t.Fatalf("bulk attendance with an existing daily mark error=%v want conflict", err)
    }
    var partialBulkCreated bool
    if err := tx.QueryRow(ctx, `
        SELECT EXISTS(SELECT 1 FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=$2 AND attendance_date=$3::date)
    `, schoolID, secondStaff.PrincipalID, today.Format("2006-01-02")).Scan(&partialBulkCreated); err != nil {
        t.Fatalf("check atomic bulk rollback: %v", err)
    }
    if partialBulkCreated { t.Fatal("bulk attendance left a partial record after duplicate conflict") }
    bulkCount, err := attendanceRepo.MarkBulk(ctx, actorID, schoolID, []int64{secondStaff.PrincipalID}, today, "present", nil)
    if err != nil || bulkCount != 1 { t.Fatalf("bulk attendance success count=%d err=%v", bulkCount, err) }

    // A failed audit insert must roll back the attendance row in the same
    // transaction, for both individual and bulk writes.
        singleFailureDate := today.AddDate(0, 0, -2)
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE OR REPLACE FUNCTION public.test_staff_attendance_audit_failure()
        RETURNS trigger LANGUAGE plpgsql AS $body$
        BEGIN
            IF NEW.action = 'school.staff_attendance.marked'
               AND NEW.metadata->>'attendance_date' = '%s' THEN
                RAISE EXCEPTION 'forced staff attendance audit failure';
            END IF;
            RETURN NEW;
        END;
        $body$;
    `, singleFailureDate.Format("2006-01-02"))); err != nil {
        t.Fatalf("create attendance audit failure function: %v", err)
    }
    if _, err := tx.Exec(ctx, `
        CREATE TRIGGER trg_test_staff_attendance_audit_failure
        BEFORE INSERT ON public.audit_events
        FOR EACH ROW EXECUTE FUNCTION public.test_staff_attendance_audit_failure()
    `); err != nil { t.Fatalf("create attendance audit failure trigger: %v", err) }
    _, singleAuditErr := attendanceRepo.Mark(ctx, actorID, schoolID, secondStaff.PrincipalID, singleFailureDate, "present", nil)
    if singleAuditErr == nil { t.Fatal("individual attendance mark succeeded despite forced audit failure") }
    var singleRowCount int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM staff_attendance_records
        WHERE school_id=$1 AND staff_principal_id=$2 AND attendance_date=$3::date
    `, schoolID, secondStaff.PrincipalID, singleFailureDate.Format("2006-01-02")).Scan(&singleRowCount); err != nil {
        t.Fatalf("check individual attendance rollback: %v", err)
    }
    if singleRowCount != 0 { t.Fatalf("individual attendance rows after audit failure=%d want=0", singleRowCount) }
    if _, err := tx.Exec(ctx, `DROP TRIGGER trg_test_staff_attendance_audit_failure ON public.audit_events`); err != nil { t.Fatalf("drop individual audit failure trigger: %v", err) }
    if _, err := tx.Exec(ctx, `DROP FUNCTION public.test_staff_attendance_audit_failure()`); err != nil { t.Fatalf("drop individual audit failure function: %v", err) }

    bulkFailureDate := today.AddDate(0, 0, -3)
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE OR REPLACE FUNCTION public.test_staff_attendance_bulk_audit_failure()
        RETURNS trigger LANGUAGE plpgsql AS $body$
        BEGIN
            IF NEW.action = 'school.staff_attendance.marked'
               AND NEW.metadata->>'attendance_date' = '%s'
               AND NEW.metadata->>'staff_principal_id' = '%d' THEN
                RAISE EXCEPTION 'forced second staff attendance audit failure';
            END IF;
            RETURN NEW;
        END;
        $body$;
    `, bulkFailureDate.Format("2006-01-02"), secondStaff.PrincipalID)); err != nil {
        t.Fatalf("create bulk attendance audit failure function: %v", err)
    }
    if _, err := tx.Exec(ctx, `
        CREATE TRIGGER trg_test_staff_attendance_bulk_audit_failure
        BEFORE INSERT ON public.audit_events
        FOR EACH ROW EXECUTE FUNCTION public.test_staff_attendance_bulk_audit_failure()
    `); err != nil { t.Fatalf("create bulk attendance audit failure trigger: %v", err) }
    _, bulkAuditErr := attendanceRepo.MarkBulk(ctx, actorID, schoolID, []int64{staff.PrincipalID, secondStaff.PrincipalID}, bulkFailureDate, "present", nil)
    if bulkAuditErr == nil { t.Fatal("bulk attendance succeeded despite forced second audit failure") }
    var partialRows, partialAudits int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM staff_attendance_records
        WHERE school_id=$1 AND staff_principal_id=ANY($2::bigint[]) AND attendance_date=$3::date
    `, schoolID, []int64{staff.PrincipalID, secondStaff.PrincipalID}, bulkFailureDate.Format("2006-01-02")).Scan(&partialRows); err != nil {
        t.Fatalf("check bulk attendance rollback: %v", err)
    }
    if partialRows != 0 { t.Fatalf("bulk attendance rows after second audit failure=%d want=0", partialRows) }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
        WHERE school_id=$1 AND action='school.staff_attendance.marked'
          AND metadata->>'attendance_date'=$2
          AND metadata->>'staff_principal_id'=ANY($3::text[])
    `, schoolID, bulkFailureDate.Format("2006-01-02"), []string{fmt.Sprint(staff.PrincipalID), fmt.Sprint(secondStaff.PrincipalID)}).Scan(&partialAudits); err != nil {
        t.Fatalf("check bulk attendance audit rollback: %v", err)
    }
    if partialAudits != 0 { t.Fatalf("bulk attendance audit rows after second audit failure=%d want=0", partialAudits) }
    if _, err := tx.Exec(ctx, `DROP TRIGGER trg_test_staff_attendance_bulk_audit_failure ON public.audit_events`); err != nil { t.Fatalf("drop bulk audit failure trigger: %v", err) }
    if _, err := tx.Exec(ctx, `DROP FUNCTION public.test_staff_attendance_bulk_audit_failure()`); err != nil { t.Fatalf("drop bulk audit failure function: %v", err) }

	if _,err:=repo.UpdateStaffQualification(ctx,actorID,schoolID+1,staff.PrincipalID,qualification.ID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"Cross tenant",Institution:"Denied",
	});!errors.Is(err,ErrStaffRecordAccessDenied){t.Fatalf("cross-school qualification update error=%v",err)}

	// Exercise every staff-record family with the caller's real school
	// permission but a forged foreign school ID. No read or write may cross it.
	if _, err := repo.ListStaffCertifications(ctx, actorID, schoolID+1, staff.PrincipalID); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school certification read error=%v want access denied", err)
	}
	if _, err := repo.ListStaffExperience(ctx, actorID, schoolID+1, staff.PrincipalID); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school experience read error=%v want access denied", err)
	}
	if _, err := repo.CreateStaffQualification(ctx, actorID, schoolID+1, staff.PrincipalID, StaffQualification{QualificationType:"degree", QualificationName:"Foreign", Institution:"Denied"}); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school qualification create error=%v want access denied", err)
	}
	if _, err := repo.CreateStaffCertification(ctx, actorID, schoolID+1, staff.PrincipalID, StaffCertification{CertificateName:"Foreign", IssuingOrganization:"Denied"}); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school certification create error=%v want access denied", err)
	}
	if _, err := repo.CreateStaffExperience(ctx, actorID, schoolID+1, staff.PrincipalID, StaffExperience{EmployerName:"Foreign", Designation:"Denied", EmploymentType:"permanent", StartDate:"2020-01-01"}); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school experience create error=%v want access denied", err)
	}
	if _, err := repo.UpdateStaffCertification(ctx, actorID, schoolID+1, staff.PrincipalID, certification.ID, StaffCertification{CertificateName:"Foreign", IssuingOrganization:"Denied"}); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school certification update error=%v want access denied", err)
	}
	if _, err := repo.UpdateStaffExperience(ctx, actorID, schoolID+1, staff.PrincipalID, experience.ID, StaffExperience{EmployerName:"Foreign", Designation:"Denied", EmploymentType:"permanent", StartDate:"2020-01-01"}); !errors.Is(err, ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school experience update error=%v want access denied", err)
	}
	var qualificationRows, certificationRows, experienceRows int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM staff_qualifications WHERE staff_principal_id=$1`, staff.PrincipalID).Scan(&qualificationRows); err != nil { t.Fatalf("count qualifications after denied writes: %v", err) }
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM staff_certifications WHERE staff_principal_id=$1`, staff.PrincipalID).Scan(&certificationRows); err != nil { t.Fatalf("count certifications after denied writes: %v", err) }
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM staff_experience WHERE staff_principal_id=$1`, staff.PrincipalID).Scan(&experienceRows); err != nil { t.Fatalf("count experience after denied writes: %v", err) }
	if qualificationRows != 1 || certificationRows != 1 || experienceRows != 1 {
		t.Fatalf("cross-school staff-record writes changed row counts: qualifications=%d certifications=%d experience=%d", qualificationRows, certificationRows, experienceRows)
	}

	qualifications,err:=repo.ListStaffQualifications(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(qualifications)!=1||qualifications[0].ID!=qualification.ID{t.Fatalf("list qualifications: items=%+v err=%v",qualifications,err)}
	certifications,err:=repo.ListStaffCertifications(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(certifications)!=1||certifications[0].ID!=certification.ID{t.Fatalf("list certifications: items=%+v err=%v",certifications,err)}

	report, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{RecordType:"all",Limit:50})
	if err != nil { t.Fatalf("list school-wide qualifications report: %v",err) }
	if report.Total != 2 || len(report.Rows) != 2 { t.Fatalf("report total=%d rows=%+v want both qualification and certification",report.Total,report.Rows) }
	if report.Rows[0].RecordType != "certification" && report.Rows[0].RecordType != "qualification" { t.Fatalf("unexpected report row ordering: %+v",report.Rows) }
	for _, row := range report.Rows {
		if row.StaffPrincipalID != staff.PrincipalID || row.EmployeeCode != staff.EmployeeCode || row.StaffName != staff.FullName {
			t.Fatalf("report row omitted same-school staff identity: %+v",row)
		}
	}
	filtered, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{RecordType:"certification",VerificationStatus:"pending",Limit:1})
	if err != nil || filtered.Total != 1 || len(filtered.Rows) != 1 || filtered.Rows[0].RecordType != "certification" {
		t.Fatalf("filtered report: page=%+v err=%v",filtered,err)
	}
	firstQualificationPage, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{RecordType:"all",Limit:1,Offset:0})
	if err != nil || firstQualificationPage.Total != 2 || len(firstQualificationPage.Rows) != 1 {
		t.Fatalf("qualification report first page: page=%+v err=%v",firstQualificationPage,err)
	}
	secondQualificationPage, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{RecordType:"all",Limit:1,Offset:1})
	if err != nil || secondQualificationPage.Total != 2 || len(secondQualificationPage.Rows) != 1 {
		t.Fatalf("qualification report second page: page=%+v err=%v",secondQualificationPage,err)
	}
	if firstQualificationPage.Rows[0].RecordType != report.Rows[0].RecordType ||
		firstQualificationPage.Rows[0].RecordID != report.Rows[0].RecordID ||
		secondQualificationPage.Rows[0].RecordType != report.Rows[1].RecordType ||
		secondQualificationPage.Rows[0].RecordID != report.Rows[1].RecordID {
		t.Fatalf("qualification report pages are not stable: all=%+v first=%+v second=%+v",
			report.Rows, firstQualificationPage.Rows, secondQualificationPage.Rows)
	}
	if _, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID+1, StaffQualificationReportFilter{RecordType:"all",Limit:50}); !errors.Is(err,ErrStaffRecordAccessDenied) {
		t.Fatalf("cross-school report access error=%v want access denied",err)
	}
	experiences,err:=repo.ListStaffExperience(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(experiences)!=1||experiences[0].ID!=experience.ID{t.Fatalf("list experience: items=%+v err=%v",experiences,err)}
    // Add historical rows directly so this repository integration test can
    // verify report pagination without changing the audited CRUD count above.
    if _,err:=tx.Exec(ctx, `
        INSERT INTO staff_experience
            (school_id,staff_principal_id,employer_name,designation,employment_type,start_date,end_date,is_current)
        VALUES
            ($1,$2,'Recent Academy','Senior Teacher','contract','2022-04-01','2024-03-31',FALSE),
            ($1,$2,'Older Academy','Teacher','temporary','2015-04-01','2017-03-31',FALSE)
    `,schoolID,staff.PrincipalID);err!=nil{t.Fatalf("insert experience report pagination fixtures: %v",err)}
    experienceReport,err:=repo.ListStaffExperienceReport(ctx,actorID,schoolID,StaffExperienceReportFilter{Limit:2})
    if err!=nil || experienceReport.Total!=3 || len(experienceReport.Rows)!=2 {t.Fatalf("experience report first page=%+v err=%v want 2 of 3",experienceReport,err)}
    if experienceReport.Rows[0].EmployerName!="Recent Academy" || experienceReport.Rows[1].RecordID!=experience.ID || experienceReport.Rows[1].PreviousDesignation!=updatedExperience.Designation {
        t.Fatalf("experience report first-page ordering/identity incorrect: %+v",experienceReport.Rows)
    }
    secondExperiencePage,err:=repo.ListStaffExperienceReport(ctx,actorID,schoolID,StaffExperienceReportFilter{Limit:2,Offset:2})
    if err!=nil || secondExperiencePage.Total!=3 || len(secondExperiencePage.Rows)!=1 || secondExperiencePage.Rows[0].EmployerName!="Older Academy" {
        t.Fatalf("experience report second page=%+v err=%v want final Older Academy row",secondExperiencePage,err)
    }
    currentOnly,err:=repo.ListStaffExperienceReport(ctx,actorID,schoolID,StaffExperienceReportFilter{CurrentOnly:true,Limit:50})
    if err!=nil || currentOnly.Total!=0 {t.Fatalf("current-only experience report page=%+v err=%v",currentOnly,err)}
    filteredExperience,err:=repo.ListStaffExperienceReport(ctx,actorID,schoolID,StaffExperienceReportFilter{EmploymentType:experience.EmploymentType,Limit:1})
    if err!=nil || filteredExperience.Total!=1 || len(filteredExperience.Rows)!=1 {t.Fatalf("filtered experience report page=%+v err=%v",filteredExperience,err)}
    if _,err:=repo.ListStaffExperienceReport(ctx,actorID,schoolID+1,StaffExperienceReportFilter{Limit:50});!errors.Is(err,ErrStaffRecordAccessDenied) {
        t.Fatalf("cross-school experience report error=%v want access denied",err)
    }

	if _,err:=repo.ListStaffQualifications(ctx,actorID,schoolID+1,staff.PrincipalID);!errors.Is(err,ErrStaffRecordAccessDenied){
		t.Fatalf("cross-school record read error=%v want ErrStaffRecordAccessDenied",err)
	}
	var auditCount int
	if err:=tx.QueryRow(ctx,`SELECT COUNT(*) FROM audit_events WHERE actor_principal_id=$1 AND school_id=$2 AND action IN ('staff.qualification.created','staff.certification.created','staff.experience.created','staff.qualification.updated','staff.certification.updated','staff.experience.updated')`,actorID,schoolID).Scan(&auditCount);err!=nil{
		t.Fatalf("count staff record audit events: %v",err)
	}
	if auditCount!=6{t.Fatalf("audit count=%d want=6 (three creates and three updates)",auditCount)}
	// Leave and substitution activity is part of a staff member's relevant history.
	if _,err:=tx.Exec(ctx,`
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES
		($1,$2,'staff.leave.requested','staff_leave_request','history-leave-fixture','success',jsonb_build_object('staff_principal_id',$3::bigint)),
		($1,$2,'staff.substitution.assigned','staff_substitution','history-substitution-fixture','success',jsonb_build_object('original_teacher_principal_id',$3::bigint,'substitute_teacher_principal_id',$4::bigint)),
		($1,$2,'school.staff.permission_override.set','user_permission_override',$3::text,'success',jsonb_build_object('permission_key','attendance.update','effect','deny')),
		($1,$2,'school.staff.permission_override.removed','user_permission_override',$3::text,'success',jsonb_build_object('permission_key','attendance.update')),
		($1,$2,'school.staff.role_assigned','staff_principal',$3::text,'success',jsonb_build_object('role_id',4)),
		($1,$2,'school.staff.role_revoked','staff_principal',$3::text,'success',jsonb_build_object('role_id',4)),
		($1,$2,'staff.account.disabled','staff_user',$5::text,'success',jsonb_build_object('target_principal_id',$3::bigint,'sessions_revoked',TRUE)),
		($1,$2,'staff.account.enabled','staff_user',$5::text,'success',jsonb_build_object('must_change_password',TRUE,'sessions_revoked',TRUE)),
		($1,$2,'staff.password.reset','staff_user',$5::text,'success',jsonb_build_object('must_change_password',TRUE)),
		($1,$2,'staff.account.updated','staff_user',$5::text,'success',jsonb_build_object('old_username','old-staff','username','new-staff','sessions_revoked',TRUE)),
		($1,$2,'staff.password.changed','staff_user',$5::text,'success',jsonb_build_object('password_change_required_cleared',TRUE))
	`,actorID,schoolID,staff.PrincipalID,secondStaff.PrincipalID,fmt.Sprint(staff.ID));err!=nil{t.Fatalf("insert leave/substitution history fixtures: %v",err)}
	// Decision audit events intentionally omit employee IDs from metadata.
	// The history query must resolve the employee through the school-scoped
	// leave request resource ID instead.
	leaveRepo:=NewStaffLeaveRepository(tx)
	leaveType,err:=leaveRepo.SaveType(ctx,actorID,schoolID,0,StaffLeaveType{Name:"History Integration Leave",IsPaid:true,RequiresApproval:true,IsActive:true})
	if err!=nil{t.Fatalf("create leave type for history fallback: %v",err)}
	leaveStart:=today.AddDate(0,0,10).Format("2006-01-02")
	leaveRequest,err:=leaveRepo.CreateRequest(ctx,actorID,schoolID,StaffLeaveRequest{
		StaffPrincipalID:staff.PrincipalID,LeaveTypeID:leaveType.ID,StartDate:leaveStart,EndDate:leaveStart,
		DurationType:"full_day",Reason:"History lookup integration fixture",
	})
	if err!=nil{t.Fatalf("create leave request for history fallback: %v",err)}
	if err:=leaveRepo.Decide(ctx,actorID,schoolID,leaveRequest.ID,true,nil);err!=nil{t.Fatalf("approve leave request for history fallback: %v",err)}
	cancelDate:=today.AddDate(0,0,11).Format("2006-01-02")
	cancelRequest,err:=leaveRepo.CreateRequest(ctx,actorID,schoolID,StaffLeaveRequest{
		StaffPrincipalID:staff.PrincipalID,LeaveTypeID:leaveType.ID,StartDate:cancelDate,EndDate:cancelDate,
		DurationType:"full_day",Reason:"Cancellation history integration fixture",
	})
	if err!=nil{t.Fatalf("create leave request for cancellation history: %v",err)}
	if err:=leaveRepo.Cancel(ctx,actorID,schoolID,cancelRequest.ID);err!=nil{t.Fatalf("cancel leave request for history fallback: %v",err)}
	rejectDate:=today.AddDate(0,0,12).Format("2006-01-02")
	rejectRequest,err:=leaveRepo.CreateRequest(ctx,actorID,schoolID,StaffLeaveRequest{
		StaffPrincipalID:staff.PrincipalID,LeaveTypeID:leaveType.ID,StartDate:rejectDate,EndDate:rejectDate,
		DurationType:"full_day",Reason:"Rejection history integration fixture",
	})
	if err!=nil{t.Fatalf("create leave request for rejection history: %v",err)}
	rejectionComment:="Not approved in integration fixture"
	if err:=leaveRepo.Decide(ctx,actorID,schoolID,rejectRequest.ID,false,&rejectionComment);err!=nil{t.Fatalf("reject leave request for history fallback: %v",err)}
	firstPage,err:=repo.ListStaffAuditEvents(ctx,actorID,schoolID,staff.PrincipalID,StaffAuditEventFilter{Limit:2,Offset:0})
	if err!=nil{t.Fatalf("list first staff audit history page: %v",err)}
	if len(firstPage.Events)!=2||!firstPage.HasMore||firstPage.Offset!=0||firstPage.Limit!=2{t.Fatalf("first audit page=%+v want 2 rows and has_more=true",firstPage)}
	secondPage,err:=repo.ListStaffAuditEvents(ctx,actorID,schoolID,staff.PrincipalID,StaffAuditEventFilter{Limit:2,Offset:2})
	if err!=nil{t.Fatalf("list second staff audit history page: %v",err)}
	if len(secondPage.Events)!=2||secondPage.Events[0].ID==firstPage.Events[0].ID{t.Fatalf("second audit page did not advance: first=%+v second=%+v",firstPage,secondPage)}
	allEvents,err:=repo.ListStaffAuditEvents(ctx,actorID,schoolID,staff.PrincipalID,StaffAuditEventFilter{Limit:100,Offset:0})
	if err!=nil{t.Fatalf("list staff audit history: %v",err)}
	events:=allEvents.Events
	if len(events)<9{t.Fatalf("staff audit history events=%d want at least 9 including account creation, six record mutations and two attendance actions",len(events))}
	foundQualificationUpdate:=false
	for _,event:=range events{if event.Action=="staff.qualification.updated"{foundQualificationUpdate=true}}
	if !foundQualificationUpdate{t.Fatal("staff audit history omitted qualification update")}
    foundAttendanceUpdate := false
    for _, event := range events { if event.Action == "school.staff_attendance.updated" { foundAttendanceUpdate = true } }
    if !foundAttendanceUpdate { t.Fatal("staff audit history omitted staff attendance update") }
	foundLeaveActivity,foundSubstitutionActivity:=false,false
	foundApprovedLeaveActivity,foundRejectedLeaveActivity,foundCancelledLeaveActivity:=false,false,false
	for _,event:=range events{
		if event.Action=="staff.leave.requested"{foundLeaveActivity=true}
		if event.Action=="staff.substitution.assigned"{foundSubstitutionActivity=true}
		if event.Action=="staff.leave.approved"{foundApprovedLeaveActivity=true}
		if event.Action=="staff.leave.rejected"{foundRejectedLeaveActivity=true}
		if event.Action=="staff.leave.cancelled"{foundCancelledLeaveActivity=true}
	}
	if !foundLeaveActivity{t.Fatal("staff audit history omitted leave activity")}
	if !foundApprovedLeaveActivity{t.Fatal("staff audit history omitted leave approval resolved through resource ID")}
	if !foundRejectedLeaveActivity{t.Fatal("staff audit history omitted leave rejection resolved through resource ID")}
	if !foundCancelledLeaveActivity{t.Fatal("staff audit history omitted leave cancellation resolved through resource ID")}
	if !foundSubstitutionActivity{t.Fatal("staff audit history omitted substitution activity")}
	foundOverrideSet,foundOverrideRemoved:=false,false
	for _,event:=range events{
		if event.Action=="school.staff.permission_override.set"{foundOverrideSet=true}
		if event.Action=="school.staff.permission_override.removed"{foundOverrideRemoved=true}
	}
	if !foundOverrideSet{t.Fatal("staff audit history omitted permission override set event")}
	if !foundOverrideRemoved{t.Fatal("staff audit history omitted permission override removal event")}
	foundRoleAssigned,foundRoleRevoked:=false,false
	for _,event:=range events{
		if event.Action=="school.staff.role_assigned"{foundRoleAssigned=true}
		if event.Action=="school.staff.role_revoked"{foundRoleRevoked=true}
	}
	if !foundRoleAssigned{t.Fatal("staff audit history omitted role assignment event")}
	if !foundRoleRevoked{t.Fatal("staff audit history omitted role revocation event")}
	foundAccountDisabled,foundAccountEnabled,foundPasswordReset,foundAccountUpdated,foundPasswordChanged:=false,false,false,false,false
	for _,event:=range events{
		if event.Action=="staff.account.disabled"{foundAccountDisabled=true}
		if event.Action=="staff.account.enabled"{foundAccountEnabled=true}
		if event.Action=="staff.password.reset"{foundPasswordReset=true}
		if event.Action=="staff.account.updated"{foundAccountUpdated=true}
		if event.Action=="staff.password.changed"{foundPasswordChanged=true}
	}
	if !foundAccountDisabled{t.Fatal("staff audit history omitted account disable event")}
	if !foundAccountEnabled{t.Fatal("staff audit history omitted account re-enable event")}
	if !foundPasswordReset{t.Fatal("staff audit history omitted password reset event")}
	if !foundAccountUpdated{t.Fatal("staff audit history omitted username update event")}
	if !foundPasswordChanged{t.Fatal("staff audit history omitted password changed event")}
	// Legacy staff accounts without staff_profiles still have account audit history.
	legacyUsername := "shl-" + suffix
	legacy, err := repo.CreateStaffAccount(ctx, actorID, schoolID, legacyUsername, "integration-test-password-hash")
	if err != nil { t.Fatalf("create profile-incomplete legacy staff account: %v", err) }
	if legacy.ProfileComplete { t.Fatal("legacy staff fixture unexpectedly has a completed profile") }
	legacyHistory, err := repo.ListStaffAuditEvents(ctx, actorID, schoolID, legacy.PrincipalID, StaffAuditEventFilter{Limit:50})
	if err != nil { t.Fatalf("list audit history for profile-incomplete legacy staff account: %v", err) }
	foundLegacyCreate := false
	for _, event := range legacyHistory.Events {
		if event.Action == "staff.account.created" { foundLegacyCreate = true }
	}
	if !foundLegacyCreate { t.Fatal("profile-incomplete legacy staff history omitted account creation event") }
	if _,err:=repo.ListStaffAuditEvents(ctx,actorID,schoolID+1,staff.PrincipalID,StaffAuditEventFilter{Limit:50});!errors.Is(err,ErrStaffRecordAccessDenied){t.Fatalf("cross-school staff audit history error=%v",err)}
}
