package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/handlers"

	"github.com/gin-gonic/gin"
)

// Parent OTP routes must remain unavailable until a production delivery
// provider and the corresponding startup configuration are explicitly wired.
func TestParentOTPAuthenticationRoutesRemainDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(
		fakeDatabase{},
		handlers.NewAuthHandler(nil, nil, nil),
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	paths := []string{
		"/auth/parent/otp/request",
		"/auth/parent/otp/verify",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("Parent OTP route %s must remain disabled: got HTTP %d, want %d",
					path, recorder.Code, http.StatusNotFound)
			}
		})
	}
}
