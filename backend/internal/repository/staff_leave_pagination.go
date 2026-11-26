package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type StaffLeaveRequestPage struct {
	Items []StaffLeaveRequest
	Total int64
}

func (r *StaffLeaveRepository) ListRequestsPage(ctx context.Context, actor, school int64, status string, staffID int64, onDate, fromDate, toDate string, page, pageSize int) (StaffLeaveRequestPage, error) {
	result := StaffLeaveRequestPage{Items: []StaffLeaveRequest{}}
	if r == nil || r.db == nil || actor <= 0 || school <= 0 || page < 1 || pageSize < 1 || pageSize > 100 || page-1 > int(^uint(0)>>1)/pageSize {
		return result, ErrInvalidStaffLeave
	}
	offset := (page - 1) * pageSize
	if offset > 1000000 {
		return result, ErrInvalidStaffLeave
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin staff leave page snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, callerOwnsTransaction := r.db.(pgx.Tx); !callerOwnsTransaction {
		if _, err := tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
			return result, fmt.Errorf("set staff leave page snapshot: %w", err)
		}
	}
	if err := authorizeGuardian(ctx, tx, actor, school, "staff.leave.read"); err != nil {
		return result, ErrStaffLeaveAccessDenied
	}
	if err := validateStaffLeaveRequestDateFilters(onDate, fromDate, toDate); err != nil {
		return result, err
	}
	if onDate != "" && (fromDate != "" || toDate != "") {
		return result, ErrInvalidStaffLeave
	}

	const where = ` WHERE q.school_id=$1 AND ($2='' OR q.status=$2) AND ($3=0 OR q.staff_principal_id=$3)
		AND (NULLIF($4,'') IS NULL OR daterange(q.start_date,q.end_date,'[]') @> NULLIF($4,'')::date)
		AND (NULLIF($5,'') IS NULL OR daterange(q.start_date,q.end_date,'[]') && daterange(NULLIF($5,'')::date,NULLIF($6,'')::date,'[]'))`
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM staff_leave_requests q"+where, school, status, staffID, onDate, fromDate, toDate).Scan(&result.Total); err != nil {
		return result, fmt.Errorf("count staff leave requests: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT q.id,q.staff_principal_id,sp.employee_code,sp.full_name,q.leave_type_id,t.name,
		q.start_date::text,q.end_date::text,q.duration_type,q.half_day_session,q.reason,q.status,
		q.decision_comment,u.username,q.decided_at,q.created_at
		FROM staff_leave_requests q
		JOIN staff_profiles sp ON sp.principal_id=q.staff_principal_id AND sp.school_id=q.school_id
		JOIN staff_leave_types t ON t.id=q.leave_type_id AND t.school_id=q.school_id
		LEFT JOIN users u ON u.principal_id=q.decided_by_principal_id AND u.school_id=q.school_id`+where+`
		ORDER BY q.start_date DESC,q.id DESC LIMIT $7 OFFSET $8`, school, status, staffID, onDate, fromDate, toDate, pageSize, offset)
	if err != nil {
		return result, fmt.Errorf("list staff leave request page: %w", err)
	}
	for rows.Next() {
		var item StaffLeaveRequest
		if err := rows.Scan(&item.ID, &item.StaffPrincipalID, &item.EmployeeCode, &item.StaffName, &item.LeaveTypeID, &item.LeaveTypeName,
			&item.StartDate, &item.EndDate, &item.DurationType, &item.HalfDaySession, &item.Reason, &item.Status,
			&item.DecisionComment, &item.DecidedByUsername, &item.DecidedAt, &item.CreatedAt); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit staff leave page snapshot: %w", err)
	}
	return result, nil
}
