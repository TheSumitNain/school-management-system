package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/models"
)

type parentFinderFake struct {
	principal *models.AuthPrincipal
	err error
	calls int
	mobiles []string
	sequence []*models.AuthPrincipal
	sequenceErrors []error
}

func (f *parentFinderFake) FindActiveParentByMobile(_ context.Context, mobile string) (*models.AuthPrincipal, error) {
	f.calls++
	f.mobiles = append(f.mobiles, mobile)
	if len(f.sequence) > 0 || len(f.sequenceErrors) > 0 {
		index := f.calls - 1
		if index < len(f.sequenceErrors) && f.sequenceErrors[index] != nil {
			return nil, f.sequenceErrors[index]
		}
		if index < len(f.sequence) {
			return f.sequence[index], nil
		}
	}
	return f.principal, f.err
}

type parentAuthLoaderFake struct {
	principal *models.AuthPrincipal
	err error
	calls int
	principalID int64
}

func (f *parentAuthLoaderFake) LoadPrincipalForAuthentication(_ context.Context, principalID int64) (*models.AuthPrincipal, error) {
	f.calls++
	f.principalID = principalID
	return f.principal, f.err
}

type parentLoginOTPFake struct {
	requestCalls int
	verifyCalls int
	principalID int64
	mobile string
	ip string
	code string
	requestErr error
	verifyErr error
}

func (f *parentLoginOTPFake) RequestOTP(_ context.Context, principalID int64, mobile, ip string) error {
	f.requestCalls++
	f.principalID, f.mobile, f.ip = principalID, mobile, ip
	return f.requestErr
}

func (f *parentLoginOTPFake) VerifyOTP(_ context.Context, principalID int64, mobile, code string) error {
	f.verifyCalls++
	f.principalID, f.mobile, f.code = principalID, mobile, code
	return f.verifyErr
}

func activeParentPrincipal(id int64) *models.AuthPrincipal {
	return &models.AuthPrincipal{
		ID: id, Type: "parent", Status: "active", TokenVersion: 1, MobileVerified: true,
	}
}

func newParentAuthenticationServiceForTest(t *testing.T, finder *parentFinderFake, loader *parentAuthLoaderFake, otp *parentLoginOTPFake) *ParentAuthenticationService {
	t.Helper()
	service, err := NewParentAuthenticationService(finder, loader, otp)
	if err != nil {
		t.Fatalf("create Parent authentication service: %v", err)
	}
	return service
}

func TestNewParentAuthenticationServiceRequiresDependencies(t *testing.T) {
	finder := &parentFinderFake{}
	loader := &parentAuthLoaderFake{}
	otp := &parentLoginOTPFake{}

	if _, err := NewParentAuthenticationService(nil, loader, otp); err == nil {
		t.Fatal("expected missing finder to be rejected")
	}
	if _, err := NewParentAuthenticationService(finder, nil, otp); err == nil {
		t.Fatal("expected missing principal loader to be rejected")
	}
	if _, err := NewParentAuthenticationService(finder, loader, nil); !errors.Is(err, ErrParentOTPDeliveryUnavailable) {
		t.Fatalf("missing OTP service error = %v, want delivery unavailable", err)
	}
}

func TestParentAuthenticationRequestOTPUsesEligibleParent(t *testing.T) {
	finder := &parentFinderFake{principal: activeParentPrincipal(17)}
	loader := &parentAuthLoaderFake{}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, loader, otp)

	if err := service.RequestOTP(context.Background(), " +919876543210 ", "203.0.113.7"); err != nil {
		t.Fatalf("request Parent OTP: %v", err)
	}
	if finder.calls != 1 || finder.mobiles[0] != "+919876543210" {
		t.Fatalf("unexpected Parent lookup: %+v", finder)
	}
	if otp.requestCalls != 1 || otp.principalID != 17 ||
		otp.mobile != "+919876543210" || otp.ip != "203.0.113.7" {
		t.Fatalf("unexpected OTP request: %+v", otp)
	}
}

func TestParentAuthenticationRequestOTPIsNoOpForUnknownOrIneligibleMobile(t *testing.T) {
	tests := []struct {
		name string
		finder *parentFinderFake
	}{
		{name: "unknown mobile", finder: &parentFinderFake{err: models.ErrPrincipalNotAuthenticated}},
		{name: "invalid returned principal", finder: &parentFinderFake{principal: &models.AuthPrincipal{ID: 17, Type: "staff", Status: "active", TokenVersion: 1}}},
		{name: "nil principal", finder: &parentFinderFake{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := &parentLoginOTPFake{}
			service := newParentAuthenticationServiceForTest(t, tt.finder, &parentAuthLoaderFake{}, otp)
			if err := service.RequestOTP(context.Background(), "+919876543210", "203.0.113.7"); err != nil {
				t.Fatalf("unknown/ineligible account should be a no-op: %v", err)
			}
			if otp.requestCalls != 0 {
				t.Fatalf("OTP sent for ineligible account: %d calls", otp.requestCalls)
			}
		})
	}
}

