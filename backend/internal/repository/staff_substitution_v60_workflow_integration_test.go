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
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$5) RETURNING id`,schoolID,yearID,sectionID,subjectID,originalID).Scan(&originalAssignmentID);err!=nil{t.Fatalf("assign original teacher: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,sectionID,subjectID,substituteID,originalID).Scan(&substituteAssignmentID);err!=nil{t.Fatalf("assign substitute teacher: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,schoolID,yearID,sectionID,subjectID,originalID,originalAssignmentID,periodID,originalID).Scan(&timetableEntryID);err!=nil{t.Fatalf("create recurring timetable entry: %v",err)}

	const coverDate="2026-11-23" // Monday; same weekday as the recurring entry.
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
	_,disabledErr:=disabled.Exec(ctx,`UPDATE users SET status='disabled' WHERE principal_id=$1`,substituteID)
	if rollbackErr:=disabled.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback substitute login savepoint: %v",rollbackErr)}
	if disabledErr==nil{t.Fatal("substitute login was disabled while future coverage remained active")}

	update,err:=tx.Begin(ctx);if err!=nil{t.Fatalf("begin recurring schedule guard savepoint: %v",err)}
	_,updateErr:=update.Exec(ctx,`UPDATE timetable_entries SET room='Changed room' WHERE id=$1`,timetableEntryID)
	if rollbackErr:=update.Rollback(ctx);rollbackErr!=nil{t.Fatalf("rollback schedule guard savepoint: %v",rollbackErr)}
	if updateErr==nil{t.Fatal("recurring timetable entry changed while an active dated substitution referenced it")}

	if _,err=tx.Exec(ctx,`UPDATE staff_substitutions SET status='cancelled',cancelled_by_principal_id=$2,cancelled_at=NOW(),updated_at=NOW() WHERE id=$1`,substitutionID,originalID);err!=nil{t.Fatalf("cancel dated substitution: %v",err)}
	if _,err=tx.Exec(ctx,`UPDATE timetable_entries SET room='Changed room' WHERE id=$1`,timetableEntryID);err!=nil{t.Fatalf("recurring timetable entry remained locked after cancellation: %v",err)}

	var secondSectionID int64
	if err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name) VALUES($1,$2,$3,'B') RETURNING id`,schoolID,yearID,classID).Scan(&secondSectionID);err!=nil{t.Fatalf("create second section: %v",err)}
	var secondAssignmentID int64
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,secondSectionID,subjectID,substituteID,originalID).Scan(&secondAssignmentID);err!=nil{t.Fatalf("assign substitute to second section: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8)`,schoolID,yearID,secondSectionID,subjectID,substituteID,secondAssignmentID,periodID,originalID);err!=nil{t.Fatalf("teacher schedule should be allowed after substitution cancellation: %v",err)}
}
