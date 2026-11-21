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
			name:       "duplicate school registration number",
			err:        repository.ErrStudentRegistrationNumberConflict,
			wantField:  "school_registration_number",
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid student contact email",
			err:        repository.ErrInvalidStudentContactEmail,
			wantField:  "email_address",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid student date of birth",
			err:        repository.ErrInvalidStudentDateOfBirth,
			wantField:  "date_of_birth",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid student measurement date",
			err:        repository.ErrInvalidStudentMeasurementDate,
			wantField:  "measured_on",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "promotion before recorded attendance",
			err:        repository.ErrPromotionBeforeAttendance,
			wantField:  "admission_date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "admission date after enrollment start",
			err:        repository.ErrAdmissionAfterEnrollmentDate,
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

func TestParseEnrollmentDateAcceptsOnlyValidCalendarDates(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid date", input: "2026-11-17", want: "2026-11-17"},
		{name: "trim surrounding whitespace", input: " 2026-11-17 ", want: "2026-11-17"},
		{name: "leap day", input: "2028-02-29", want: "2028-02-29"},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid leap day", input: "2027-02-29", wantErr: true},
		{name: "impossible day", input: "2026-11-31", wantErr: true},
		{name: "timestamp is not a date-only API value", input: "2026-11-17T00:00:00Z", wantErr: true},
		{name: "locale date is not accepted", input: "17/11/2026", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEnrollmentDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseEnrollmentDate(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEnrollmentDate(%q): %v", tt.input, err)
			}
			if formatted := got.Format("2006-01-02"); formatted != tt.want {
				t.Fatalf("parsed date = %q, want %q", formatted, tt.want)
			}
		})
	}
}

func TestAcademicErrorRequiresSubjectImpactReview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPatch, "/school/subjects/44", nil)

	academicError(ctx, repository.ErrSubjectDeactivationConfirmationRequired, "Subject")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	var payload struct {
		Status string `json:"status"`
		Code string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "error" || payload.Code != "subject_deactivation_confirmation_required" || payload.Message == "" {
		t.Fatalf("unexpected subject deactivation response: %+v", payload)
	}
}

func TestAcademicErrorExplainsInactiveSubjectEnrollmentConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		err error
		code string
	}{
		{err: repository.ErrInactiveSubjectInCurriculum, code: "inactive_subject_in_curriculum"},
		{err: repository.ErrInactiveSubjectSelection, code: "inactive_subject_selection"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPut, "/school/enrollments/1/subjects", nil)
			academicError(ctx, tt.err, "Student subject choices")
			if recorder.Code != http.StatusConflict {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
			}
			var payload struct { Code string `json:"code"`; Message string `json:"message"` }
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil { t.Fatalf("decode response: %v", err) }
			if payload.Code != tt.code || payload.Message == "" { t.Fatalf("unexpected conflict response: %+v", payload) }
		})
	}
}
