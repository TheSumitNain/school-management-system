package services

import (
	"context"
	"fmt"

	"school-management-system/internal/repository"
)

// ParentOTPVerifier is the proof-of-control boundary for parent registration.
type ParentOTPVerifier interface {
	VerifyOTP(ctx context.Context, mobileE164, code string) error
}

// ParentGuardianMatchFinder returns only official, portal-eligible guardian
// records for an exact E.164 mobile number.
type ParentGuardianMatchFinder interface {
	FindPortalEligibleGuardiansByMobile(ctx context.Context, mobileE164 string) ([]repository.ParentGuardianMatch, error)
}

// ParentGuardianMatchingService verifies mobile ownership before consulting
// school guardian records. It does not create Parent principals or links;
// those belong to the subsequent account-provisioning stage.
type ParentGuardianMatchingService struct {
	otp       ParentOTPVerifier
	guardians ParentGuardianMatchFinder
}

func NewParentGuardianMatchingService(
	otp ParentOTPVerifier,
	guardians ParentGuardianMatchFinder,
) (*ParentGuardianMatchingService, error) {
	if otp == nil {
		return nil, fmt.Errorf("parent OTP verifier is required")
	}
	if guardians == nil {
		return nil, fmt.Errorf("parent guardian match repository is required")
	}
	return &ParentGuardianMatchingService{otp: otp, guardians: guardians}, nil
}

// VerifyMobileAndMatchGuardians consumes a valid OTP before looking up official
// guardian records. An empty result is a valid "no eligible record" outcome;
// it never falls back to name-only or unverified contact matching.
func (s *ParentGuardianMatchingService) VerifyMobileAndMatchGuardians(
	ctx context.Context, mobileE164, code string,
) ([]repository.ParentGuardianMatch, error) {
	if s == nil || s.otp == nil || s.guardians == nil {
		return nil, ErrParentOTPInvalid
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return nil, ErrParentOTPInvalid
	}
	if err := s.otp.VerifyOTP(ctx, mobile, code); err != nil {
		return nil, err
	}
	matches, err := s.guardians.FindPortalEligibleGuardiansByMobile(ctx, mobile)
	if err != nil {
		return nil, fmt.Errorf("match verified parent mobile to official guardians: %w", err)
	}
	return matches, nil
}
