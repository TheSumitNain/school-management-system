package repository

import (
 "context"
 "testing"
 "time"
)

func TestStaffLeaveHolidayV59SchemaAndPermissionsPostgresIntegration(t *testing.T) {
 _,pool,userID:=newPostgresIntegrationRepository(t)
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
 var ready bool
 err:=pool.QueryRow(ctx,`
 SELECT to_regclass('public.staff_leave_types') IS NOT NULL
 AND to_regclass('public.staff_leave_requests') IS NOT NULL
 AND to_regclass('public.school_holidays') IS NOT NULL
 AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_zz_grant_staff_leave_holiday_permissions' AND NOT tgisinternal)
 AND (SELECT COUNT(*)=6 FROM public.permissions WHERE permission_key=ANY(ARRAY[
 'staff.leave.read','staff.leave.manage_types','staff.leave.manage_requests','staff.leave.decide',
 'staff.holidays.read','staff.holidays.manage']::text[]) AND scope_type='school' AND is_delegable=FALSE AND is_active=TRUE)
 AND EXISTS(
  SELECT 1 FROM public.users u JOIN public.roles r ON r.school_id=u.school_id AND r.role_key='school_admin'
  JOIN public.role_permissions rp ON rp.role_id=r.id JOIN public.permissions p ON p.id=rp.permission_id
  WHERE u.id=$1 AND p.permission_key='staff.leave.read' AND r.is_system=TRUE
 )
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conname='fk_staff_leave_request_staff_school')
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conname='fk_staff_leave_request_type_school')
 `).Scan(&ready)
 if err!=nil{t.Fatalf("inspect V59 staff leave/holiday schema: %v",err)}
 if !ready{t.Fatal("V59 leave/holiday tables, constraints, six non-delegable permissions, or School Admin grants are missing")}
 var granted int
 if err:=pool.QueryRow(ctx,`SELECT COUNT(*)::integer FROM public.users u JOIN public.roles r ON r.school_id=u.school_id AND r.role_key='school_admin' AND r.is_system=TRUE JOIN public.role_permissions rp ON rp.role_id=r.id JOIN public.permissions p ON p.id=rp.permission_id WHERE u.id=$1 AND p.permission_key IN ('staff.leave.read','staff.leave.manage_types','staff.leave.manage_requests','staff.leave.decide','staff.holidays.read','staff.holidays.manage')`,userID).Scan(&granted);err!=nil{t.Fatalf("count V59 permission grants: %v",err)}
 if granted!=6{t.Fatalf("V59 grants=%d want=6 leave/holiday grants",granted)}
}
