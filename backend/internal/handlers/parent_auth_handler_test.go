package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/models"

	"github.com/gin-gonic/gin"
)

type fakeParentAuthenticationService struct {
	principal       *models.AuthPrincipal
	verifyErr       error
	requestErr      error
	requestedMobile string
	requestedIP     string
	requestCalls    int
	verifyMobile    string
	verifyCode      string
	verifyCalls     int
}

func (f *fakeParentAuthenticationService) RequestOTP(_ context.Context, mobile, clientIP string) error {
	f.requestCalls++
	f.requestedMobile = mobile
	f.requestedIP = clientIP
	return f.requestErr
}

func (f *fakeParentAuthenticationService) VerifyOTP(_ context.Context, mobile, code string) (*models.AuthPrincipal, error) {
	f.verifyCalls++
	f.verifyMobile = mobile
	f.verifyCode = code
	return f.principal, f.verifyErr
}

func parentAuthTestRouter(handler *ParentAuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/parent/otp/request", handler.RequestOTP)
	router.POST("/auth/parent/otp/verify", handler.VerifyOTP)
	return router
}

func TestParentOTPRequestDoesNotEnumerateAccounts(t *testing.T) {
	eligible := &fakeParentAuthenticationService{}
	ineligible := &fakeParentAuthenticationService{requestErr: errors.New("account lookup or delivery failed")}

	eligibleRecorder := httptest.NewRecorder()
	eligibleRequest := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/request",
		strings.NewReader(`{"mobile_e164":"+919876543210"}`))
	eligibleRequest.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(eligible, nil, nil)).ServeHTTP(eligibleRecorder, eligibleRequest)

	ineligibleRecorder := httptest.NewRecorder()
	ineligibleRequest := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/request",
		strings.NewReader(`{"mobile_e164":"+919876543210"}`))
	ineligibleRequest.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(ineligible, nil, nil)).ServeHTTP(ineligibleRecorder, ineligibleRequest)

	if eligibleRecorder.Code != http.StatusAccepted || ineligibleRecorder.Code != http.StatusAccepted {
		t.Fatalf("both cases must return 202; eligible=%d ineligible=%d",
			eligibleRecorder.Code, ineligibleRecorder.Code)
	}
	if eligibleRecorder.Body.String() != ineligibleRecorder.Body.String() {
		t.Fatalf("request responses disclose account/delivery state: eligible=%q ineligible=%q",
			eligibleRecorder.Body.String(), ineligibleRecorder.Body.String())
	}
	if eligible.requestCalls != 1 || ineligible.requestCalls != 1 {
		t.Fatalf("expected both requests to reach service: eligible=%d ineligible=%d",
			eligible.requestCalls, ineligible.requestCalls)
	}
	if eligible.requestedMobile != "+919876543210" || eligible.requestedIP == "" {
		t.Fatalf("service did not receive normalized mobile and client IP: %+v", eligible)
	}
}

func TestParentOTPRequestRejectsInvalidMobile(t *testing.T) {
	service := &fakeParentAuthenticationService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/request",
		strings.NewReader(`{"mobile_e164":"9876543210"}`))
	request.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(service, nil, nil)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if service.requestCalls != 0 {
		t.Fatal("invalid mobile must not reach authentication service")
	}
}

func TestParentOTPVerifyIssuesPrincipalSession(t *testing.T) {
	service := &fakeParentAuthenticationService{
		principal: &models.AuthPrincipal{
			ID: 42, Type: "parent", Status: "active", TokenVersion: 3, MobileVerified: true,
		},
	}
	sessions := &fakeSessionService{refreshToken: "parent-refresh"}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/verify",
		strings.NewReader(`{"mobile_e164":"+919876543210","code":"012345"}`))
	request.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(service, fakeTokenService{token: "parent-access"}, sessions)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if service.verifyCalls != 1 || service.verifyMobile != "+919876543210" || service.verifyCode != "012345" {
		t.Fatalf("unexpected verification arguments: %+v", service)
	}
	if sessions.issuedUserID != 42 {
		t.Fatalf("session issued for principal %d, want 42", sessions.issuedUserID)
	}
	body := recorder.Body.String()
	for _, expected := range []string{`"access_token":"parent-access"`, `"refresh_token":"parent-refresh"`, `"principal_type":"parent"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, "school_id") || strings.Contains(body, "username") {
		t.Fatalf("Parent response must not expose staff-only fields: %s", body)
	}
}

func TestParentOTPVerifyRejectsInvalidChallengeWithoutSession(t *testing.T) {
	service := &fakeParentAuthenticationService{verifyErr: errors.New("challenge unavailable")}
	sessions := &fakeSessionService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/verify",
		strings.NewReader(`{"mobile_e164":"+919876543210","code":"123456"}`))
	request.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(service, fakeTokenService{token: "unused"}, sessions)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if sessions.issuedUserID != 0 {
		t.Fatal("invalid OTP must not issue a refresh session")
	}
	if strings.Contains(recorder.Body.String(), "challenge unavailable") {
		t.Fatal("response leaked internal authentication error")
	}
}

func TestParentOTPVerifyRejectsNonParentPrincipal(t *testing.T) {
	service := &fakeParentAuthenticationService{
		principal: &models.AuthPrincipal{
			ID: 42, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 3,
		},
	}
	sessions := &fakeSessionService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/parent/otp/verify",
		strings.NewReader(`{"mobile_e164":"+919876543210","code":"123456"}`))
	request.Header.Set("Content-Type", "application/json")
	parentAuthTestRouter(NewParentAuthHandler(service, fakeTokenService{token: "unused"}, sessions)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if sessions.issuedUserID != 0 {
		t.Fatal("non-Parent principal must not receive a session")
	}
}
