package repository

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"

    "school-management-system/internal/schooltime"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

var (
    ErrStaffAttendanceAccessDenied = errors.New("staff attendance access denied")
    ErrStaffAttendanceNotFound = errors.New("staff attendance record not found")
    ErrStaffAttendanceConflict = errors.New("staff attendance already exists for this employee and date")
    ErrInvalidStaffAttendance = errors.New("invalid staff attendance data")
)

type StaffAttendanceRecord struct {
    ID *int64 `json:"id"`
    SchoolID int64 `json:"school_id"`
    StaffPrincipalID int64 `json:"staff_principal_id"`
    EmployeeCode string `json:"employee_code"`
    FullName string `json:"full_name"`
    Designation string `json:"designation"`
    AttendanceDate string `json:"attendance_date"`
    Status *string `json:"status,omitempty"`
    Remarks *string `json:"remarks,omitempty"`
    MarkedByUsername *string `json:"marked_by_username,omitempty"`
    UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type StaffAttendanceMonthlySummary struct {
    StaffPrincipalID int64 `json:"staff_principal_id"`
    EmployeeCode string `json:"employee_code"`
    FullName string `json:"full_name"`
    Designation string `json:"designation"`
    PresentDays int `json:"present_days"`
    AbsentDays int `json:"absent_days"`
    LateDays int `json:"late_days"`
    HalfDays int `json:"half_days"`
    Records json.RawMessage `json:"records"`
}

type StaffAttendanceRepository struct { db repositoryDB }

func NewStaffAttendanceRepository(db repositoryDB) *StaffAttendanceRepository {
    return &StaffAttendanceRepository{db: db}
}

func validStaffAttendanceStatus(status string) bool {
    switch status {
    case "present", "absent", "late", "half_day":
        return true
    default:
        return false
    }
}

func staffAttendanceWriteError(err error) error {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505":
            return ErrStaffAttendanceConflict
        case "23503", "23514", "22001", "22007":
            return ErrInvalidStaffAttendance
        }
    }
    if errors.Is(err, pgx.ErrNoRows) { return ErrStaffAttendanceNotFound }
    return err
}

