package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrTimetableAccessDenied = errors.New("timetable operation is not authorized")
	ErrTimetableNotFound = errors.New("timetable record not found")
	ErrTimetableConflict = errors.New("timetable slot conflicts with an existing entry")
	ErrInvalidTimetableInput = errors.New("invalid timetable input")
)

type TimetablePeriod struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	AcademicYearID int64 `json:"academic_year_id"`
	PeriodNumber int `json:"period_number"`
	Name string `json:"name"`
	StartsAt string `json:"starts_at"`
	EndsAt string `json:"ends_at"`
	IsBreak bool `json:"is_break"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TimetableEntry struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	AcademicYearID int64 `json:"academic_year_id"`
	SectionID int64 `json:"section_id"`
	SectionName string `json:"section_name"`
	ClassName string `json:"class_name"`
	SubjectID int64 `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	TeacherPrincipalID int64 `json:"teacher_principal_id"`
	TeacherUsername string `json:"teacher_username"`
	TeacherSubjectAssignmentID int64 `json:"teacher_subject_assignment_id"`
	PeriodID int64 `json:"period_id"`
	PeriodNumber int `json:"period_number"`
	PeriodName string `json:"period_name"`
	StartsAt string `json:"starts_at"`
	EndsAt string `json:"ends_at"`
	DayOfWeek int `json:"day_of_week"`
	Room *string `json:"room,omitempty"`
	CreatedByPrincipalID int64 `json:"created_by_principal_id"`
	UpdatedByPrincipalID int64 `json:"updated_by_principal_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func timetableWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return ErrTimetableConflict
		case "23503", "23514", "22001", "22007", "P0001":
			return ErrInvalidTimetableInput
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTimetableNotFound
	}
	return err
}

func timetableAuthorize(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if err := authorizeAcademic(ctx, db, actorID, schoolID, permission); err != nil {
		if errors.Is(err, ErrAcademicAccessDenied) { return ErrTimetableAccessDenied }
		return err
	}
	return nil
}

func (r *AcademicRepository) GetCurrentTimetableYear(ctx context.Context, actorID, schoolID int64) (*AcademicYear, error) {
	if r == nil || r.db == nil { return nil, ErrTimetableAccessDenied }
	if err := timetableAuthorize(ctx, r.db, actorID, schoolID, "timetable.read"); err != nil { return nil, err }
	var year AcademicYear
	err := r.db.QueryRow(ctx, `SELECT id,school_id,name,starts_on,ends_on,status,is_current,created_at,updated_at
		FROM academic_years WHERE school_id=$1 AND is_current=TRUE AND status='active'`, schoolID).Scan(
		&year.ID,&year.SchoolID,&year.Name,&year.StartsOn,&year.EndsOn,&year.Status,&year.IsCurrent,&year.CreatedAt,&year.UpdatedAt)
	if err != nil { return nil, timetableWriteError(err) }
	return &year,nil
}

func (r *AcademicRepository) ListTimetablePeriods(ctx context.Context, actorID, schoolID, yearID int64) ([]TimetablePeriod, error) {
	if r == nil || r.db == nil || yearID <= 0 {
		return nil, ErrInvalidTimetableInput
	}
	if err := timetableAuthorize(ctx, r.db, actorID, schoolID, "timetable.read"); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,school_id,academic_year_id,period_number,name,
		to_char(starts_at,'HH24:MI'),to_char(ends_at,'HH24:MI'),is_break,created_at,updated_at
		FROM timetable_periods WHERE school_id=$1 AND academic_year_id=$2
		ORDER BY period_number,id`, schoolID, yearID)
	if err != nil { return nil, fmt.Errorf("list timetable periods: %w", err) }
	defer rows.Close()
	items := make([]TimetablePeriod, 0)
	for rows.Next() {
		var v TimetablePeriod
		if err := rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.PeriodNumber,&v.Name,&v.StartsAt,&v.EndsAt,&v.IsBreak,&v.CreatedAt,&v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan timetable period: %w", err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate timetable periods: %w", err) }
	return items, nil
}

