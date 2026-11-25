package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeStaffAccountService struct {
	user *models.User
	err error
	actorID int64
	schoolID int64
	username string
	password string
	calls int
	profile models.StaffProfile
	profileTargetID int64
	profileCalls int
	directoryFilter models.StaffDirectoryFilter
	directoryActorID int64
	directorySchoolID int64
}

func (f *fakeStaffAccountService) UpdateProfile(_ context.Context, actorID, schoolID, targetID int64, profile models.StaffProfile) (*models.User, error) {
	f.profileCalls++
	f.actorID, f.schoolID, f.profileTargetID, f.profile = actorID, schoolID, targetID, profile
	return f.user, f.err
}

func (f *fakeStaffAccountService) Create(_ context.Context, actorID, schoolID int64, username, password string) (*models.User, error) {
	f.calls++
	f.actorID, f.schoolID, f.username, f.password = actorID, schoolID, username, password
	return f.user, f.err
}

func (f *fakeStaffAccountService) CreateWithProfile(_ context.Context, actorID, schoolID int64, username, password string, _ models.StaffProfile) (*models.User, error) {
	f.calls++
	f.actorID, f.schoolID, f.username, f.password = actorID, schoolID, username, password
	return f.user, f.err
}

func (f *fakeStaffAccountService) List(context.Context, int64, int64) ([]models.User, error) {
	return []models.User{{ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one", PasswordHash: "private-hash-marker", Status: "active", MustChangePassword: true}}, f.err
}

func (f *fakeStaffAccountService) ListDirectoryPage(_ context.Context, actorID, schoolID int64, filter models.StaffDirectoryFilter) (*models.StaffDirectoryPage, error) {
	f.directoryActorID, f.directorySchoolID, f.directoryFilter = actorID, schoolID, filter
	return &models.StaffDirectoryPage{Staff: []models.User{{ID: 15, PrincipalID: 27, SchoolID: schoolID, Username: "teacher.one", Status: "active"}}, Total: 51, Limit: filter.Limit, Offset: filter.Offset}, f.err
}

func (f *fakeStaffAccountService) Disable(context.Context, int64, int64, int64) error {
	return f.err
}

func (f *fakeStaffAccountService) Update(context.Context, int64, int64, int64, string) (*models.User, error) {
	return f.user, f.err
}

func (f *fakeStaffAccountService) Enable(context.Context, int64, int64, int64, string) error {
	return f.err
}

func TestStaffHandlerCreateSuccessDoesNotExposePassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one", PasswordHash: "private-hash-marker",
		Status: "active", MustChangePassword: true,
	}}
	router := gin.New()
	router.POST("/school/staff", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Create)
	req := httptest.NewRequest(http.MethodPost, "/school/staff",
		strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123","full_name":"Rahul Sharma","employee_code":"EMP-001","designation":"Teacher","mobile_number":"9876543210"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || service.calls != 1 ||
		service.actorID != 31 || service.schoolID != 8 || service.username != "teacher.one" {
		t.Fatalf("unexpected staff create response: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "A-Strong-Initial-Pass-123") || strings.Contains(rec.Body.String(), "PasswordHash") || strings.Contains(rec.Body.String(), "private-hash-marker") {
		t.Fatalf("staff create response leaked password material: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"must_change_password":true`) {
		t.Fatalf("new staff account must require first-login password change: %s", rec.Body.String())
	}
}

func TestStaffHandlerCreateRejectsMissingAuthMalformedAndServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	handler := NewStaffHandler(service)
	router := gin.New()
	router.POST("/school/staff", handler.Create)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/school/staff", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("missing auth status=%d calls=%d", rec.Code, service.calls)
	}

	router = gin.New()
	router.POST("/school/staff", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, handler.Create)
	req := httptest.NewRequest(http.MethodPost, "/school/staff", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || service.calls != 0 {
		t.Fatalf("malformed body status=%d calls=%d", rec.Code, service.calls)
	}

    req = httptest.NewRequest(http.MethodPost, "/school/staff",
        strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123"}`))
    req.Header.Set("Content-Type", "application/json")
    rec = httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    if rec.Code != http.StatusBadRequest || service.calls != 0 {
        t.Fatalf("missing employee profile status=%d calls=%d body=%s", rec.Code, service.calls, rec.Body.String())
    }

	for _, test := range []struct {
		err error
		want int
	}{
		{err: services.ErrInvalidStaffAccount, want: http.StatusBadRequest},
		{err: services.ErrInvalidPassword, want: http.StatusBadRequest},
		{err: services.ErrCompromisedPassword, want: http.StatusBadRequest},
		{err: services.ErrPasswordBreachCheckUnavailable, want: http.StatusServiceUnavailable},
		{err: repository.ErrUsernameTaken, want: http.StatusConflict},
        {err: repository.ErrEmployeeCodeTaken, want: http.StatusConflict},
		{err: repository.ErrRolePermissionDenied, want: http.StatusForbidden},
		{err: errors.New("internal"), want: http.StatusInternalServerError},
	} {
		service.err = test.err
		req = httptest.NewRequest(http.MethodPost, "/school/staff",
			strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123","full_name":"Rahul Sharma","employee_code":"EMP-001","designation":"Teacher","mobile_number":"9876543210"}`))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != test.want {
			t.Fatalf("error %v: status=%d want=%d body=%s", test.err, rec.Code, test.want, rec.Body.String())
		}
	}
}

func TestStaffHandlerListUsesAuthenticatedSchoolAndReturnsPrincipalIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.GET("/school/staff", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).List)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/school/staff", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"principal_id":27`) ||
		!strings.Contains(rec.Body.String(), "teacher.one") || strings.Contains(rec.Body.String(), "PasswordHash") || strings.Contains(rec.Body.String(), "private-hash-marker") {
		t.Fatalf("unexpected staff listing response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandlerDirectoryPageForwardsAuthenticatedScopeAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.GET("/school/staff/directory", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).ListDirectoryPage)
	req := httptest.NewRequest(http.MethodGet, "/school/staff/directory?search=teacher&employment_status=active&account_status=enabled&profile_status=complete&limit=25&offset=50", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || service.directoryActorID != 31 || service.directorySchoolID != 8 {
		t.Fatalf("directory request status=%d actor=%d school=%d body=%s", rec.Code, service.directoryActorID, service.directorySchoolID, rec.Body.String())
	}
	want := models.StaffDirectoryFilter{Search:"teacher", EmploymentStatus:"active", AccountStatus:"enabled", ProfileStatus:"complete", Limit:25, Offset:50}
	if service.directoryFilter != want || !strings.Contains(rec.Body.String(), `"total":51`) || !strings.Contains(rec.Body.String(), `"offset":50`) {
		t.Fatalf("directory filter/metadata mismatch: got=%+v body=%s", service.directoryFilter, rec.Body.String())
	}
}

func TestStaffHandlerDirectoryPageRejectsMalformedPaginationBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.GET("/school/staff/directory", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).ListDirectoryPage)
	for _, target := range []string{"/school/staff/directory?limit=abc", "/school/staff/directory?offset=1.5"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusBadRequest || service.directoryFilter.Limit != 0 {
			t.Fatalf("target=%s status=%d filter=%+v body=%s", target, rec.Code, service.directoryFilter, rec.Body.String())
		}
	}
}

func TestStaffHandlerDisableRequiresAuthenticatedPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.DELETE("/school/staff/:principalID", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Disable)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/school/staff/42", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d, body=%s", rec.Code, rec.Body.String())
	}
}


