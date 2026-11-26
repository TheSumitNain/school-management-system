package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestStaffSubstitutionV61SerializesConcurrentSubstituteSlotAssignmentsPostgresIntegration
// starts two independent transactions assigning one substitute to different sections
// in the same weekday/period. Exactly one active cover may win.
func TestStaffSubstitutionV61SerializesConcurrentSubstituteSlotAssignmentsPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var schoolID, actorPrincipalID int64
	if err := pool.QueryRow(ctx, "SELECT school_id,principal_id FROM users WHERE id=$1", actorUserID).Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load integration actor: %v", err)
	}

	var schoolAdminRoleID int64
	if err:=pool.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`,schoolID).Scan(&schoolAdminRoleID);err!=nil { t.Fatalf("load School Admin role: %v",err) }

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	setupTx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin concurrent substitution fixture: %v", err) }
	defer func() { _ = setupTx.Rollback(context.Background()) }()

	var platformAdminPrincipalID int64
	err=setupTx.QueryRow(ctx,`SELECT ap.id FROM platform_admin_accounts pa
		JOIN auth_principals ap ON ap.id=pa.principal_id
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		JOIN role_permissions rp ON rp.role_id=r.id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE ap.principal_type='platform_admin' AND ap.status='active'
		  AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
		  AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
		LIMIT 1`).Scan(&platformAdminPrincipalID)
	if errors.Is(err,pgx.ErrNoRows) {
		var bootstrapExists bool
		if checkErr:=setupTx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)
			OR EXISTS(SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
				WHERE r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin')`).Scan(&bootstrapExists);checkErr!=nil { t.Fatalf("check Platform Admin bootstrap state: %v",checkErr) }
		if bootstrapExists { t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate") }
		platformAdminPrincipalID,err=NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx,setupTx,fmt.Sprintf("substitution-race-%d@example.invalid",time.Now().UnixNano()),"integration-test-password-hash")
	}
	if err!=nil { t.Fatalf("resolve Platform Admin actor: %v",err) }
	if _,err=setupTx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,actorPrincipalID,schoolAdminRoleID,platformAdminPrincipalID);err!=nil { t.Fatalf("grant School Admin role to race actor: %v",err) }

	var originalOne, originalTwo, substitute int64
	for i, target := range []*int64{&originalOne, &originalTwo, &substitute} {
		if err := setupTx.QueryRow(ctx, `INSERT INTO users(school_id,username,password_hash,status)
			VALUES($1,$2,'integration-test-password-hash','active') RETURNING principal_id`,
			schoolID, fmt.Sprintf("subrace-%s-%d", suffix, i)).Scan(target); err != nil {
			t.Fatalf("create race teacher %d: %v", i, err)
		}
	}
	for i, teacher := range []struct{id int64; name, code string}{
		{originalOne,"Race Original One","R1"},{originalTwo,"Race Original Two","R2"},{substitute,"Race Substitute","RS"},
	} {
		if _, err := setupTx.Exec(ctx, `INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number,staff_type)
			VALUES($1,$2,$3,$4,'Teacher','9876543210','teaching')`, teacher.id, schoolID, fmt.Sprintf("%s-%s-%d",teacher.code,suffix,i), teacher.name); err != nil {
			t.Fatalf("create race staff profile %d: %v", i, err)
		}
	}

	var teacherRoleID int64
	if err:=setupTx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		AND scope_type='school' AND role_key='teacher' AND is_system=TRUE`,schoolID).Scan(&teacherRoleID);err!=nil { t.Fatalf("load system Teacher role: %v",err) }
	for _,teacherID:=range []int64{originalOne,originalTwo,substitute} {
		if _,err:=setupTx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,teacherID,teacherRoleID,actorPrincipalID);err!=nil { t.Fatalf("assign Teacher role to race fixture %d: %v",teacherID,err) }
	}

	var yearID,classID,sectionOne,sectionTwo,subjectID,periodID int64
	if err := setupTx.QueryRow(ctx, `INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,CURRENT_DATE-30,CURRENT_DATE+365,'active') RETURNING id`,schoolID,"substitution-race-"+suffix).Scan(&yearID); err != nil { t.Fatalf("create race academic year: %v",err) }
	if err := setupTx.QueryRow(ctx, `INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,"Substitution Race "+suffix).Scan(&classID); err != nil { t.Fatalf("create race class: %v",err) }
	for i,target := range []*int64{&sectionOne,&sectionTwo} {
		if err := setupTx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name) VALUES($1,$2,$3,$4) RETURNING id`,schoolID,yearID,classID,fmt.Sprintf("S%d",i+1)).Scan(target); err != nil { t.Fatalf("create race section %d: %v",i,err) }
	}
	if err := setupTx.QueryRow(ctx,`INSERT INTO subjects(school_id,name) VALUES($1,$2) RETURNING id`,schoolID,"Substitution Race Subject "+suffix).Scan(&subjectID); err != nil { t.Fatalf("create race subject: %v",err) }
	if err := setupTx.QueryRow(ctx,`INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Race Period',TIME '09:00',TIME '09:40') RETURNING id`,schoolID,yearID).Scan(&periodID); err != nil { t.Fatalf("create race period: %v",err) }

	sections:=[]int64{sectionOne,sectionTwo}
	originals:=[]int64{originalOne,originalTwo}
	entries:=make([]int64,2)
	for i:=range sections {
		for _,teacherID:=range []int64{originals[i],substitute} {
			if _,err:=setupTx.Exec(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
				VALUES($1,$2,$3,$4,$5,$6)`,schoolID,yearID,sections[i],subjectID,teacherID,actorPrincipalID);err!=nil { t.Fatalf("create race teacher assignment: %v",err) }
		}
		var assignmentID int64
		if err:=setupTx.QueryRow(ctx,`SELECT id FROM teacher_subject_assignments WHERE school_id=$1 AND academic_year_id=$2 AND section_id=$3 AND subject_id=$4 AND teacher_principal_id=$5 AND revoked_at IS NULL`,
			schoolID,yearID,sections[i],subjectID,originals[i]).Scan(&assignmentID);err!=nil { t.Fatalf("read original teacher assignment: %v",err) }
		if err:=setupTx.QueryRow(ctx,`INSERT INTO timetable_entries(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,schoolID,yearID,sections[i],subjectID,originals[i],assignmentID,periodID,actorPrincipalID).Scan(&entries[i]);err!=nil { t.Fatalf("create original recurring timetable entry %d: %v",i,err) }
	}
	if err:=setupTx.Commit(ctx);err!=nil { t.Fatalf("commit concurrent substitution fixture: %v",err) }

	nowIST:=time.Now().In(time.FixedZone("IST",5*60*60+30*60))
	daysUntilMonday:=(int(time.Monday)-int(nowIST.Weekday())+7)%7
	if daysUntilMonday==0 { daysUntilMonday=7 }
	coverDate:=nowIST.AddDate(0,0,daysUntilMonday).Format("2006-01-02")

	// Hold the exact V61 advisory lock before launching either insert. Both
	// independent transactions must reach the real V61 trigger and wait there.
	lockKeyText:=fmt.Sprintf("%d:%d:%d:%d",schoolID,substitute,1,periodID)
	var barrierKey int64
	if err:=pool.QueryRow(ctx,"SELECT hashtextextended($1,0)",lockKeyText).Scan(&barrierKey);err!=nil { t.Fatalf("compute V61 slot lock key: %v",err) }
	barrierConn,err:=pool.Acquire(ctx);if err!=nil { t.Fatalf("acquire V61 lock holder connection: %v",err) };defer barrierConn.Release()
	if _,err:=barrierConn.Exec(ctx,"SELECT pg_advisory_lock($1)",barrierKey);err!=nil { t.Fatalf("hold V61 slot lock: %v",err) }
	barrierHeld:=true
	defer func(){if barrierHeld { _,_=barrierConn.Exec(context.Background(),"SELECT pg_advisory_unlock($1)",barrierKey) }}()

	type outcome struct{entryID int64;err error}
	results:=make(chan outcome,2)
	for i,entryID:=range entries {
		originalID:=originals[i]
		go func(id,original int64){
			_,insertErr:=(&AcademicRepository{db:pool}).CreateStaffSubstitution(ctx,actorPrincipalID,schoolID,id,original,substitute,coverDate,nil,"Concurrent substitute slot race")
			results<-outcome{entryID:id,err:insertErr}
		}(entryID,originalID)
	}

	deadline:=time.Now().Add(10*time.Second)
	for {
		var waiting int
		if err:=pool.QueryRow(ctx,`SELECT count(*) FROM pg_locks
			WHERE locktype='advisory' AND NOT granted AND objsubid=1
			AND classid=(($1::bigint >> 32) & 4294967295)::oid
			AND objid=($1::bigint & 4294967295)::oid`,barrierKey).Scan(&waiting);err!=nil { t.Fatalf("inspect V61 slot lock waiters: %v",err) }
		if waiting>=2 { break }
		if time.Now().After(deadline) { t.Fatalf("both inserts did not reach the V61 slot lock; waiting=%d",waiting) }
		time.Sleep(10*time.Millisecond)
	}
	if _,err:=barrierConn.Exec(ctx,"SELECT pg_advisory_unlock($1)",barrierKey);err!=nil { t.Fatalf("release V61 slot lock: %v",err) }
	barrierHeld=false

	successes,conflicts:=0,0
	var successfulEntryID, conflictingEntryID int64
	for i:=0;i<2;i++ {
		select {
		case result:=<-results:
			if result.err==nil { successes++; successfulEntryID=result.entryID } else if errors.Is(result.err,ErrStaffSubstitutionConflict) { conflicts++; conflictingEntryID=result.entryID } else { t.Fatalf("unexpected concurrent insert error for timetable entry %d: %v",result.entryID,result.err) }
		case <-ctx.Done(): t.Fatalf("timed out collecting concurrent substitution results: %v",ctx.Err())
		}
	}
	if successes!=1||conflicts!=1 { t.Fatalf("race results successes=%d conflicts=%d; want exactly one of each",successes,conflicts) }
	var activeCount int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM staff_substitutions WHERE school_id=$1 AND substitute_teacher_principal_id=$2 AND cover_date=$3::date AND period_id=$4 AND status='active'`,schoolID,substitute,coverDate,periodID).Scan(&activeCount);err!=nil { t.Fatalf("count active substitute assignments: %v",err) }
	if activeCount!=1 { t.Fatalf("race left %d active assignments; want 1",activeCount) }

	// Neither side of an active future substitution may be disabled. The
	// database guard is authoritative, and the repository returns a stable
	// domain error so the School Admin UI can explain the required action.
	userRepo := NewUserRepository(pool)
	for _, teacherID := range []int64{originalOne, substitute} {
		if err := userRepo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, teacherID); !errors.Is(err, ErrStaffProfileHasActiveSubstitutions) {
			t.Fatalf("disable teacher %d with active substitution error=%v; want ErrStaffProfileHasActiveSubstitutions", teacherID, err)
		}
	}
	var stillActive int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth_principals WHERE id=ANY($1::bigint[]) AND status='active'`, []int64{originalOne, substitute}).Scan(&stillActive); err != nil {
		t.Fatalf("verify both substitution teachers remain active: %v", err)
	}
	if stillActive != 2 { t.Fatalf("active substitution teachers remaining=%d; want 2", stillActive) }

	// Cancel the winning cover, then hold a canonical principal disable
	// uncommitted while the losing timetable slot is retried. The insertion
	// must wait for the same principal-row lock and reject the disabled
	// substitute after the disable transaction commits.
	var substitutionID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM staff_substitutions WHERE school_id=$1 AND timetable_entry_id=$2 AND status='active'`, schoolID, successfulEntryID).Scan(&substitutionID); err != nil {
		t.Fatalf("load winning substitution before lifecycle race: %v", err)
	}
	if err := (&AcademicRepository{db:pool}).CancelStaffSubstitution(ctx, actorPrincipalID, schoolID, substitutionID); err != nil {
		t.Fatalf("cancel winning substitution before disable race: %v", err)
	}
	disableTx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin uncommitted substitute disable: %v", err) }
	if _, err := disableTx.Exec(ctx, `UPDATE auth_principals SET status='disabled',token_version=token_version+1,updated_at=NOW() WHERE id=$1 AND status='active'`, substitute); err != nil {
		_ = disableTx.Rollback(ctx)
		t.Fatalf("disable substitute inside lifecycle race transaction: %v", err)
	}
	var conflictingOriginal int64
	if conflictingEntryID == entries[0] { conflictingOriginal = originalOne } else { conflictingOriginal = originalTwo }
	retryResult := make(chan error, 1)
	go func() {
		_, createErr := (&AcademicRepository{db:pool}).CreateStaffSubstitution(ctx, actorPrincipalID, schoolID, conflictingEntryID, conflictingOriginal, substitute, coverDate, nil, "Substitution racing account disable")
		retryResult <- createErr
	}()
	deadline = time.Now().Add(10 * time.Second)
	for {
		var waiting int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query ILIKE '%INSERT INTO staff_substitutions%'`).Scan(&waiting); err != nil {
			_ = disableTx.Rollback(ctx)
			t.Fatalf("inspect substitution insert waiting on principal lock: %v", err)
		}
		if waiting > 0 { break }
		select {
		case err := <-retryResult:
			_ = disableTx.Rollback(ctx)
			t.Fatalf("substitution creation completed before the in-flight disable committed: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			_ = disableTx.Rollback(ctx)
			t.Fatal("substitution insert never waited on the principal lock held by the disable transaction")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := disableTx.Commit(ctx); err != nil { t.Fatalf("commit substitute disable during race: %v", err) }
	select {
	case err := <-retryResult:
		if !errors.Is(err, ErrInvalidStaffSubstitution) {
			t.Fatalf("substitution after concurrent substitute disable error=%v; want ErrInvalidStaffSubstitution", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("substitution insert did not finish after substitute disable committed")
	}
}
