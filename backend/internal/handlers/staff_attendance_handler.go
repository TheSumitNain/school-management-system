package handlers

import (
    "context"
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"

    "school-management-system/internal/repository"
    "school-management-system/internal/schooltime"
    "school-management-system/internal/services"

    "github.com/gin-gonic/gin"
)

type StaffAttendanceManagementService interface {
    ListDaily(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceRecord, error)
    ListMonthly(context.Context, int64, int64, time.Time) ([]repository.StaffAttendanceMonthlySummary, error)
    Mark(context.Context, int64, int64, int64, time.Time, string, *string) (*repository.StaffAttendanceRecord, error)
    MarkBulk(context.Context, int64, int64, []int64, time.Time, string, *string) (int, error)
    Update(context.Context, int64, int64, int64, string, *string) (*repository.StaffAttendanceRecord, error)
}

func parseStaffAttendanceDate(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01-02") != value {
		return time.Time{}, false
	}
	return parsed, true
}

func parseStaffAttendanceMonth(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01") != value {
		return time.Time{}, false
	}
	return parsed, true
}

type StaffAttendanceHandler struct { service StaffAttendanceManagementService }

func NewStaffAttendanceHandler(service StaffAttendanceManagementService) *StaffAttendanceHandler {
    return &StaffAttendanceHandler{service: service}
}

func (h *StaffAttendanceHandler) unavailable(c *gin.Context) bool {
    if h == nil || h.service == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Staff attendance service is unavailable"})
        return true
    }
    return false
}

func (h *StaffAttendanceHandler) ListDaily(c *gin.Context) {
    actor, school, ok := staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
    if h.unavailable(c) { return }
    raw := strings.TrimSpace(c.Query("date"))
    if raw == "" { raw = schooltime.TodayDate() }
    day, valid := parseStaffAttendanceDate(raw)
    if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"date must use YYYY-MM-DD"}); return }
    items, err := h.service.ListDaily(c.Request.Context(), actor, school, day)
    if err != nil { writeStaffAttendanceError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"status":"success","date":raw,"data":items})
}

func (h *StaffAttendanceHandler) ListMonthly(c *gin.Context) {
    actor, school, ok := staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
    if h.unavailable(c) { return }
    raw := strings.TrimSpace(c.Query("month"))
    if raw == "" { raw = schooltime.Today().Format("2006-01") }
    month, valid := parseStaffAttendanceMonth(raw)
    if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"month must use YYYY-MM"}); return }
    items, err := h.service.ListMonthly(c.Request.Context(), actor, school, month)
    if err != nil { writeStaffAttendanceError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"status":"success","month":raw,"data":items})
}

type staffAttendanceMarkRequest struct {
    StaffPrincipalID int64 `json:"staff_principal_id"`
    AttendanceDate string `json:"attendance_date"`
    Status string `json:"status"`
    Remarks *string `json:"remarks"`
}

func (h *StaffAttendanceHandler) Mark(c *gin.Context) {
    actor, school, ok := staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
    if h.unavailable(c) { return }
    var req staffAttendanceMarkRequest
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid staff attendance payload"}); return }
    day, valid := parseStaffAttendanceDate(req.AttendanceDate)
    if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"attendance_date must use YYYY-MM-DD"}); return }
    item, err := h.service.Mark(c.Request.Context(), actor, school, req.StaffPrincipalID, day, req.Status, req.Remarks)
    if err != nil { writeStaffAttendanceError(c, err); return }
    c.JSON(http.StatusCreated, gin.H{"status":"success","data":item})
}

type staffAttendanceBulkMarkRequest struct {
    StaffPrincipalIDs []int64 `json:"staff_principal_ids"`
    AttendanceDate string `json:"attendance_date"`
    Status string `json:"status"`
    Remarks *string `json:"remarks"`
}

func (h *StaffAttendanceHandler) MarkBulk(c *gin.Context) {
    actor, school, ok := staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
    if h.unavailable(c) { return }
    var req staffAttendanceBulkMarkRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid bulk staff attendance payload"})
        return
    }
    day, valid := parseStaffAttendanceDate(req.AttendanceDate)
    if !valid {
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"attendance_date must use YYYY-MM-DD"})
        return
    }
    created, err := h.service.MarkBulk(c.Request.Context(), actor, school, req.StaffPrincipalIDs, day, req.Status, req.Remarks)
    if err != nil { writeStaffAttendanceError(c, err); return }
    c.JSON(http.StatusCreated, gin.H{"status":"success","created":created})
}

type staffAttendanceUpdateRequest struct {
    Status string `json:"status"`
    Remarks *string `json:"remarks"`
}

func (h *StaffAttendanceHandler) Update(c *gin.Context) {
    actor, school, ok := staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
    if h.unavailable(c) { return }
    recordID, err := strconv.ParseInt(c.Param("recordID"), 10, 64)
    if err != nil || recordID <= 0 { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid staff attendance record ID"}); return }
    var req staffAttendanceUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid staff attendance payload"}); return }
    item, err := h.service.Update(c.Request.Context(), actor, school, recordID, req.Status, req.Remarks)
    if err != nil { writeStaffAttendanceError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"status":"success","data":item})
}

func writeStaffAttendanceError(c *gin.Context, err error) {
    switch {
    case errors.Is(err, repository.ErrStaffAttendanceAccessDenied):
        c.JSON(http.StatusForbidden, gin.H{"status":"error","message":"You are not authorized to access staff attendance"})
    case errors.Is(err, repository.ErrStaffAttendanceNotFound):
        c.JSON(http.StatusNotFound, gin.H{"status":"error","message":"Staff attendance record or employee not found"})
    case errors.Is(err, repository.ErrStaffAttendanceConflict):
        c.JSON(http.StatusConflict, gin.H{"status":"error","message":"Attendance is already marked for this employee and date"})
    case errors.Is(err, repository.ErrInvalidStaffAttendance), errors.Is(err, services.ErrInvalidStaffAttendance):
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Check the staff attendance fields"})
    default:
        c.JSON(http.StatusInternalServerError, gin.H{"status":"error","message":"Unable to process staff attendance"})
    }
}
