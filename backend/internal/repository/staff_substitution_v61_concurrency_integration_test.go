package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	setupTx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin concurrent substitution fixture: %v", err) }
	defer func() { _ = setupTx.Rollback(context.Background()) }()

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
		if _, err := setupTx.Exec(ctx, `INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number)
			VALUES($1,$2,$3,$4,'Teacher','9876543210')`, teacher.id, schoolID, fmt.Sprintf("%s-%s-%d",teacher.code,suffix,i), teacher.name); err != nil {
			t.Fatalf("create race staff profile %d: %v", i, err)
		}
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

	barrierKey:=time.Now().UnixNano()
	barrierSuffix:=suffix[len(suffix)-8:]
	functionName:="test_substitution_race_barrier_fn_"+barrierSuffix
	triggerName:="aaa_test_substitution_race_barrier_"+barrierSuffix
	if _,err:=pool.Exec(ctx,fmt.Sprintf("CREATE FUNCTION public.%s() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN PERFORM pg_advisory_xact_lock(%d); RETURN NEW; END;'",functionName,barrierKey));err!=nil { t.Fatalf("create deterministic race barrier function: %v",err) }
	if _,err:=pool.Exec(ctx,fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON public.staff_substitutions FOR EACH ROW EXECUTE FUNCTION public.%s()",triggerName,functionName));err!=nil {
		_,_=pool.Exec(context.Background(),fmt.Sprintf("DROP FUNCTION IF EXISTS public.%s()",functionName))
		t.Fatalf("create deterministic race barrier trigger: %v",err)
	}
	defer func(){ cleanupCtx,cleanupCancel:=context.WithTimeout(context.Background(),5*time.Second);defer cleanupCancel();_,_=pool.Exec(cleanupCtx,fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON public.staff_substitutions",triggerName));_,_=pool.Exec(cleanupCtx,fmt.Sprintf("DROP FUNCTION IF EXISTS public.%s()",functionName)) }()

	barrierConn,err:=pool.Acquire(ctx);if err!=nil { t.Fatalf("acquire race barrier connection: %v",err) };defer barrierConn.Release()
	if _,err:=barrierConn.Exec(ctx,"SELECT pg_advisory_lock($1)",barrierKey);err!=nil { t.Fatalf("hold deterministic race barrier: %v",err) }
	barrierHeld:=true
	defer func(){if barrierHeld { _,_=barrierConn.Exec(context.Background(),"SELECT pg_advisory_unlock($1)",barrierKey) }}()

	type outcome struct{entryID int64;err error}
	results:=make(chan outcome,2)
	for i,entryID:=range entries {
		originalID:=originals[i]
		go func(id,original int64){
			_,insertErr:=pool.Exec(ctx,`INSERT INTO staff_substitutions(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,substitute_teacher_principal_id,reason,created_by_principal_id)
				VALUES($1,$2,$3,$4::date,$5,$6,'Concurrent substitute slot race',$7)`,schoolID,id,periodID,coverDate,original,substitute,actorPrincipalID)
			results<-outcome{entryID:id,err:insertErr}
		}(entryID,originalID)
	}

	deadline:=time.Now().Add(10*time.Second)
	for {
		var waiting int
		if err:=pool.QueryRow(ctx,"SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted").Scan(&waiting);err!=nil { t.Fatalf("inspect concurrent substitution lock waiters: %v",err) }
		if waiting>=2 { break }
		if time.Now().After(deadline) { t.Fatalf("both assignments did not reach race barrier; waiting=%d",waiting) }
		time.Sleep(10*time.Millisecond)
	}
	if _,err:=barrierConn.Exec(ctx,"SELECT pg_advisory_unlock($1)",barrierKey);err!=nil { t.Fatalf("release deterministic race barrier: %v",err) }
	barrierHeld=false

	successes,conflicts:=0,0
	for i:=0;i<2;i++ {
		select {
		case result:=<-results:
			if result.err==nil { successes++ } else if strings.Contains(result.err.Error(),"ux_staff_substitutions_substitute_period_active") { conflicts++ } else { t.Fatalf("unexpected concurrent insert error for timetable entry %d: %v",result.entryID,result.err) }
		case <-ctx.Done(): t.Fatalf("timed out collecting concurrent substitution results: %v",ctx.Err())
		}
	}
	if successes!=1||conflicts!=1 { t.Fatalf("race results successes=%d conflicts=%d; want exactly one of each",successes,conflicts) }
	var activeCount int
	if err:=pool.QueryRow(ctx,`SELECT count(*) FROM staff_substitutions WHERE school_id=$1 AND substitute_teacher_principal_id=$2 AND cover_date=$3::date AND period_id=$4 AND status='active'`,schoolID,substitute,coverDate,periodID).Scan(&activeCount);err!=nil { t.Fatalf("count active substitute assignments: %v",err) }
	if activeCount!=1 { t.Fatalf("race left %d active assignments; want 1",activeCount) }
}
