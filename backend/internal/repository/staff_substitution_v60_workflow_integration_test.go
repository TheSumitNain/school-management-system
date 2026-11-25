package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestStaffSubstitutionV60DatabaseGuardsPostgresIntegration(t *testing.T) {
	_,pool,userID:=newPostgresIntegrationRepository(t)
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
	defer cancel()
	tx,err:=pool.Begin(ctx)
	if err!=nil{t.Fatalf("begin rollback-scoped substitution test: %v",err)}
	defer func(){_=tx.Rollback(ctx)}()

	var schoolID,actorPrincipalID int64
	if err=tx.QueryRow(ctx,`SELECT school_id,principal_id FROM users WHERE id=$1`,userID).Scan(&schoolID,&actorPrincipalID);err!=nil{t.Fatalf("load actor fixture: %v",err)}

	// Teacher assignments require an active School Admin actor. Resolve or
	// create a rollback-scoped Platform Admin exactly as the academic V17 tests do.
	var schoolAdminRoleID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`,schoolID).Scan(&schoolAdminRoleID);err!=nil{t.Fatalf("load School Admin role: %v",err)}
	var platformAdminPrincipalID int64
	err=tx.QueryRow(ctx,`SELECT ap.id FROM platform_admin_accounts pa
		JOIN auth_principals ap ON ap.id=pa.principal_id
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		JOIN role_permissions rp ON rp.role_id=r.id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE ap.principal_type='platform_admin' AND ap.status='active'
		  AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
		  AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
		LIMIT 1`).Scan(&platformAdminPrincipalID)
	if errors.Is(err,pgx.ErrNoRows){
		var bootstrapExists bool
		if checkErr:=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)
			OR EXISTS(SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
				WHERE r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin')`).Scan(&bootstrapExists);checkErr!=nil{t.Fatalf("check Platform Admin bootstrap state: %v",checkErr)}
		if bootstrapExists{t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")}
		platformAdminPrincipalID,err=NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx,tx,fmt.Sprintf("substitution-test-%d@example.invalid",time.Now().UnixNano()),"integration-test-password-hash")
	}
	if err!=nil{t.Fatalf("resolve rollback-scoped Platform Admin actor: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,actorPrincipalID,schoolAdminRoleID,platformAdminPrincipalID);err!=nil{t.Fatalf("grant School Admin role to test actor: %v",err)}

	suffix:=time.Now().UnixNano()
	var originalUserID,originalID,substituteUserID,substituteID int64
	if err=tx.QueryRow(ctx,`INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,$3,'active') RETURNING id,principal_id`,schoolID,fmt.Sprintf("original-%d",suffix),"integration-test-password-hash").Scan(&originalUserID,&originalID);err!=nil{t.Fatalf("create original teacher user: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,$3,'active') RETURNING id,principal_id`,schoolID,fmt.Sprintf("substitute-%d",suffix),"integration-test-password-hash").Scan(&substituteUserID,&substituteID);err!=nil{t.Fatalf("create substitute user: %v",err)}
	var teacherRoleID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		AND scope_type='school' AND role_key='teacher' AND is_system=TRUE`,schoolID).Scan(&teacherRoleID);err!=nil{t.Fatalf("load Teacher role: %v",err)}
	for _,teacherID:=range []int64{originalID,substituteID}{
		if _,err=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,teacherID,teacherRoleID,actorPrincipalID);err!=nil{t.Fatalf("assign Teacher role to %d: %v",teacherID,err)}
	}

	for _,staff:=range []struct{id int64;code,name string}{{originalID,fmt.Sprintf("OR-%d",suffix),"Original Teacher"},{substituteID,fmt.Sprintf("SU-%d",suffix),"Substitute Teacher"}} {
		if _,err=tx.Exec(ctx,`INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number)
			VALUES($1,$2,$3,$4,'Teacher','9876543210')`,staff.id,schoolID,staff.code,staff.name);err!=nil{t.Fatalf("create staff profile %s: %v",staff.name,err)}
	}

	var yearID,classID,sectionID,subjectID,periodID,originalAssignmentID,substituteAssignmentID,timetableEntryID int64
	if err=tx.QueryRow(ctx,`INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active') RETURNING id`,schoolID,fmt.Sprintf("substitution-year-%d",suffix)).Scan(&yearID);err!=nil{t.Fatalf("create academic year: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,fmt.Sprintf("Substitution Class %d",suffix)).Scan(&classID);err!=nil{t.Fatalf("create class: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name) VALUES($1,$2,$3,'A') RETURNING id`,schoolID,yearID,classID).Scan(&sectionID);err!=nil{t.Fatalf("create section: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO subjects(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,fmt.Sprintf("Substitution Subject %d",suffix)).Scan(&subjectID);err!=nil{t.Fatalf("create subject: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Period 1',TIME '09:00',TIME '09:40') RETURNING id`,schoolID,yearID).Scan(&periodID);err!=nil{t.Fatalf("create period: %v",err)}
	var secondPeriodID int64
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,2,'Period 2',TIME '10:00',TIME '10:40') RETURNING id`,schoolID,yearID).Scan(&secondPeriodID);err!=nil{t.Fatalf("create second period: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,sectionID,subjectID,originalID,actorPrincipalID).Scan(&originalAssignmentID);err!=nil{t.Fatalf("assign original teacher: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,sectionID,subjectID,substituteID,actorPrincipalID).Scan(&substituteAssignmentID);err!=nil{t.Fatalf("assign substitute teacher: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,room,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,'Original room',$8,$8) RETURNING id`,schoolID,yearID,sectionID,subjectID,originalID,originalAssignmentID,periodID,originalID).Scan(&timetableEntryID);err!=nil{t.Fatalf("create recurring timetable entry: %v",err)}

	// Use the next Monday in the school's IST calendar so this remains a
	// future substitution regardless of when CI runs.
	nowIST:=time.Now().In(time.FixedZone("IST",5*60*60+30*60))
	daysUntilMonday:=(int(time.Monday)-int(nowIST.Weekday())+7)%7
	if daysUntilMonday==0{daysUntilMonday=7}
	coverDate:=nowIST.AddDate(0,0,daysUntilMonday).Format("2006-01-02")
	var substitutionID int64
	if err=tx.QueryRow(ctx,`INSERT INTO staff_substitutions(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,substitute_teacher_principal_id,reason,created_by_principal_id)
		VALUES($1,$2,$3,$4::date,$5,$6,'Cover for approved absence',$5) RETURNING id`,schoolID,timetableEntryID,periodID,coverDate,originalID,substituteID).Scan(&substitutionID);err!=nil{t.Fatalf("create valid dated substitution: %v",err)}

	duplicate,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin duplicate savepoint: %v",err)}
	_,duplicateErr:=duplicate.Exec(ctx,`INSERT INTO staff_substitutions(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,substitute_teacher_principal_id,reason,created_by_principal_id)
		VALUES($1,$2,$3,$4::date,$5,$6,'Duplicate cover',$5)`,schoolID,timetableEntryID,periodID,coverDate,originalID,substituteID)
	if rollbackErr:=duplicate.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback duplicate savepoint: %v",rollbackErr)}
	if duplicateErr==nil{t.Fatal("duplicate active cover for one timetable entry/date was accepted")}

	inactive,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin substitute employment guard savepoint: %v",err)}
	_,inactiveErr:=inactive.Exec(ctx,`UPDATE staff_profiles SET employment_status='inactive' WHERE principal_id=$1`,substituteID)
	if rollbackErr:=inactive.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback substitute employment savepoint: %v",rollbackErr)}
	if inactiveErr==nil{t.Fatal("substitute employment was deactivated while future coverage remained active")}

	disabled,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin substitute login guard savepoint: %v",err)}
	_,disabledErr:=disabled.Exec(ctx,`UPDATE auth_principals SET status='disabled' WHERE id=$1`,substituteID)
	if rollbackErr:=disabled.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback substitute login savepoint: %v",rollbackErr)}
	if disabledErr==nil{t.Fatal("substitute login was disabled while future coverage remained active")}

	reasonEdit,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin immutable reason savepoint: %v",err)}
	_,reasonErr:=reasonEdit.Exec(ctx,`UPDATE staff_substitutions SET reason='Changed reason' WHERE id=$1`,substitutionID)
	if rollbackErr:=reasonEdit.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback immutable reason savepoint: %v",rollbackErr)}
	if reasonErr==nil{t.Fatal("substitution reason was mutable after creation")}

	deleteHistory,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin immutable delete savepoint: %v",err)}
	_,deleteErr:=deleteHistory.Exec(ctx,`DELETE FROM staff_substitutions WHERE id=$1`,substitutionID)
	if rollbackErr:=deleteHistory.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback immutable delete savepoint: %v",rollbackErr)}
	if deleteErr==nil{t.Fatal("substitution history was deleted instead of cancelled")}

	revokeAssignment,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin assignment revocation guard savepoint: %v",err)}
	_,revokeErr:=revokeAssignment.Exec(ctx,`UPDATE teacher_subject_assignments SET revoked_at=NOW() WHERE id=$1`,substituteAssignmentID)
	if rollbackErr:=revokeAssignment.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback assignment revocation savepoint: %v",rollbackErr)}
	if revokeErr==nil{t.Fatal("substitute's matching assignment was revoked while future cover remained active")}

	update,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin recurring schedule guard savepoint: %v",err)}
	_,updateErr:=update.Exec(ctx,`UPDATE timetable_entries SET room='Changed room' WHERE id=$1`,timetableEntryID)
	if rollbackErr:=update.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback schedule guard savepoint: %v",rollbackErr)}
	if updateErr==nil{t.Fatal("recurring timetable entry changed while an active dated substitution referenced it")}

	var secondSectionID int64
	if err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name) VALUES($1,$2,$3,'B') RETURNING id`,schoolID,yearID,classID).Scan(&secondSectionID);err!=nil{t.Fatalf("create second section: %v",err)}
	var secondAssignmentID int64
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,secondSectionID,subjectID,substituteID,actorPrincipalID).Scan(&secondAssignmentID);err!=nil{t.Fatalf("assign substitute to second section: %v",err)}
	conflictSlot,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin conflicting recurring slot savepoint: %v",err)}
	_,slotErr:=conflictSlot.Exec(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8)`,schoolID,yearID,secondSectionID,subjectID,substituteID,secondAssignmentID,periodID,actorPrincipalID)
	if rollbackErr:=conflictSlot.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback conflicting timetable slot savepoint: %v",rollbackErr)}
	if slotErr==nil{t.Fatal("new recurring timetable slot conflicted with active dated substitution but was accepted")}

	// Force the assignment audit insert to fail and prove the repository transaction
	// does not leave an unaudited substitution row behind.
	var auditFailureEntryID int64
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,schoolID,yearID,sectionID,subjectID,originalID,originalAssignmentID,secondPeriodID,actorPrincipalID).Scan(&auditFailureEntryID);err!=nil{t.Fatalf("create second timetable entry for audit rollback test: %v",err)}
	const assignAuditTrigger="trg_test_staff_substitution_assign_audit_failure"
	if _,err=tx.Exec(ctx,`CREATE OR REPLACE FUNCTION public.test_staff_substitution_assign_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $ BEGIN IF NEW.action='staff.substitution.assigned' THEN RAISE EXCEPTION 'forced substitution assignment audit failure'; END IF; RETURN NEW; END $`);err!=nil{t.Fatalf("create assignment audit failure function: %v",err)}
	if _,err=tx.Exec(ctx,`CREATE TRIGGER trg_test_staff_substitution_assign_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_staff_substitution_assign_audit_failure()`);err!=nil{t.Fatalf("create assignment audit failure trigger: %v",err)}
	_,createAuditErr:=(&AcademicRepository{db:tx}).CreateStaffSubstitution(ctx,actorPrincipalID,schoolID,auditFailureEntryID,originalID,substituteID,coverDate,nil,"Audit rollback test")
	if createAuditErr==nil{t.Fatal("substitution creation unexpectedly succeeded when assignment audit insert failed")}
	var unauditedRows int
	if err=tx.QueryRow(ctx,`SELECT count(*) FROM staff_substitutions WHERE school_id=$1 AND timetable_entry_id=$2 AND cover_date=$3::date AND status='active'`,schoolID,auditFailureEntryID,coverDate).Scan(&unauditedRows);err!=nil{t.Fatalf("count substitution rows after assignment audit failure: %v",err)}
	if unauditedRows!=0{t.Fatalf("assignment audit failure left %d substitution rows; want 0",unauditedRows)}
	if _,err=tx.Exec(ctx,`DROP TRIGGER IF EXISTS trg_test_staff_substitution_assign_audit_failure ON public.audit_events`);err!=nil{t.Fatalf("drop assignment audit failure trigger: %v",err)}
	if _,err=tx.Exec(ctx,`DROP FUNCTION IF EXISTS public.test_staff_substitution_assign_audit_failure()`);err!=nil{t.Fatalf("drop assignment audit failure function: %v",err)}

	// Cancellation is equally atomic: an audit failure must leave the cover active.
	if _,err=tx.Exec(ctx,`CREATE OR REPLACE FUNCTION public.test_staff_substitution_cancel_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $ BEGIN IF NEW.action='staff.substitution.cancelled' THEN RAISE EXCEPTION 'forced substitution cancellation audit failure'; END IF; RETURN NEW; END $`);err!=nil{t.Fatalf("create cancellation audit failure function: %v",err)}
	if _,err=tx.Exec(ctx,`CREATE TRIGGER trg_test_staff_substitution_cancel_audit_failure BEFORE INSERT ON public.audit_events FOR EACH ROW EXECUTE FUNCTION public.test_staff_substitution_cancel_audit_failure()`);err!=nil{t.Fatalf("create cancellation audit failure trigger: %v",err)}
	cancelAuditErr:=(&AcademicRepository{db:tx}).CancelStaffSubstitution(ctx,actorPrincipalID,schoolID,substitutionID)
	if cancelAuditErr==nil{t.Fatal("substitution cancellation unexpectedly succeeded when audit insert failed")}
	var activeAfterAuditFailure int
	if err=tx.QueryRow(ctx,`SELECT count(*) FROM staff_substitutions WHERE school_id=$1 AND id=$2 AND status='active'`,schoolID,substitutionID).Scan(&activeAfterAuditFailure);err!=nil{t.Fatalf("read substitution after cancellation audit failure: %v",err)}
	if activeAfterAuditFailure!=1{t.Fatalf("cancellation audit failure left active row count=%d; want 1",activeAfterAuditFailure)}
	if _,err=tx.Exec(ctx,`DROP TRIGGER IF EXISTS trg_test_staff_substitution_cancel_audit_failure ON public.audit_events`);err!=nil{t.Fatalf("drop cancellation audit failure trigger: %v",err)}
	if _,err=tx.Exec(ctx,`DROP FUNCTION IF EXISTS public.test_staff_substitution_cancel_audit_failure()`);err!=nil{t.Fatalf("drop cancellation audit failure function: %v",err)}

	// Exercise the production cancellation workflow, including its atomic audit write.
	// Cancellation audit rows intentionally omit teacher IDs from metadata.
	if err=(&AcademicRepository{db:tx}).CancelStaffSubstitution(ctx,actorPrincipalID,schoolID,substitutionID);err!=nil{t.Fatalf("cancel dated substitution through repository workflow: %v",err)}
	historyRepo:=NewUserRepository(tx)
	for _,teacherID:=range []int64{originalID,substituteID}{
		history,historyErr:=historyRepo.ListStaffAuditEvents(ctx,actorPrincipalID,schoolID,teacherID,StaffAuditEventFilter{Limit:100,Offset:0})
		if historyErr!=nil{t.Fatalf("list substitution history for teacher %d: %v",teacherID,historyErr)}
		foundCancellation:=false
		for _,event:=range history.Events{if event.Action=="staff.substitution.cancelled"{foundCancellation=true;break}}
		if !foundCancellation{t.Fatalf("staff history for teacher %d omitted cancellation resolved through substitution resource ID",teacherID)}
	}
	cancelledReason,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin cancelled history guard savepoint: %v",err)}
	_,cancelledReasonErr:=cancelledReason.Exec(ctx,`UPDATE staff_substitutions SET reason='Changed after cancellation' WHERE id=$1`,substitutionID)
	if rollbackErr:=cancelledReason.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback cancelled history guard savepoint: %v",rollbackErr)}
	if cancelledReasonErr==nil{t.Fatal("cancelled substitution history was mutable")}
	if _,err=tx.Exec(ctx,`UPDATE timetable_entries SET room='Changed room',period_id=$2 WHERE id=$1`,timetableEntryID,secondPeriodID);err!=nil{t.Fatalf("recurring timetable entry remained locked after cancellation: %v",err)}
	var snapshotRoom,snapshotClass,snapshotSection,snapshotSubject,snapshotPeriod,snapshotOriginal,snapshotSubstitute string
	if err=tx.QueryRow(ctx,`SELECT room_snapshot,class_name_snapshot,section_name_snapshot,subject_name_snapshot,period_name_snapshot,original_teacher_name_snapshot,substitute_teacher_name_snapshot
		FROM staff_substitutions WHERE id=$1`,substitutionID).Scan(&snapshotRoom,&snapshotClass,&snapshotSection,&snapshotSubject,&snapshotPeriod,&snapshotOriginal,&snapshotSubstitute);err!=nil{t.Fatalf("read immutable substitution snapshots: %v",err)}
	if snapshotRoom!="Original room"||snapshotClass==""||snapshotSection!="A"||snapshotSubject==""||snapshotPeriod!="Period 1"||snapshotOriginal!="Original Teacher"||snapshotSubstitute!="Substitute Teacher"{
		t.Fatalf("substitution snapshots did not preserve the original timetable context: room=%q class=%q section=%q subject=%q period=%q original=%q substitute=%q",snapshotRoom,snapshotClass,snapshotSection,snapshotSubject,snapshotPeriod,snapshotOriginal,snapshotSubstitute)
	}
	var retained int
	if err=tx.QueryRow(ctx,`SELECT count(*) FROM staff_substitutions WHERE id=$1 AND status='cancelled'`,substitutionID).Scan(&retained);err!=nil||retained!=1{t.Fatalf("cancelled substitution history was not retained: count=%d err=%v",retained,err)}
	var secondEntryID int64
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,schoolID,yearID,secondSectionID,subjectID,substituteID,secondAssignmentID,periodID,actorPrincipalID).Scan(&secondEntryID);err!=nil{t.Fatalf("teacher schedule should be allowed after substitution cancellation: %v",err)}
	var thirdPeriodID int64
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,3,'Period 3',TIME '11:00',TIME '11:40') RETURNING id`,schoolID,yearID).Scan(&thirdPeriodID);err!=nil{t.Fatalf("create third timetable period: %v",err)}
	var secondOriginalAssignmentID int64
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,secondSectionID,subjectID,originalID,actorPrincipalID).Scan(&secondOriginalAssignmentID);err!=nil{t.Fatalf("assign original teacher to second section: %v",err)}
	var pastEntryID int64
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,schoolID,yearID,secondSectionID,subjectID,originalID,secondOriginalAssignmentID,thirdPeriodID,actorPrincipalID).Scan(&pastEntryID);err!=nil{t.Fatalf("create original teacher's third-period timetable entry: %v",err)}
	var pastSubstitutionID int64
	err=tx.QueryRow(ctx,`INSERT INTO staff_substitutions(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,substitute_teacher_principal_id,reason,created_by_principal_id)
		VALUES($1,$2,$3,(date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date-7),$4,$5,'Completed cover history',$4) RETURNING id`,
		schoolID,pastEntryID,thirdPeriodID,originalID,substituteID).Scan(&pastSubstitutionID)
	if err!=nil{t.Fatalf("create past dated substitution history: %v",err)}
	if _,err=tx.Exec(ctx,`UPDATE timetable_entries SET room='Updated after completed cover' WHERE id=$1`,pastEntryID);err!=nil{t.Fatalf("completed cover history should not block future recurring timetable edits: %v",err)}
	pastDelete,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin past timetable history delete savepoint: %v",err)}
	_,pastDeleteErr:=pastDelete.Exec(ctx,`DELETE FROM timetable_entries WHERE id=$1`,pastEntryID)
	if rollbackErr:=pastDelete.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback past timetable history delete savepoint: %v",rollbackErr)}
	if pastDeleteErr==nil{t.Fatal("completed substitution history must still prevent hard deletion of its timetable entry")}
	if _,err=tx.Exec(ctx,`UPDATE auth_principals SET status='disabled' WHERE id=$1`,substituteID);err!=nil{t.Fatalf("disable canonical substitute principal after completed cover: %v",err)}
	canonicalStatus,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin canonical principal validation savepoint: %v",err)}
	_,canonicalStatusErr:=canonicalStatus.Exec(ctx,`INSERT INTO staff_substitutions(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,substitute_teacher_principal_id,reason,created_by_principal_id)
		VALUES($1,$2,$3,(date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date+7),$4,$5,'Disabled principal must not be assignable',$4)`,schoolID,pastEntryID,thirdPeriodID,originalID,substituteID)
	if rollbackErr:=canonicalStatus.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback canonical principal validation savepoint: %v",rollbackErr)}
	if canonicalStatusErr==nil{t.Fatal("disabled canonical auth principal was accepted as a substitute while legacy users.status remained active")}
}
