package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeStaffPasswordService struct {
	changeErr error
	resetErr error
	changePrincipalID int64
	changeCurrent string
	changeNext string
	resetActorID int64
	resetSchoolID int64
	resetTargetID int64
	resetNext string
}

func (f *fakeStaffPasswordService) ChangePassword(_ context.Context, principalID int64, current, next string) error {
	f.changePrincipalID, f.changeCurrent, f.changeNext = principalID, current, next
	return f.changeErr
}

func (f *fakeStaffPasswordService) ResetPassword(_ context.Context, actorID, schoolID, targetID int64, next string) error {
	f.resetActorID, f.resetSchoolID, f.resetTargetID, f.resetNext = actorID, schoolID, targetID, next
	return f.resetErr
}

func TestChangeStaffPasswordHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffPasswordService{}
	handler := &AuthHandler{staffPasswordService: service}
	router := gin.New()
	router.POST("/auth/password/change", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(21))
		c.Set("auth_principal_type", "staff")
		c.Next()
	}, handler.ChangeStaffPassword)

	req := httptest.NewRequest(http.MethodPost, "/auth/password/change",
		strings.NewReader(`{"current_password":"OldStrong!Password123","new_password":"NewStrong!Password456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if service.changePrincipalID != 21 || service.changeCurrent != "OldStrong!Password123" ||
		service.changeNext != "NewStrong!Password456" {
		t.Fatalf("unexpected password-change service input: %+v", service)
	}
}

func TestResetStaffPasswordHandlerUsesAuthenticatedSchoolScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffPasswordService{}
	handler := &AuthHandler{staffPasswordService: service}
	router := gin.New()
	router.POST("/school/staff/:principalID/password/reset", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_principal_type", "staff")
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, handler.ResetStaffPassword)

	req := httptest.NewRequest(http.MethodPost, "/school/staff/44/password/reset",
		strings.NewReader(`{"new_password":"Temporary!Password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if service.resetActorID != 31 || service.resetSchoolID != 8 ||
		service.resetTargetID != 44 || service.resetNext != "Temporary!Password123" {
		t.Fatalf("unexpected scoped reset service input: %+v", service)
	}
}

func TestResetStaffPasswordHandlerMapsOutOfSchoolTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeStaffPasswordService{resetErr: errors.New("not found")}
	handler := &AuthHandler{staffPasswordService: service}
	router := gin.New()
	router.POST("/school/staff/:principalID/password/reset", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, handler.ResetStaffPassword)

	req := httptest.NewRequest(http.MethodPost, "/school/staff/44/password/reset",
		strings.NewReader(`{"new_password":"Temporary!Password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for unexpected repository error; body=%s", rec.Code, rec.Body.String())
	}
}
