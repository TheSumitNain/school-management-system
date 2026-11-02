package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"school-management-system/internal/repository"
)

type parentOTPVerifierCapture struct {
	calls  int
	mobile string
	code   string
	err    error
}

func (c *parentOTPVerifierCapture) VerifyOTP(_ context.Context, mobile, code string) error {
	c.calls++
	c.mobile = mobile
	c.code = code
	return c.err
}

type parentGuardianMatchFinderCapture struct {
	calls   int
	mobile  string
	matches []repository.ParentGuardianMatch
	err     error
}

func (c *parentGuardianMatchFinderCapture) FindPortalEligibleGuardiansByMobile(
	_ context.Context, mobile string,
) ([]repository.ParentGuardianMatch, error) {
	c.calls++
	c.mobile = mobile
	return c.matches, c.err
}

func TestParentGuardianMatchingServiceVerifiesOTPBeforeLookup(t *testing.T) {
	expected := []repository.ParentGuardianMatch{
		{GuardianID: 11, SchoolID: 3, GuardianName: "Guardian One", ActiveStudentCount: 2},
		{GuardianID: 29, SchoolID: 8, GuardianName: "Guardian Two", ActiveStudentCount: 1},
	}
	otp := &parentOTPVerifierCapture{}
	finder := &parentGuardianMatchFinderCapture{matches: expected}
	service, err := NewParentGuardianMatchingService(otp, finder)
	if err != nil {
		t.Fatalf("create matching service: %v", err)
	}

	got, err := service.VerifyMobileAndMatchGuardians(context.Background(), " +919876543210 ", "123456")
	if err != nil {
		t.Fatalf("verify and match: %v", err)
	}
	if otp.calls != 1 || otp.mobile != "+919876543210" || otp.code != "123456" {
		t.Fatalf("unexpected OTP verification: %+v", otp)
	}
	if finder.calls != 1 || finder.mobile != "+919876543210" {
		t.Fatalf("unexpected guardian lookup: %+v", finder)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("matches = %#v, want %#v", got, expected)
	}
}

func TestParentGuardianMatchingServiceStopsWhenOTPIsInvalid(t *testing.T) {
	otp := &parentOTPVerifierCapture{err: ErrParentOTPInvalid}
	finder := &parentGuardianMatchFinderCapture{}
	service, err := NewParentGuardianMatchingService(otp, finder)
	if err != nil {
		t.Fatalf("create matching service: %v", err)
	}

	if _, err := service.VerifyMobileAndMatchGuardians(context.Background(), "+919876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if finder.calls != 0 {
		t.Fatalf("guardian lookup ran %d times after invalid OTP", finder.calls)
	}
}

func TestParentGuardianMatchingServiceRejectsInvalidMobileBeforeOTP(t *testing.T) {
	otp := &parentOTPVerifierCapture{}
	finder := &parentGuardianMatchFinderCapture{}
	service, err := NewParentGuardianMatchingService(otp, finder)
	if err != nil {
		t.Fatalf("create matching service: %v", err)
	}

	if _, err := service.VerifyMobileAndMatchGuardians(context.Background(), "9876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if otp.calls != 0 || finder.calls != 0 {
		t.Fatalf("invalid mobile reached dependencies: otp=%d lookup=%d", otp.calls, finder.calls)
	}
}

func TestParentGuardianMatchingServiceReturnsNoMatchesWithoutProvisioning(t *testing.T) {
	otp := &parentOTPVerifierCapture{}
	finder := &parentGuardianMatchFinderCapture{matches: []repository.ParentGuardianMatch{}}
	service, err := NewParentGuardianMatchingService(otp, finder)
	if err != nil {
		t.Fatalf("create matching service: %v", err)
	}

	got, err := service.VerifyMobileAndMatchGuardians(context.Background(), "+919876543210", "123456")
	if err != nil {
		t.Fatalf("empty official match should not be an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %#v", got)
	}
}