func TestStaffHandlerUpdateAndEnableUseAuthenticatedScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "new.teacher", Status: "active",
	}}
	handler := NewStaffHandler(service)
	auth := func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}

	router := gin.New()
	router.PATCH("/school/staff/:principalID", auth, handler.Update)
	req := httptest.NewRequest(http.MethodPatch, "/school/staff/27", strings.NewReader(`{"username":"new.teacher"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "new.teacher") {
		t.Fatalf("unexpected staff update response: status=%d body=%s", rec.Code, rec.Body.String())
	}

	router = gin.New()
	router.POST("/school/staff/:principalID/enable", auth, handler.Enable)
	req = httptest.NewRequest(http.MethodPost, "/school/staff/27/enable",
		strings.NewReader(`{"temporary_password":"A-Strong-Temporary-Pass-123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "password change is required") {
		t.Fatalf("unexpected staff enable response: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandlerUpdateProfileParsesAndPassesCompleteProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one",
		EmployeeCode: "EMP-001", FullName: "Teacher One", Designation: "Class Teacher",
		MobileNumber: "9876543210", Email: "teacher@example.com",
		EmploymentStatus: "inactive", ProfileComplete: true, Status: "disabled",
	}}
	router := gin.New()
	router.PATCH("/school/staff/:principalID", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Update)

	req := httptest.NewRequest(http.MethodPatch, "/school/staff/27", strings.NewReader(
		`{"username":"teacher.one","employee_code":"EMP-001","full_name":"Teacher One","designation":"Class Teacher","mobile_number":"9876543210","email":"teacher@example.com","joining_date":"2026-04-01","employment_status":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || service.profileCalls != 1 ||
		service.actorID != 31 || service.schoolID != 8 || service.profileTargetID != 27 {
		t.Fatalf("profile update status=%d calls=%d service=%+v body=%s", rec.Code, service.profileCalls, service, rec.Body.String())
	}
	if service.profile.EmployeeCode != "EMP-001" || service.profile.FullName != "Teacher One" ||
		service.profile.Designation != "Class Teacher" || service.profile.MobileNumber != "9876543210" ||
		service.profile.Email != "teacher@example.com" || service.profile.EmploymentStatus != "inactive" ||
		service.profile.JoiningDate == nil || service.profile.JoiningDate.Format("2006-01-02") != "2026-04-01" {
		t.Fatalf("profile payload was not passed intact: %+v", service.profile)
	}
	if !strings.Contains(rec.Body.String(), `"employment_status":"inactive"`) ||
		!strings.Contains(rec.Body.String(), `"joining_date"`) ||
		strings.Contains(rec.Body.String(), "PasswordHash") {
		t.Fatalf("unexpected profile update response: %s", rec.Body.String())
	}
}

func TestStaffHandlerUpdateProfileDoesNotExposePasswordHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one",
		PasswordHash: "private-hash-marker", EmployeeCode: "EMP-001",
		FullName: "Teacher One", Designation: "Class Teacher",
		MobileNumber: "9876543210", EmploymentStatus: "active",
	}}
	router := gin.New()
	router.PATCH("/school/staff/:principalID", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Update)
	req := httptest.NewRequest(http.MethodPatch, "/school/staff/27", strings.NewReader(
		`{"employee_code":"EMP-001","full_name":"Teacher One","designation":"Class Teacher","mobile_number":"9876543210","employment_status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || service.profileCalls != 1 {
		t.Fatalf("profile update status=%d calls=%d body=%s", rec.Code, service.profileCalls, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "private-hash-marker") || strings.Contains(rec.Body.String(), "PasswordHash") {
		t.Fatalf("staff profile update response leaked password hash: %s", rec.Body.String())
	}
}

func TestStaffHandlerUpdateProfileRejectsMalformedJoiningDateBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{user: &models.User{}}
	router := gin.New()
	router.PATCH("/school/staff/:principalID", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Update)
	req := httptest.NewRequest(http.MethodPatch, "/school/staff/27", strings.NewReader(
		`{"employee_code":"EMP-001","full_name":"Teacher One","designation":"Class Teacher","mobile_number":"9876543210","email":"teacher@example.com","joining_date":"2026-02-30","employment_status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || service.profileCalls != 0 {
		t.Fatalf("malformed joining date status=%d service calls=%d body=%s", rec.Code, service.profileCalls, rec.Body.String())
	}
}

func TestStaffHandlerEnableMapsPasswordAndNotFoundErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.POST("/school/staff/:principalID/enable", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Enable)

	for _, test := range []struct {
		err error
		want int
	}{
		{err: services.ErrInvalidPassword, want: http.StatusBadRequest},
		{err: repository.ErrStaffProfileIncomplete, want: http.StatusConflict},
		{err: repository.ErrStaffProfileInactive, want: http.StatusConflict},
		{err: repository.ErrUserNotFound, want: http.StatusNotFound},
		{err: repository.ErrRolePermissionDenied, want: http.StatusForbidden},
		{err: errors.New("internal"), want: http.StatusInternalServerError},
	} {
		service.err = test.err
		req := httptest.NewRequest(http.MethodPost, "/school/staff/27/enable",
			strings.NewReader(`{"temporary_password":"A-Strong-Temporary-Pass-123"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != test.want {
			t.Fatalf("error %v: status=%d want=%d body=%s", test.err, rec.Code, test.want, rec.Body.String())
		}
	}
}

func TestParseStaffJoiningDate(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantValid bool
	}{
		{name: "empty optional date", value: "", wantValid: true},
		{name: "valid date", value: "2026-11-23", wantValid: true},
		{name: "year one", value: "0001-01-01", wantValid: true},
		{name: "year zero", value: "0000-01-01"},
		{name: "invalid calendar date", value: "2026-02-30"},
		{name: "non canonical date", value: "2026-1-01"},
		{name: "whitespace around date", value: " 2026-11-23 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := parseStaffJoiningDate(tt.value)
			if valid != tt.wantValid {
				t.Fatalf("valid=%v want=%v", valid, tt.wantValid)
			}
			if valid && strings.TrimSpace(tt.value) != "" {
				if got == nil || got.Format("2006-01-02") != tt.value {
					t.Fatalf("unexpected parsed joining date: %#v", got)
				}
			}
			if valid && strings.TrimSpace(tt.value) == "" && got != nil {
				t.Fatalf("empty optional date should remain nil: %#v", got)
			}
		})
	}
}


func TestStaffHandlerCreateRejectsNilUserWithoutError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.POST("/school/staff", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Create)
	req := httptest.NewRequest(http.MethodPost, "/school/staff", strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123","full_name":"Rahul Sharma","employee_code":"EMP-001","designation":"Teacher","mobile_number":"9876543210"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "Unable to create staff account") {
		t.Fatalf("nil successful create result must return a controlled 500: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStaffHandlerUpdateRejectsNilUserWithoutError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffAccountService{}
	router := gin.New()
	router.PUT("/school/staff/:principalID", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Update)
	req := httptest.NewRequest(http.MethodPut, "/school/staff/27", strings.NewReader(`{"username":"teacher.updated"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "Unable to update staff account") {
		t.Fatalf("nil successful update result must return a controlled 500: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
