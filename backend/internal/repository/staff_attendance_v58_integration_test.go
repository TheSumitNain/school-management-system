package repository

import (
    "context"
    "errors"
    "testing"
    "time"
    "strings"

    schooltime "school-management-system/internal/schooltime"
)

func TestStaffAttendanceV58SchemaAndPermissionsPostgresIntegration(t *testing.T) {
    _, pool, userID := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var ready bool
    err := pool.QueryRow(ctx, `
        SELECT
            to_regclass('public.staff_attendance_records') IS NOT NULL
            AND to_regclass('public.uq_staff_attendance_school_staff_date') IS NOT NULL
            AND to_regclass('public.ix_staff_attendance_school_date') IS NOT NULL
            AND EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_validate_staff_attendance_date' AND NOT tgisinternal)
            AND EXISTS (SELECT 1 FROM pg_proc WHERE proname='validate_staff_attendance_date')
            AND (
                SELECT COUNT(*) = 4
                FROM public.permissions
                WHERE permission_key IN (
                    'staff.attendance.read',
                    'staff.attendance.mark',
                    'staff.attendance.update_same_day',
                    'staff.attendance.correct_past'
                )
                  AND scope_type = 'school'
                  AND is_active = TRUE
            )
            AND EXISTS (
                SELECT 1
                FROM public.schools s
                JOIN public.roles r
                  ON r.school_id = s.id
                 AND r.role_key = 'school_admin'
                 AND r.scope_type = 'school'
                 AND r.principal_type = 'staff'
                 AND r.is_system = TRUE
                JOIN public.role_permissions rp ON rp.role_id = r.id
                JOIN public.permissions p ON p.id = rp.permission_id
                WHERE p.permission_key = 'staff.attendance.read'
            )
            AND EXISTS (
                SELECT 1 FROM pg_trigger
                WHERE tgname = 'trg_z_grant_staff_attendance_permissions'
                  AND NOT tgisinternal
            )
    `).Scan(&ready)
    if err != nil { t.Fatalf("inspect V58 staff attendance schema and permissions: %v", err) }
    if !ready { t.Fatal("V58 staff attendance table, unique index, date trigger, permission catalogue, School Admin grant, or future-school trigger is missing") }

    var granted int
    if err := pool.QueryRow(ctx, `
        SELECT COUNT(*)::integer
        FROM public.roles r
        JOIN public.role_permissions rp ON rp.role_id = r.id
        JOIN public.permissions p ON p.id = rp.permission_id
        WHERE r.school_id = (SELECT school_id FROM public.users WHERE id = $1)
          AND r.role_key = 'school_admin'
          AND r.scope_type = 'school'
          AND r.principal_type = 'staff'
          AND r.is_system = TRUE
          AND p.permission_key = ANY(ARRAY[
              'staff.attendance.read',
              'staff.attendance.mark',
              'staff.attendance.update_same_day',
              'staff.attendance.correct_past'
          ]::text[])
    `, userID).Scan(&granted); err != nil {
        t.Fatalf("count new-school staff attendance grants: %v", err)
    }
    if granted != 4 { t.Fatalf("new-school staff attendance grants=%d want=4", granted) }

    var definition string
    if err := pool.QueryRow(ctx, `
        SELECT pg_get_functiondef('public.validate_staff_attendance_date()'::regprocedure)
    `).Scan(&definition); err != nil {
        t.Fatalf("inspect staff attendance date guard: %v", err)
    }
    if !strings.Contains(definition, "Asia/Kolkata") {
        t.Fatal("staff attendance date trigger must use the Asia/Kolkata school calendar")
    }
}

func TestStaffAttendanceRejectsDuplicatePerSchoolDatePostgresIntegration(t *testing.T) {
    _, pool, _ := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var constraints int
    if err := pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM pg_constraint
        WHERE conrelid = 'public.staff_attendance_records'::regclass
          AND conname = 'uq_staff_attendance_school_staff_date'
          AND contype = 'u'
    `).Scan(&constraints); err != nil {
        t.Fatalf("inspect staff attendance uniqueness: %v", err)
    }
    if constraints != 1 { t.Fatalf("staff attendance uniqueness constraints=%d want=1", constraints) }
}


// An actor without attendance permissions must not learn whether a staff
// principal exists or is active through the attendance-marking result.
func TestStaffAttendanceMarkAuthorizesBeforeResolvingTargetPostgresIntegration(t *testing.T) {
    _, pool, actorUserID := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    var actorPrincipalID, schoolID int64
    if err := pool.QueryRow(ctx, `
        SELECT principal_id, school_id FROM users WHERE id = $1
    `, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
        t.Fatalf("load unprivileged attendance actor: %v", err)
    }

    date, err := time.Parse("2006-01-02", schooltime.TodayDate())
    if err != nil {
        t.Fatalf("parse school-local attendance date: %v", err)
    }

    _, err = NewStaffAttendanceRepository(pool).Mark(
        ctx, actorPrincipalID, schoolID, 9_999_999_999, date, "present", nil,
    )
    if !errors.Is(err, ErrStaffAttendanceAccessDenied) {
        t.Fatalf("unauthorized attendance mark error = %v, want ErrStaffAttendanceAccessDenied (target existence must not be disclosed)", err)
    }
}