func (r *AcademicRepository) SaveTimetablePeriod(ctx context.Context, actorID, schoolID, id, yearID int64, number int, name, startsAt, endsAt string, isBreak bool) (*TimetablePeriod, error) {
	if r == nil || r.db == nil { return nil, ErrTimetableAccessDenied }
	if err := timetableAuthorize(ctx, r.db, actorID, schoolID, "timetable.manage"); err != nil { return nil, err }
	name = strings.TrimSpace(name)
	if yearID <= 0 || number <= 0 || name == "" || len([]rune(name)) > 80 { return nil, ErrInvalidTimetableInput }

	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin timetable period save: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var v TimetablePeriod
	if id == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO timetable_periods
			(school_id,academic_year_id,period_number,name,starts_at,ends_at,is_break)
			VALUES($1,$2,$3,$4,$5::time,$6::time,$7)
			RETURNING id,school_id,academic_year_id,period_number,name,to_char(starts_at,'HH24:MI'),to_char(ends_at,'HH24:MI'),is_break,created_at,updated_at`,
			schoolID,yearID,number,name,startsAt,endsAt,isBreak).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.PeriodNumber,&v.Name,&v.StartsAt,&v.EndsAt,&v.IsBreak,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err = tx.QueryRow(ctx, `UPDATE timetable_periods SET period_number=$4,name=$5,starts_at=$6::time,ends_at=$7::time,is_break=$8,updated_at=NOW()
			WHERE id=$1 AND school_id=$2 AND academic_year_id=$3
			RETURNING id,school_id,academic_year_id,period_number,name,to_char(starts_at,'HH24:MI'),to_char(ends_at,'HH24:MI'),is_break,created_at,updated_at`,
			id,schoolID,yearID,number,name,startsAt,endsAt,isBreak).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.PeriodNumber,&v.Name,&v.StartsAt,&v.EndsAt,&v.IsBreak,&v.CreatedAt,&v.UpdatedAt)
	}
	if err != nil { return nil, timetableWriteError(err) }
	action := "school.timetable.period.created"
	if id != 0 { action = "school.timetable.period.updated" }
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'timetable_period',$4,'success',jsonb_build_object('academic_year_id',$5::bigint,'period_number',$6::integer))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),yearID,number); err != nil { return nil, fmt.Errorf("audit timetable period: %w", err) }
	if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit timetable period: %w", err) }
	return &v, nil
}

const timetableEntryColumns = `te.id,te.school_id,te.academic_year_id,te.section_id,s.name,c.name,
	te.subject_id,sub.name,te.teacher_principal_id,u.username,te.teacher_subject_assignment_id,
	te.period_id,p.period_number,p.name,to_char(p.starts_at,'HH24:MI'),to_char(p.ends_at,'HH24:MI'),
	te.day_of_week,te.room,te.created_by_principal_id,te.updated_by_principal_id,te.created_at,te.updated_at`

func (r *AcademicRepository) ListTimetableEntries(ctx context.Context, actorID, schoolID, yearID, sectionID int64) ([]TimetableEntry, error) {
	if r == nil || r.db == nil || yearID <= 0 || sectionID < 0 { return nil, ErrInvalidTimetableInput }
	if err := timetableAuthorize(ctx, r.db, actorID, schoolID, "timetable.read"); err != nil { return nil, err }
	teacherOnly, err := isTeacherWithoutSchoolAdmin(ctx, r.db, actorID, schoolID)
	if err != nil { return nil, fmt.Errorf("check timetable teacher boundary: %w", err) }
	rows, err := r.db.Query(ctx, `SELECT `+timetableEntryColumns+`
		FROM timetable_entries te
		JOIN sections s ON s.id=te.section_id AND s.school_id=te.school_id AND s.academic_year_id=te.academic_year_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN subjects sub ON sub.id=te.subject_id AND sub.school_id=te.school_id
		JOIN users u ON u.principal_id=te.teacher_principal_id AND u.school_id=te.school_id
		JOIN timetable_periods p ON p.id=te.period_id AND p.school_id=te.school_id AND p.academic_year_id=te.academic_year_id
		WHERE te.school_id=$1 AND te.academic_year_id=$2 AND ($3=0 OR te.section_id=$3)
		  AND (NOT $4 OR te.teacher_principal_id=$5)
		ORDER BY te.day_of_week,p.period_number,s.display_order,s.name`,schoolID,yearID,sectionID,teacherOnly,actorID)
	if err != nil { return nil, fmt.Errorf("list timetable entries: %w", err) }
	defer rows.Close()
	items := make([]TimetableEntry,0)
	for rows.Next() {
		var v TimetableEntry
		if err := rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.SectionID,&v.SectionName,&v.ClassName,
			&v.SubjectID,&v.SubjectName,&v.TeacherPrincipalID,&v.TeacherUsername,&v.TeacherSubjectAssignmentID,
			&v.PeriodID,&v.PeriodNumber,&v.PeriodName,&v.StartsAt,&v.EndsAt,&v.DayOfWeek,&v.Room,
			&v.CreatedByPrincipalID,&v.UpdatedByPrincipalID,&v.CreatedAt,&v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan timetable entry: %w", err)
		}
		items=append(items,v)
	}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate timetable entries: %w",err)}
	return items,nil
}

func (r *AcademicRepository) SaveTimetableEntry(ctx context.Context, actorID, schoolID, id, yearID, sectionID, subjectID, teacherID, assignmentID, periodID int64, day int, room *string) (*TimetableEntry,error) {
	if r==nil||r.db==nil{return nil,ErrTimetableAccessDenied}
	if err:=timetableAuthorize(ctx,r.db,actorID,schoolID,"timetable.manage");err!=nil{return nil,err}
	if yearID<=0||sectionID<=0||subjectID<=0||teacherID<=0||assignmentID<=0||periodID<=0||day<1||day>7{return nil,ErrInvalidTimetableInput}
	if room!=nil { v:=strings.TrimSpace(*room);if len([]rune(v))>80{return nil,ErrInvalidTimetableInput};if v==""{room=nil}else{room=&v} }

	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin timetable entry save: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	var v TimetableEntry
	if id==0 {
		err=tx.QueryRow(ctx,`INSERT INTO timetable_entries
			(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,teacher_subject_assignment_id,period_id,day_of_week,room,created_by_principal_id,updated_by_principal_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10) RETURNING id`,
			schoolID,yearID,sectionID,subjectID,teacherID,assignmentID,periodID,day,room,actorID).Scan(&v.ID)
	} else {
		err=tx.QueryRow(ctx,`UPDATE timetable_entries SET section_id=$3,subject_id=$4,teacher_principal_id=$5,teacher_subject_assignment_id=$6,period_id=$7,day_of_week=$8,room=$9,updated_by_principal_id=$10,updated_at=NOW()
			WHERE id=$1 AND school_id=$2 AND academic_year_id=$11 RETURNING id`,
			id,schoolID,sectionID,subjectID,teacherID,assignmentID,periodID,day,room,actorID,yearID).Scan(&v.ID)
	}
	if err!=nil{return nil,timetableWriteError(err)}
	action:="school.timetable.entry.created";if id!=0{action="school.timetable.entry.updated"}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'timetable_entry',$4,'success',jsonb_build_object('academic_year_id',$5::bigint,'section_id',$6::bigint,'day_of_week',$7::integer,'period_id',$8::bigint))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),yearID,sectionID,day,periodID);err!=nil{return nil,fmt.Errorf("audit timetable entry: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit timetable entry: %w",err)}
	return r.getTimetableEntry(ctx,schoolID,yearID,v.ID)
}

func (r *AcademicRepository) DeleteTimetableEntry(ctx context.Context,actorID,schoolID,yearID,id int64)error{
	if r==nil||r.db==nil{return ErrTimetableAccessDenied}
	if err:=timetableAuthorize(ctx,r.db,actorID,schoolID,"timetable.manage");err!=nil{return err}
	tx,err:=r.db.Begin(ctx);if err!=nil{return fmt.Errorf("begin timetable entry deletion: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	tag,err:=tx.Exec(ctx,`DELETE FROM timetable_entries WHERE id=$1 AND school_id=$2 AND academic_year_id=$3`,id,schoolID,yearID)
	if err!=nil{return fmt.Errorf("delete timetable entry: %w",err)}
	if tag.RowsAffected()==0{return ErrTimetableNotFound}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome)
		VALUES($1,$2,'school.timetable.entry.deleted','timetable_entry',$3,'success')`,actorID,schoolID,fmt.Sprint(id));err!=nil{return fmt.Errorf("audit timetable entry deletion: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit timetable entry deletion: %w",err)}
	return nil
}

