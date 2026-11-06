package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"school-management-system/internal/models"
)

// ParentPrincipalFinder performs an exact, eligibility-filtered lookup by
// verified mobile. Implementations must not return profile or child data.
type ParentPrincipalFinder interface {
	FindActiveParentByMobile(context.Context, string) (*models.AuthPrincipal, error)
}

// ParentAuthenticationPrincipalLoader reloads live principal state after OTP
// consumption and immediately before the caller issues an authenticated session.
type ParentAuthenticationPrincipalLoader interface {
	LoadPrincipalForAuthentication(context.Context, int64) (*models.AuthPrincipal, error)
}

type ParentLoginOTP interface {
	RequestOTP(context.Context, int64, string, string) error
	VerifyOTP(context.Context, int64, string, string) error
}

// ParentAuthenticationService resolves the account for an OTP challenge and
// revalidates the same identity after OTP consumption.
type ParentAuthenticationService struct {
	parents ParentPrincipalFinder
	loader  ParentAuthenticationPrincipalLoader
	otp     ParentLoginOTP
}

func NewParentAuthenticationService(
	parents ParentPrincipalFinder,
	loader ParentAuthenticationPrincipalLoader,
	otp ParentLoginOTP,
) (*ParentAuthenticationService, error) {
	if parents == nil {
		return nil, fmt.Errorf("Parent account finder is required")
	}
	if loader == nil {
		return nil, fmt.Errorf("Parent authentication principal loader is required")
	}
	if otp == nil {
		return nil, ErrParentOTPDeliveryUnavailable
	}
	return &ParentAuthenticationService{parents: parents, loader: loader, otp: otp}, nil
}

// RequestOTP deliberately treats an unknown or ineligible mobile as a
// successful no-op. HTTP callers must return the same public response whether
// an account exists or not. Operational and delivery errors remain available
// to internal callers for logging/alerting.
func (s *ParentAuthenticationService) RequestOTP(
	ctx context.Context, mobileE164, clientIP string,
) error {
	if s == nil || s.parents == nil || s.otp == nil {
		return ErrParentOTPDeliveryUnavailable
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return ErrParentOTPInvalid
	}
	principal, err := s.parents.FindActiveParentByMobile(ctx, mobile)
	if errors.Is(err, models.ErrPrincipalNotAuthenticated) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve Parent account for OTP request: %w", err)
	}
	if !validParentAuthenticationPrincipal(principal) {
		return nil
	}
	if err := s.otp.RequestOTP(ctx, principal.ID, mobile, clientIP); err != nil {
		return fmt.Errorf("request Parent login OTP: %w", err)
	}
	return nil
}

// VerifyOTP consumes a challenge for the resolved account, then performs a
// fresh repository lookup and canonical principal load. A changed, disabled,
// unverified, or otherwise ineligible identity cannot receive a session.
func (s *ParentAuthenticationService) VerifyOTP(
	ctx context.Context, mobileE164, code string,
) (*models.AuthPrincipal, error) {
	if s == nil || s.parents == nil || s.loader == nil || s.otp == nil {
		return nil, ErrParentOTPInvalid
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil || strings.TrimSpace(code) == "" {
		return nil, ErrParentOTPInvalid
	}
	before, err := s.parents.FindActiveParentByMobile(ctx, mobile)
	if err != nil || !validParentAuthenticationPrincipal(before) {
		return nil, ErrParentOTPInvalid
	}
	if err := s.otp.VerifyOTP(ctx, before.ID, mobile, strings.TrimSpace(code)); err != nil {
		return nil, err
	}

	// Re-resolve by the submitted mobile after OTP consumption to ensure the
	// mobile still maps to the same active Parent principal.
	after, err := s.parents.FindActiveParentByMobile(ctx, mobile)
	if err != nil || !validParentAuthenticationPrincipal(after) || after.ID != before.ID {
		return nil, ErrParentOTPInvalid
	}
	principal, err := s.loader.LoadPrincipalForAuthentication(ctx, before.ID)
	if err != nil || !validParentAuthenticationPrincipal(principal) {
		return nil, ErrParentOTPInvalid
	}
	return principal, nil
}

func validParentAuthenticationPrincipal(principal *models.AuthPrincipal) bool {
	return principal != nil &&
		principal.ID > 0 &&
		principal.Type == "parent" &&
		principal.Status == "active" &&
		principal.TokenVersion > 0 &&
		principal.MobileVerified
}