func TestParentAuthenticationRequestOTPPropagatesOperationalErrors(t *testing.T) {
	dbErr := errors.New("database unavailable")
	finder := &parentFinderFake{err: dbErr}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, &parentAuthLoaderFake{}, otp)

	if err := service.RequestOTP(context.Background(), "+919876543210", "203.0.113.7"); !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want repository error", err)
	}
	if otp.requestCalls != 0 {
		t.Fatal("OTP delivery attempted after account lookup failure")
	}

	finder = &parentFinderFake{principal: activeParentPrincipal(17)}
	otp = &parentLoginOTPFake{requestErr: ErrParentOTPDeliveryUnavailable}
	service = newParentAuthenticationServiceForTest(t, finder, &parentAuthLoaderFake{}, otp)
	if err := service.RequestOTP(context.Background(), "+919876543210", "203.0.113.7"); !errors.Is(err, ErrParentOTPDeliveryUnavailable) {
		t.Fatalf("delivery error = %v, want delivery unavailable", err)
	}
}

func TestParentAuthenticationVerifyOTPRevalidatesPrincipalBeforeSession(t *testing.T) {
	finder := &parentFinderFake{
		sequence: []*models.AuthPrincipal{activeParentPrincipal(17), activeParentPrincipal(17)},
	}
	loader := &parentAuthLoaderFake{principal: activeParentPrincipal(17)}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, loader, otp)

	principal, err := service.VerifyOTP(context.Background(), " +919876543210 ", "123456")
	if err != nil {
		t.Fatalf("verify Parent OTP: %v", err)
	}
	if principal.ID != 17 || principal.Type != "parent" {
		t.Fatalf("unexpected authenticated principal: %+v", principal)
	}
	if finder.calls != 2 || loader.calls != 1 || loader.principalID != 17 {
		t.Fatalf("expected lookup before OTP and revalidation after OTP: finder=%d loader=%d", finder.calls, loader.calls)
	}
	if otp.verifyCalls != 1 || otp.principalID != 17 || otp.mobile != "+919876543210" || otp.code != "123456" {
		t.Fatalf("unexpected OTP verification arguments: %+v", otp)
	}
}

func TestParentAuthenticationVerifyOTPRejectsChangedPrincipal(t *testing.T) {
	finder := &parentFinderFake{
		sequence: []*models.AuthPrincipal{activeParentPrincipal(17), activeParentPrincipal(18)},
	}
	loader := &parentAuthLoaderFake{principal: activeParentPrincipal(17)}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, loader, otp)

	if _, err := service.VerifyOTP(context.Background(), "+919876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if loader.calls != 0 {
		t.Fatal("session principal was loaded after account identity changed")
	}
}

func TestParentAuthenticationVerifyOTPRejectsDisabledAccountAfterConsumption(t *testing.T) {
	finder := &parentFinderFake{
		sequence: []*models.AuthPrincipal{activeParentPrincipal(17), activeParentPrincipal(17)},
	}
	loader := &parentAuthLoaderFake{err: models.ErrPrincipalNotAuthenticated}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, loader, otp)

	if _, err := service.VerifyOTP(context.Background(), "+919876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if loader.calls != 1 {
		t.Fatalf("expected final principal reload, got %d calls", loader.calls)
	}
}

func TestParentAuthenticationVerifyOTPRejectsMalformedInputBeforeLookup(t *testing.T) {
	finder := &parentFinderFake{principal: activeParentPrincipal(17)}
	otp := &parentLoginOTPFake{}
	service := newParentAuthenticationServiceForTest(t, finder, &parentAuthLoaderFake{}, otp)

	if _, err := service.VerifyOTP(context.Background(), "9876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("invalid mobile error = %v, want invalid OTP", err)
	}
	if _, err := service.VerifyOTP(context.Background(), "+919876543210", ""); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("empty code error = %v, want invalid OTP", err)
	}
	if finder.calls != 0 || otp.verifyCalls != 0 {
		t.Fatal("malformed input reached account lookup or OTP service")
	}
}
