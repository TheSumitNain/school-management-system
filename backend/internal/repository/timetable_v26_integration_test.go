package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestTimetableV26SchemaPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ready bool
	err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.timetable_periods') IS NOT NULL
		   AND to_regclass('public.timetable_entries') IS NOT NULL
		   AND to_regclass('public.ux_timetable_entries_section_slot') IS NOT NULL
		   AND to_regclass('public.ux_timetable_entries_teacher_slot') IS NOT NULL
		   AND EXISTS (
		       SELECT 1 FROM pg_trigger
		        WHERE tgname = 'trg_validate_v26_timetable_entry'
		          AND NOT tgisinternal
		   )
		   AND EXISTS (
		       SELECT 1 FROM pg_trigger
		        WHERE tgname = 'trg_guard_v26_teacher_assignment_schedule'
		          AND NOT tgisinternal
		   )
		   AND EXISTS (
		       SELECT 1 FROM pg_constraint
		        WHERE conname = 'timetable_entries_assignment_scope_fkey'
		   )
	`).Scan(&ready)
	if err != nil {
		t.Fatalf("inspect V26 timetable schema: %v", err)
	}
	if !ready {
		t.Fatal("V26 timetable tables, conflict indexes, assignment scope, or validation triggers are missing")
	}
}

func TestTimetablePeriodOverlapRejectedPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin rollback-scoped timetable test: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load school fixture: %v", err)
	}
	suffix := time.Now().UnixNano()
	var yearID int64
	if err := tx.QueryRow(ctx, `INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active') RETURNING id`,
		schoolID, fmt.Sprintf("timetable-overlap-%d", suffix)).Scan(&yearID); err != nil {
		t.Fatalf("create academic year fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Period 1',TIME '09:00',TIME '09:40')`, schoolID, yearID); err != nil {
		t.Fatalf("create first timetable period: %v", err)
	}
	savepoint, err := tx.Begin(ctx)
	if err != nil { t.Fatalf("begin overlap savepoint: %v", err) }
	_, overlapErr := tx.Exec(ctx, `INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,2,'Period 2',TIME '09:30',TIME '10:10')`, schoolID, yearID)
	if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil {
		t.Fatalf("rollback overlap savepoint: %v", rollbackErr)
	}
	if overlapErr == nil {
		t.Fatal("overlapping timetable periods were accepted")
	}
}

func TestTeacherWorkloadReportPostgresIntegration(t *testing.T) {
	_,pool,userID:=newPostgresIntegrationRepository(t)
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
	defer cancel()
	tx,err:=pool.Begin(ctx);if err!=nil{t.Fatalf("begin workload report test: %v",err)}
	defer func(){_=tx.Rollback(ctx)}()

	var schoolID,actorID int64
	if err=tx.QueryRow(ctx,`SELECT school_id,principal_id FROM users WHERE id=$1`,userID).Scan(&schoolID,&actorID);err!=nil{t.Fatalf("load workload actor: %v",err)}
	var schoolAdminRoleID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff' AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`,schoolID).Scan(&schoolAdminRoleID);err!=nil{t.Fatalf("load School Admin role: %v",err)}
	var platformAdminID int64
	err=tx.QueryRow(ctx,`SELECT ap.id FROM platform_admin_accounts pa
		JOIN auth_principals ap ON ap.id=pa.principal_id
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		JOIN role_permissions rp ON rp.role_id=r.id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE ap.principal_type='platform_admin' AND ap.status='active' AND r.role_key='platform_admin'
		  AND r.scope_type='platform' AND r.is_system=TRUE AND p.permission_key='platform.permissions.delegate'
		  AND p.scope_type='platform' AND p.is_active=TRUE LIMIT 1`).Scan(&platformAdminID)
	if errors.Is(err,pgx.ErrNoRows){
		var exists bool
		if checkErr:=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)`).Scan(&exists);checkErr!=nil{t.Fatalf("check Platform Admin state: %v",checkErr)}
		if exists{t.Skip("no active Platform Admin with delegation permission available for workload fixture")}
		platformAdminID,err=NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx,tx,fmt.Sprintf("workload-%d@example.invalid",time.Now().UnixNano()),"integration-test-password-hash")
	}
	if err!=nil{t.Fatalf("resolve Platform Admin actor: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,actorID,schoolAdminRoleID,platformAdminID);err!=nil{t.Fatalf("grant School Admin role to workload actor: %v",err)}

	suffix:=time.Now().UnixNano()
	var teacherID int64
	if err=tx.QueryRow(ctx,`INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,'integration-test-password-hash','active') RETURNING principal_id`,schoolID,fmt.Sprintf("workload-teacher-%d",suffix)).Scan(&teacherID);err!=nil{t.Fatalf("create workload teacher: %v",err)}
	var teacherRoleID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff' AND scope_type='school' AND role_key='teacher' AND is_system=TRUE`,schoolID).Scan(&teacherRoleID);err!=nil{t.Fatalf("load Teacher role: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,teacherID,teacherRoleID,actorID);err!=nil{t.Fatalf("assign Teacher role: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number)
		VALUES($1,$2,$3,'Workload Teacher','Teacher','9876543218')`,teacherID,schoolID,fmt.Sprintf("WL-%d",suffix));err!=nil{t.Fatalf("create workload teacher profile: %v",err)}

	var yearID,classID,sectionID,subjectID,periodID,assignmentID int64
	if err=tx.QueryRow(ctx,`INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active') RETURNING id`,schoolID,fmt.Sprintf("workload-year-%d",suffix)).Scan(&yearID);err!=nil{t.Fatalf("create workload year: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,fmt.Sprintf("Workload Class %d",suffix)).Scan(&classID);err!=nil{t.Fatalf("create workload class: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name) VALUES($1,$2,$3,'A') RETURNING id`,schoolID,yearID,classID).Scan(&sectionID);err!=nil{t.Fatalf("create workload section: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO subjects(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,fmt.Sprintf("Workload Subject %d",suffix)).Scan(&subjectID);err!=nil{t.Fatalf("create workload subject: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Period 1',TIME '09:00',TIME '09:40') RETURNING id`,schoolID,yearID).Scan(&periodID);err!=nil{t.Fatalf("create workload period: %v",err)}
	if err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,sectionID,subjectID,teacherID,actorID).Scan(&assignmentID);err!=nil{t.Fatalf("create workload assignment: %v",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8)`,schoolID,yearID,sectionID,subjectID,teacherID,assignmentID,periodID,actorID);err!=nil{t.Fatalf("create workload timetable entry: %v",err)}

	report,err:=NewAcademicRepository(tx).ListTeacherWorkloadReport(ctx,actorID,schoolID,yearID)
	if err!=nil{t.Fatalf("list teacher workload report: %v",err)}
	if len(report)!=1{t.Fatalf("expected one workload row, got %d: %+v",len(report),report)}
	row:=report[0]
	if row.TeacherPrincipalID!=teacherID||row.TeacherName!="Workload Teacher"||row.DayOfWeek!=1||row.ScheduledPeriods!=1||row.DistinctSections!=1||row.DistinctSubjects!=1||row.FirstPeriod==nil||*row.FirstPeriod!=1||row.LastPeriod==nil||*row.LastPeriod!=1{
		t.Fatalf("unexpected workload report row: %+v",row)
	}
}

