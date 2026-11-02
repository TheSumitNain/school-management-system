package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

type parentRegistrationProvisionerCapture struct {
	calls  int
	mobile string
	result *repository.ParentRegistrationResult
	err    error
}

func (c *parentRegistrationProvisionerCapture) ProvisionVerifiedMobile(
	_ context.Context, mobile string,
) (*repository.ParentRegistrationResult, error) {
	c.calls++
	c.mobile = mobile
	return c.result, c.err
}

func TestParentRegistrationVerifiesOTPBeforeProvisioning(t *testing.T) {
	otp := &parentOTPVerifierCapture{}
	provisioner := &parentRegistrationProvisionerCapture{
		result: &repository.ParentRegistrationResult{
			Principal: &models.AuthPrincipal{
				ID: 91, Type: "parent", Status: "active", TokenVersion: 1, MobileVerified: true,
			},
			LinkedGuardianCount: 2,
		},
	}
	service, err := NewParentRegistrationService(otp, provisioner)
	if err != nil {
		t.Fatalf("create registration service: %v", err)
	}

	principal, linked, err := service.Register(context.Background(), " +919876543210 ", "123456")
	if err != nil {
		t.Fatalf("register verified Parent: %v", err)
	}
	if otp.calls != 1 || otp.mobile != "+919876543210" || otp.code != "123456" {
		t.Fatalf("unexpected OTP verification: %+v", otp)
	}
	if provisioner.calls != 1 || provisioner.mobile != "+919876543210" {
		t.Fatalf("unexpected provisioning call: %+v", provisioner)
	}
	if principal.ID != 91 || linked != 2 {
		t.Fatalf("registration result principal=%+v linked=%d", principal, linked)
	}
}

func TestParentRegistrationDoesNotProvisionWhenOTPIsInvalid(t *testing.T) {
	otp := &parentOTPVerifierCapture{err: ErrParentOTPInvalid}
	provisioner := &parentRegistrationProvisionerCapture{}
	service, err := NewParentRegistrationService(otp, provisioner)
	if err != nil {
		t.Fatalf("create registration service: %v", err)
	}

	if _, _, err := service.Register(context.Background(), "+919876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if provisioner.calls != 0 {
		t.Fatalf("provisioning ran %d times after invalid OTP", provisioner.calls)
	}
}

func TestParentRegistrationRejectsInvalidMobileBeforeOTP(t *testing.T) {
	otp := &parentOTPVerifierCapture{}
	provisioner := &parentRegistrationProvisionerCapture{}
	service, err := NewParentRegistrationService(otp, provisioner)
	if err != nil {
		t.Fatalf("create registration service: %v", err)
	}

	if _, _, err := service.Register(context.Background(), "9876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP")
	}
	if otp.calls != 0 || provisioner.calls != 0 {
		t.Fatalf("invalid mobile reached dependencies: otp=%d provisioner=%d", otp.calls, provisioner.calls)
	}
}

func TestParentRegistrationRejectsInvalidProvisioningResult(t *testing.T) {
	tests := []struct {
		name   string
		result *repository.ParentRegistrationResult
	}{
		{name: "nil result"},
		{name: "nil principal", result: &repository.ParentRegistrationResult{LinkedGuardianCount: 1}},
		{name: "wrong principal type", result: &repository.ParentRegistrationResult{
			Principal: &models.AuthPrincipal{ID: 1, Type: "staff", Status: "active", TokenVersion: 1, MobileVerified: true},
			LinkedGuardianCount: 1,
		}},
		{name: "unverified mobile", result: &repository.ParentRegistrationResult{
			Principal: &models.AuthPrincipal{ID: 1, Type: "parent", Status: "active", TokenVersion: 1},
			LinkedGuardianCount: 1,
		}},
		{name: "no linked guardians", result: &repository.ParentRegistrationResult{
			Principal: &models.AuthPrincipal{ID: 1, Type: "parent", Status: "active", TokenVersion: 1, MobileVerified: true},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewParentRegistrationService(
				&parentOTPVerifierCapture{},
				&parentRegistrationProvisionerCapture{result: tt.result},
			)
			if err != nil {
				t.Fatalf("create registration service: %v", err)
			}
			if _, _, err := service.Register(context.Background(), "+919876543210", "123456"); err == nil {
				t.Fatal("expected invalid provisioning result error")
			}
		})
	}
}
