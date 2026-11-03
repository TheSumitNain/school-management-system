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
}

func (f *fakeStaffAccountService) Create(_ context.Context, actorID, schoolID int64, username, password string) (*models.User, error) {
	f.calls++
	f.actorID, f.schoolID, f.username, f.password = actorID, schoolID, username, password
	return f.user, f.err
}

func (f *fakeStaffAccountService) List(context.Context, int64, int64) ([]models.User, error) {
	return []models.User{{ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one", Status: "active", MustChangePassword: true}}, f.err
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
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one",
		Status: "active", MustChangePassword: true,
	}}
	router := gin.New()
	router.POST("/school/staff", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, NewStaffHandler(service).Create)
	req := httptest.NewRequest(http.MethodPost, "/school/staff",
		strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || service.calls != 1 ||
		service.actorID != 31 || service.schoolID != 8 || service.username != "teacher.one" {
		t.Fatalf("unexpected staff create response: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "A-Strong-Initial-Pass-123") || strings.Contains(rec.Body.String(), "PasswordHash") {
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

	for _, test := range []struct {
		err error
		want int
	}{
		{err: services.ErrInvalidStaffAccount, want: http.StatusBadRequest},
		{err: services.ErrInvalidPassword, want: http.StatusBadRequest},
		{err: repository.ErrUsernameTaken, want: http.StatusConflict},
		{err: repository.ErrRolePermissionDenied, want: http.StatusForbidden},
		{err: errors.New("internal"), want: http.StatusInternalServerError},
	} {
		service.err = test.err
		req = httptest.NewRequest(http.MethodPost, "/school/staff",
			strings.NewReader(`{"username":"teacher.one","initial_password":"A-Strong-Initial-Pass-123"}`))
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
		!strings.Contains(rec.Body.String(), "teacher.one") || strings.Contains(rec.Body.String(), "PasswordHash") {
		t.Fatalf("unexpected staff listing response: status=%d body=%s", rec.Code, rec.Body.String())
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
