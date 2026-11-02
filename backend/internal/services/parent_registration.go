package services

import (
	"context"
	"fmt"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

// ParentRegistrationProvisioner is the transactional account-provisioning
// boundary. It must recheck official guardian eligibility at write time.
type ParentRegistrationProvisioner interface {
	ProvisionVerifiedMobile(context.Context, string) (*repository.ParentRegistrationResult, error)
}

// ParentRegistrationService consumes a pre-account OTP before creating a
// Parent principal, account subtype, role assignment, or guardian links.
type ParentRegistrationService struct {
	otp        ParentOTPVerifier
	provisioner ParentRegistrationProvisioner
}

func NewParentRegistrationService(
	otp ParentOTPVerifier,
	provisioner ParentRegistrationProvisioner,
) (*ParentRegistrationService, error) {
	if otp == nil {
		return nil, fmt.Errorf("parent OTP verifier is required")
	}
	if provisioner == nil {
		return nil, fmt.Errorf("parent registration repository is required")
	}
	return &ParentRegistrationService{otp: otp, provisioner: provisioner}, nil
}

// Register verifies and consumes the OTP first. The repository then performs
// a fresh, atomic eligibility check and provisions every eligible guardian
// record sharing this exact mobile, including records in multiple schools.
func (s *ParentRegistrationService) Register(
	ctx context.Context, mobileE164, code string,
) (*models.AuthPrincipal, int, error) {
	if s == nil || s.otp == nil || s.provisioner == nil {
		return nil, 0, ErrParentOTPInvalid
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return nil, 0, ErrParentOTPInvalid
	}
	if err := s.otp.VerifyOTP(ctx, mobile, code); err != nil {
		return nil, 0, err
	}

	result, err := s.provisioner.ProvisionVerifiedMobile(ctx, mobile)
	if err != nil {
		return nil, 0, fmt.Errorf("provision verified Parent account: %w", err)
	}
	if result == nil || result.Principal == nil ||
		result.Principal.ID <= 0 ||
		result.Principal.Type != "parent" ||
		result.Principal.Status != "active" ||
		result.Principal.TokenVersion <= 0 ||
		!result.Principal.MobileVerified ||
		result.LinkedGuardianCount <= 0 {
		return nil, 0, fmt.Errorf("Parent registration repository returned an invalid result")
	}
	return result.Principal, result.LinkedGuardianCount, nil
}