func (r *StaffAttendanceRepository) ListDaily(ctx context.Context, actorID, schoolID int64, day time.Time) ([]StaffAttendanceRecord, error) {
    if r == nil || r.db == nil || day.Year() < 1 { return nil, ErrInvalidStaffAttendance }
    if err := authorizeGuardian(ctx, r.db, actorID, schoolID, "staff.attendance.read"); err != nil {
        return nil, ErrStaffAttendanceAccessDenied
    }
    rows, err := r.db.Query(ctx, `
        SELECT a.id, sp.principal_id, sp.employee_code, sp.full_name, sp.designation,
               $2::date::text, a.status, a.remarks, marker.username, a.updated_at
        FROM staff_profiles sp
        LEFT JOIN staff_attendance_records a
          ON a.school_id = sp.school_id
         AND a.staff_principal_id = sp.principal_id
         AND a.attendance_date = $2::date
        LEFT JOIN users marker
          ON marker.principal_id = a.marked_by_principal_id
         AND marker.school_id = a.school_id
        WHERE sp.school_id = $1
          AND sp.employment_status = 'active'
        ORDER BY LOWER(sp.full_name), sp.employee_code, sp.principal_id
        LIMIT 1000
    `, schoolID, day.Format("2006-01-02"))
    if err != nil { return nil, fmt.Errorf("list daily staff attendance: %w", err) }
    defer rows.Close()
    items := make([]StaffAttendanceRecord, 0)
    for rows.Next() {
        var item StaffAttendanceRecord
        if err := rows.Scan(&item.ID, &item.StaffPrincipalID, &item.EmployeeCode, &item.FullName, &item.Designation,
            &item.AttendanceDate, &item.Status, &item.Remarks, &item.MarkedByUsername, &item.UpdatedAt); err != nil {
            return nil, fmt.Errorf("scan daily staff attendance: %w", err)
        }
        item.SchoolID = schoolID
        items = append(items, item)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate daily staff attendance: %w", err) }
    return items, nil
}

func (r *StaffAttendanceRepository) ListMonthly(ctx context.Context, actorID, schoolID int64, month time.Time) ([]StaffAttendanceMonthlySummary, error) {
    if r == nil || r.db == nil || month.Year() < 1 { return nil, ErrInvalidStaffAttendance }
    if err := authorizeGuardian(ctx, r.db, actorID, schoolID, "staff.attendance.read"); err != nil {
        return nil, ErrStaffAttendanceAccessDenied
    }
    monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
    monthEnd := monthStart.AddDate(0, 1, 0)
    rows, err := r.db.Query(ctx, `
        SELECT sp.principal_id, sp.employee_code, sp.full_name, sp.designation,
               COUNT(a.id) FILTER (WHERE a.status = 'present')::integer,
               COUNT(a.id) FILTER (WHERE a.status = 'absent')::integer,
               COUNT(a.id) FILTER (WHERE a.status = 'late')::integer,
               COUNT(a.id) FILTER (WHERE a.status = 'half_day')::integer,
               COALESCE(jsonb_agg(jsonb_build_object(
                   'id', a.id, 'date', a.attendance_date, 'status', a.status, 'remarks', a.remarks
               ) ORDER BY a.attendance_date) FILTER (WHERE a.id IS NOT NULL), '[]'::jsonb)
        FROM staff_profiles sp
        LEFT JOIN staff_attendance_records a
          ON a.school_id = sp.school_id
         AND a.staff_principal_id = sp.principal_id
         AND a.attendance_date >= $2::date
         AND a.attendance_date < $3::date
        WHERE sp.school_id = $1
          AND (sp.employment_status = 'active' OR EXISTS (
              SELECT 1 FROM staff_attendance_records historic
              WHERE historic.school_id = sp.school_id
                AND historic.staff_principal_id = sp.principal_id
                AND historic.attendance_date >= $2::date
                AND historic.attendance_date < $3::date
          ))
        GROUP BY sp.principal_id, sp.employee_code, sp.full_name, sp.designation
        ORDER BY LOWER(sp.full_name), sp.employee_code, sp.principal_id
        LIMIT 1000
    `, schoolID, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
    if err != nil { return nil, fmt.Errorf("list monthly staff attendance: %w", err) }
    defer rows.Close()
    items := make([]StaffAttendanceMonthlySummary, 0)
    for rows.Next() {
        var item StaffAttendanceMonthlySummary
        if err := rows.Scan(&item.StaffPrincipalID, &item.EmployeeCode, &item.FullName, &item.Designation,
            &item.PresentDays, &item.AbsentDays, &item.LateDays, &item.HalfDays, &item.Records); err != nil {
            return nil, fmt.Errorf("scan monthly staff attendance: %w", err)
        }
        items = append(items, item)
    }
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate monthly staff attendance: %w", err) }
    return items, nil
}

func (r *StaffAttendanceRepository) Mark(ctx context.Context, actorID, schoolID, staffID int64, day time.Time, status string, remarks *string) (*StaffAttendanceRecord, error) {
    if r == nil || r.db == nil || actorID <= 0 || schoolID <= 0 || staffID <= 0 || day.Year() < 1 || !validStaffAttendanceStatus(status) {
        return nil, ErrInvalidStaffAttendance
    }
    status = strings.TrimSpace(status)
    if remarks != nil {
        value := strings.TrimSpace(*remarks)
        if len([]rune(value)) > 500 { return nil, ErrInvalidStaffAttendance }
        if value == "" { remarks = nil } else { remarks = &value }
    }
    date := day.Format("2006-01-02")
    today := schooltime.TodayDate()
    if date > today { return nil, ErrInvalidStaffAttendance }

    tx, err := r.db.Begin(ctx)
    if err != nil { return nil, fmt.Errorf("begin staff attendance mark: %w", err) }
    defer func() { _ = tx.Rollback(ctx) }()

    permission := "staff.attendance.mark"
    if date < today { permission = "staff.attendance.correct_past" }
    if err := authorizeGuardian(ctx, tx, actorID, schoolID, permission); err != nil {
        return nil, ErrStaffAttendanceAccessDenied
    }

    var active bool
    if err := tx.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM staff_profiles sp
            WHERE sp.principal_id = $1 AND sp.school_id = $2
              AND sp.employment_status = 'active'
        )
    `, staffID, schoolID).Scan(&active); err != nil {
        return nil, fmt.Errorf("check staff attendance target: %w", err)
    }
    if !active { return nil, ErrStaffAttendanceNotFound }

    var id int64
    err = tx.QueryRow(ctx, `
        INSERT INTO staff_attendance_records
            (school_id, staff_principal_id, attendance_date, status, remarks, marked_by_principal_id, updated_by_principal_id)
        VALUES ($1,$2,$3,$4,$5,$6,$6)
        RETURNING id
    `, schoolID, staffID, date, status, remarks, actorID).Scan(&id)
    if err != nil { return nil, staffAttendanceWriteError(err) }

    if _, err := tx.Exec(ctx, `
        INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
        VALUES($1,$2,'school.staff_attendance.marked','staff_attendance_record',$3,'success',
               jsonb_build_object('staff_principal_id',$4::bigint,'attendance_date',$5::date,'status',$6::text))
    `, actorID, schoolID, fmt.Sprint(id), staffID, date, status); err != nil {
        return nil, fmt.Errorf("audit staff attendance mark: %w", err)
    }
    if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit staff attendance mark: %w", err) }
    return r.get(ctx, schoolID, id)
}

func (r *StaffAttendanceRepository) Update(ctx context.Context, actorID, schoolID, recordID int64, status string, remarks *string) (*StaffAttendanceRecord, error) {
    if r == nil || r.db == nil || actorID <= 0 || schoolID <= 0 || recordID <= 0 || !validStaffAttendanceStatus(strings.TrimSpace(status)) {
        return nil, ErrInvalidStaffAttendance
    }
    status = strings.TrimSpace(status)
    if remarks != nil {
        value := strings.TrimSpace(*remarks)
        if len([]rune(value)) > 500 { return nil, ErrInvalidStaffAttendance }
        if value == "" { remarks = nil } else { remarks = &value }
    }
    tx, err := r.db.Begin(ctx)
    if err != nil { return nil, fmt.Errorf("begin staff attendance update: %w", err) }
    defer func() { _ = tx.Rollback(ctx) }()

    // Determine the actor's attendance-write scope before looking up the
    // record. A caller with neither permission must not be able to enumerate
    // attendance record IDs through different not-found/forbidden responses.
    sameDayErr := authorizeGuardian(ctx, tx, actorID, schoolID, "staff.attendance.update_same_day")
    pastErr := authorizeGuardian(ctx, tx, actorID, schoolID, "staff.attendance.correct_past")
    if sameDayErr != nil && !errors.Is(sameDayErr, ErrGuardianAccessDenied) {
        return nil, fmt.Errorf("authorize same-day staff attendance update: %w", sameDayErr)
    }
    if pastErr != nil && !errors.Is(pastErr, ErrGuardianAccessDenied) {
        return nil, fmt.Errorf("authorize past staff attendance correction: %w", pastErr)
    }
    canUpdateSameDay := sameDayErr == nil
    canCorrectPast := pastErr == nil
    if !canUpdateSameDay && !canCorrectPast {
        return nil, ErrStaffAttendanceAccessDenied
    }

    var day time.Time
    var staffID int64
    err = tx.QueryRow(ctx, `
        SELECT staff_principal_id, attendance_date FROM staff_attendance_records
        WHERE id = $1 AND school_id = $2 FOR UPDATE
    `, recordID, schoolID).Scan(&staffID, &day)
    if errors.Is(err, pgx.ErrNoRows) {
        // Only callers with both scopes can receive a definitive not-found
        // result; partial-scope callers get the same denial as a wrong-date
        // record, so record existence is not disclosed.
        if canUpdateSameDay && canCorrectPast { return nil, ErrStaffAttendanceNotFound }
        return nil, ErrStaffAttendanceAccessDenied
    }
    if err != nil { return nil, fmt.Errorf("load staff attendance record: %w", err) }

    if day.Format("2006-01-02") < schooltime.TodayDate() {
        if !canCorrectPast { return nil, ErrStaffAttendanceAccessDenied }
    } else if !canUpdateSameDay {
        return nil, ErrStaffAttendanceAccessDenied
    }
    _, err = tx.Exec(ctx, `
        UPDATE staff_attendance_records
           SET status = $3, remarks = $4, updated_by_principal_id = $5, updated_at = NOW()
         WHERE id = $1 AND school_id = $2
    `, recordID, schoolID, status, remarks, actorID)
    if err != nil { return nil, staffAttendanceWriteError(err) }
    if _, err := tx.Exec(ctx, `
        INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
        VALUES($1,$2,'school.staff_attendance.updated','staff_attendance_record',$3,'success',
               jsonb_build_object('staff_principal_id',$4::bigint,'attendance_date',$5::date,'status',$6::text))
    `, actorID, schoolID, fmt.Sprint(recordID), staffID, day.Format("2006-01-02"), status); err != nil {
        return nil, fmt.Errorf("audit staff attendance update: %w", err)
    }
    if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit staff attendance update: %w", err) }
    return r.get(ctx, schoolID, recordID)
}

func (r *StaffAttendanceRepository) get(ctx context.Context, schoolID, recordID int64) (*StaffAttendanceRecord, error) {
    var item StaffAttendanceRecord
    err := r.db.QueryRow(ctx, `
        SELECT a.id, a.school_id, a.staff_principal_id, sp.employee_code, sp.full_name, sp.designation,
               a.attendance_date::text, a.status, a.remarks, marker.username, a.updated_at
        FROM staff_attendance_records a
        JOIN staff_profiles sp ON sp.principal_id = a.staff_principal_id AND sp.school_id = a.school_id
        JOIN users marker ON marker.principal_id = a.marked_by_principal_id AND marker.school_id = a.school_id
        WHERE a.school_id = $1 AND a.id = $2
    `, schoolID, recordID).Scan(&item.ID, &item.SchoolID, &item.StaffPrincipalID, &item.EmployeeCode, &item.FullName,
        &item.Designation, &item.AttendanceDate, &item.Status, &item.Remarks, &item.MarkedByUsername, &item.UpdatedAt)
    if err != nil { return nil, staffAttendanceWriteError(err) }
    return &item, nil
}

func (r *StaffAttendanceRepository) MarkBulk(ctx context.Context, actorID, schoolID int64, staffIDs []int64, day time.Time, status string, remarks *string) (int, error) {
    if r == nil || r.db == nil || actorID <= 0 || schoolID <= 0 || len(staffIDs) == 0 || len(staffIDs) > 1000 || day.Year() < 1 || !validStaffAttendanceStatus(status) {
        return 0, ErrInvalidStaffAttendance
    }
    seen := make(map[int64]struct{}, len(staffIDs))
    for _, staffID := range staffIDs {
        if staffID <= 0 { return 0, ErrInvalidStaffAttendance }
        if _, exists := seen[staffID]; exists { return 0, ErrInvalidStaffAttendance }
        seen[staffID] = struct{}{}
    }
    status = strings.TrimSpace(status)
    if remarks != nil {
        value := strings.TrimSpace(*remarks)
        if len([]rune(value)) > 500 { return 0, ErrInvalidStaffAttendance }
        if value == "" { remarks = nil } else { remarks = &value }
    }
    date := day.Format("2006-01-02")
    today := schooltime.TodayDate()
    if date > today { return 0, ErrInvalidStaffAttendance }

    tx, err := r.db.Begin(ctx)
    if err != nil { return 0, fmt.Errorf("begin bulk staff attendance mark: %w", err) }
    defer func() { _ = tx.Rollback(ctx) }()

    permission := "staff.attendance.mark"
    if date < today { permission = "staff.attendance.correct_past" }
    if err := authorizeGuardian(ctx, tx, actorID, schoolID, permission); err != nil {
        return 0, ErrStaffAttendanceAccessDenied
    }

    for _, staffID := range staffIDs {
        var active bool
        if err := tx.QueryRow(ctx, `
            SELECT EXISTS (
                SELECT 1 FROM staff_profiles
                WHERE principal_id = $1 AND school_id = $2 AND employment_status = 'active'
            )
        `, staffID, schoolID).Scan(&active); err != nil {
            return 0, fmt.Errorf("check bulk staff attendance target: %w", err)
        }
        if !active { return 0, ErrStaffAttendanceNotFound }

        var id int64
        if err := tx.QueryRow(ctx, `
            INSERT INTO staff_attendance_records
                (school_id, staff_principal_id, attendance_date, status, remarks, marked_by_principal_id, updated_by_principal_id)
            VALUES ($1,$2,$3,$4,$5,$6,$6)
            RETURNING id
        `, schoolID, staffID, date, status, remarks, actorID).Scan(&id); err != nil {
            return 0, staffAttendanceWriteError(err)
        }
        if _, err := tx.Exec(ctx, `
            INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
            VALUES($1,$2,'school.staff_attendance.marked','staff_attendance_record',$3,'success',
                   jsonb_build_object('staff_principal_id',$4::bigint,'attendance_date',$5::date,'status',$6::text,'bulk',TRUE))
        `, actorID, schoolID, fmt.Sprint(id), staffID, date, status); err != nil {
            return 0, fmt.Errorf("audit bulk staff attendance mark: %w", err)
        }
    }
    if err := tx.Commit(ctx); err != nil { return 0, fmt.Errorf("commit bulk staff attendance mark: %w", err) }
    return len(staffIDs), nil
}