func (r *AcademicRepository) getTimetableEntry(ctx context.Context,schoolID,yearID,id int64)(*TimetableEntry,error){
	var v TimetableEntry
	err:=r.db.QueryRow(ctx,`SELECT `+timetableEntryColumns+`
		FROM timetable_entries te
		JOIN sections s ON s.id=te.section_id AND s.school_id=te.school_id AND s.academic_year_id=te.academic_year_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN subjects sub ON sub.id=te.subject_id AND sub.school_id=te.school_id
		JOIN users u ON u.principal_id=te.teacher_principal_id AND u.school_id=te.school_id
		JOIN timetable_periods p ON p.id=te.period_id AND p.school_id=te.school_id AND p.academic_year_id=te.academic_year_id
		WHERE te.school_id=$1 AND te.academic_year_id=$2 AND te.id=$3`,schoolID,yearID,id).Scan(
		&v.ID,&v.SchoolID,&v.AcademicYearID,&v.SectionID,&v.SectionName,&v.ClassName,&v.SubjectID,&v.SubjectName,
		&v.TeacherPrincipalID,&v.TeacherUsername,&v.TeacherSubjectAssignmentID,&v.PeriodID,&v.PeriodNumber,&v.PeriodName,
		&v.StartsAt,&v.EndsAt,&v.DayOfWeek,&v.Room,&v.CreatedByPrincipalID,&v.UpdatedByPrincipalID,&v.CreatedAt,&v.UpdatedAt)
	if err!=nil{return nil,timetableWriteError(err)}
	return &v,nil
}
