package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

func TestAcademicErrorReturnsFieldSpecificAdmissionConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		wantField  string
		wantStatus int
	}{
		{
			name:       "duplicate year admission number",
			err:        repository.ErrAcademicAdmissionNumberConflict,
			wantField:  "admission_number",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "duplicate active section roll number",
			err:        repository.ErrAcademicRollNumberConflict,
			wantField:  "roll_number",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid student contact email",
			err:        repository.ErrInvalidStudentContactEmail,
			wantField:  "email_address",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "promotion before recorded attendance",
			err:        repository.ErrPromotionBeforeAttendance,
			wantField:  "admission_date",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/school/students", nil)

			academicError(ctx, tt.err, "Student admission")

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			var payload struct {
				Status      string            `json:"status"`
				Message     string            `json:"message"`
				FieldErrors map[string]string `json:"field_errors"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Status != "error" || payload.Message == "" || payload.FieldErrors[tt.wantField] == "" {
				t.Fatalf("unexpected field conflict response: %+v", payload)
			}
			if len(payload.FieldErrors) != 1 {
				t.Fatalf("field_errors = %#v, want exactly the conflicting field", payload.FieldErrors)
			}
		})
	}
}

func TestAcademicErrorDoesNotMislabelGenericUniqueConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/school/students", nil)

	academicError(ctx, repository.ErrAcademicConflict, "Student admission")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := payload["field_errors"]; exists {
		t.Fatalf("generic academic conflict must not claim a specific field: %#v", payload)
	}
}
