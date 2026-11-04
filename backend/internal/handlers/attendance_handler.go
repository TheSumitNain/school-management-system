package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type AttendanceManagementService interface {
	List(context.Context, int64, int64, int64, int64, time.Time) ([]repository.AttendanceRecord, error)
	Mark(context.Context, int64, int64, int64, time.Time, string, *string) (*repository.AttendanceRecord, error)
	Update(context.Context, int64, int64, int64, string, *string) (*repository.AttendanceRecord, error)
}

type AttendanceHandler struct { service AttendanceManagementService }
func NewAttendanceHandler(service AttendanceManagementService) *AttendanceHandler { return &AttendanceHandler{service: service} }

func (h *AttendanceHandler) List(c *gin.Context) {
	if h == nil || h.service == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Attendance service is unavailable"}); return }
	yearID, err := optionalPositiveID(c.Query("academic_year_id"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid academic_year_id"}); return }
	sectionID, err := optionalPositiveID(c.Query("section_id"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid section_id"}); return }
	var day time.Time
	if raw := strings.TrimSpace(c.Query("date")); raw != "" {
		day, err = time.Parse("2006-01-02", raw)
		if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"date must use YYYY-MM-DD"}); return }
	}
	items, err := h.service.List(c.Request.Context(), attendanceActor(c), attendanceSchool(c), yearID, sectionID, day)
	if err != nil { writeAttendanceError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","data":items})
}

type attendanceMarkRequest struct {
	EnrollmentID int64 `json:"enrollment_id"`
	AttendanceDate string `json:"attendance_date"`
	Status string `json:"status"`
	Remarks *string `json:"remarks"`
}
func (h *AttendanceHandler) Mark(c *gin.Context) {
	if h == nil || h.service == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Attendance service is unavailable"}); return }
	var req attendanceMarkRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid attendance payload"}); return }
	day, err := time.Parse("2006-01-02", req.AttendanceDate)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"attendance_date must use YYYY-MM-DD"}); return }
	v, err := h.service.Mark(c.Request.Context(), attendanceActor(c), attendanceSchool(c), req.EnrollmentID, day, req.Status, req.Remarks)
	if err != nil { writeAttendanceError(c, err); return }
	c.JSON(http.StatusCreated, gin.H{"status":"success","data":v})
}

type attendanceUpdateRequest struct { Status string `json:"status"`; Remarks *string `json:"remarks"` }
func (h *AttendanceHandler) Update(c *gin.Context) {
	if h == nil || h.service == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Attendance service is unavailable"}); return }
	id, err := strconv.ParseInt(c.Param("attendanceID"), 10, 64)
	if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid attendance ID"}); return }
	var req attendanceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid attendance payload"}); return }
	v, err := h.service.Update(c.Request.Context(), attendanceActor(c), attendanceSchool(c), id, req.Status, req.Remarks)
	if err != nil { writeAttendanceError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","data":v})
}

func attendanceActor(c *gin.Context) int64 { v,_:=c.Get(middleware.ContextPrincipalIDKey); id,_:=v.(int64); return id }
func attendanceSchool(c *gin.Context) int64 { v,_:=c.Get(middleware.ContextSchoolIDKey); id,_:=v.(int64); return id }

func writeAttendanceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrAttendanceAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"status":"error","message":"You are not authorized to access this attendance scope"})
	case errors.Is(err, repository.ErrAttendanceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status":"error","message":"Attendance record not found"})
	case errors.Is(err, repository.ErrAttendanceConflict):
		c.JSON(http.StatusConflict, gin.H{"status":"error","message":"Attendance already exists for this student and date"})
	case errors.Is(err, repository.ErrInvalidAttendanceInput):
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid attendance data"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status":"error","message":"Attendance operation failed"})
	}
}
