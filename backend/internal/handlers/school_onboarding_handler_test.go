package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeSchoolOnboardingService struct {
	result *services.SchoolOnboardingResult
	err error
	actorID int64
	input services.SchoolOnboardingInput
	calls int
	schools []repository.PlatformSchoolRecord
	listErr error
}

func (f *fakeSchoolOnboardingService) ListPlatformSchools(_ context.Context, actorID int64) ([]repository.PlatformSchoolRecord, error) { f.actorID = actorID; return f.schools, f.listErr }

func (f *fakeSchoolOnboardingService) CreateSchoolWithAdmin(_ context.Context, actorID int64, input services.SchoolOnboardingInput) (*services.SchoolOnboardingResult, error) {
	f.calls++
	f.actorID = actorID
	f.input = input
	return f.result, f.err
}

func TestSchoolOnboardingCreateSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSchoolOnboardingService{result: &services.SchoolOnboardingResult{
		SchoolID: 12, SchoolName: "Example School", SchoolCode: "EXAMPLE-12",
		SchoolStatus: "active", AdminPrincipalID: 44, AdminUsername: "school.admin",
	}}
	handler := NewSchoolOnboardingHandler(service)
	router := gin.New()
	router.POST("/platform/schools", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(7))
		c.Next()
	}, handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/platform/schools", strings.NewReader(`{
		"name":"Example School","code":"example-12",
		"school_admin_username":"School.Admin",
		"school_admin_initial_password":"LongStrong!Pass123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if service.calls != 1 || service.actorID != 7 {
		t.Fatalf("service did not receive authenticated actor: %+v", service)
	}
	if service.input.Name != "Example School" || service.input.Code != "example-12" ||
		service.input.AdminUsername != "School.Admin" {
		t.Fatalf("unexpected onboarding input: %+v", service.input)
	}
	if strings.Contains(rec.Body.String(), "LongStrong") || strings.Contains(rec.Body.String(), "school_admin_initial_password") {
		t.Fatalf("response must not contain the supplied password: %s", rec.Body.String())
	}
}

func TestSchoolOnboardingCreateRejectsMissingActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSchoolOnboardingService{}
	router := gin.New()
	router.POST("/platform/schools", NewSchoolOnboardingHandler(service).Create)

	req := httptest.NewRequest(http.MethodPost, "/platform/schools", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("expected unauthenticated request to be rejected, got %d and %d calls", rec.Code, service.calls)
	}
}

func TestSchoolOnboardingCreateMapsValidationAndConflictErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		err error
		want int
	}{
		{name: "invalid input", err: services.ErrInvalidSchoolOnboarding, want: http.StatusBadRequest},
		{name: "invalid password", err: services.ErrInvalidPassword, want: http.StatusBadRequest},
		{name: "compromised password", err: services.ErrCompromisedPassword, want: http.StatusBadRequest},
		{name: "password breach checker unavailable", err: services.ErrPasswordBreachCheckUnavailable, want: http.StatusServiceUnavailable},
		{name: "duplicate school code", err: repository.ErrSchoolCodeTaken, want: http.StatusConflict},
		{name: "duplicate username", err: repository.ErrUsernameTaken, want: http.StatusConflict},
		{name: "service failure", err: errors.New("sensitive internal database detail"), want: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeSchoolOnboardingService{err: test.err}
			router := gin.New()
			router.POST("/platform/schools", func(c *gin.Context) {
				c.Set(middleware.ContextPrincipalIDKey, int64(7))
				c.Next()
			}, NewSchoolOnboardingHandler(service).Create)
			req := httptest.NewRequest(http.MethodPost, "/platform/schools", strings.NewReader(`{"name":"X","code":"XX","school_admin_username":"abc","school_admin_initial_password":"LongStrong!Pass123"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.want {
				t.Fatalf("expected %d, got %d: %s", test.want, rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "sensitive internal database detail") {
				t.Fatalf("internal error details must not be exposed: %s", rec.Body.String())
			}
		})
	}
}


func TestSchoolOnboardingListRequiresAuthenticatedPlatformActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSchoolOnboardingService{schools: []repository.PlatformSchoolRecord{{ID: 8, Name: "Central", Code: "CENTRAL", Status: "active"}}}
	router := gin.New()
	router.GET("/platform/schools", func(c *gin.Context) { c.Set(middleware.ContextPrincipalIDKey, int64(5)); c.Next() }, NewSchoolOnboardingHandler(service).List)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/platform/schools", nil))
	if rec.Code != http.StatusOK || service.actorID != 5 || !strings.Contains(rec.Body.String(), "CENTRAL") { t.Fatalf("unexpected school listing response: %d %s", rec.Code, rec.Body.String()) }
}
