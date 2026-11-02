package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGenerateParentOTP(t *testing.T) {
	for i := 0; i < 100; i++ {
		code, err := GenerateParentOTP()
		if err != nil {
			t.Fatalf("GenerateParentOTP() error = %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("OTP length = %d, want 6", len(code))
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("OTP %q contains a non-decimal character", code)
			}
		}
	}
}

func TestNormalizeParentMobileRequiresCanonicalE164(t *testing.T) {
	valid := []string{"+919876543210", "+14155552671"}
	for _, value := range valid {
		got, err := NormalizeParentMobile(value)
		if err != nil || got != value {
			t.Fatalf("NormalizeParentMobile(%q) = %q, %v", value, got, err)
		}
	}

	invalid := []string{"9876543210", "919876543210", "+0123456789", "+123", "+1234567890123456", ""}
	for _, value := range invalid {
		if _, err := NormalizeParentMobile(value); err == nil {
			t.Errorf("NormalizeParentMobile(%q) unexpectedly succeeded", value)
		}
	}
}

func TestParentOTPHMACsAreBoundAndConstantTimeVerified(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	mobileHash, err := HashParentMobile(key, "+919876543210")
	if err != nil {
		t.Fatalf("HashParentMobile() error = %v", err)
	}
	if len(mobileHash) != 64 {
		t.Fatalf("mobile hash length = %d, want 64", len(mobileHash))
	}

	otpHash, err := HashParentOTP(key, mobileHash, "042719")
	if err != nil {
		t.Fatalf("HashParentOTP() error = %v", err)
	}
	if !VerifyParentOTPHash(key, mobileHash, "042719", otpHash) {
		t.Fatal("expected correct OTP to verify")
	}
	if VerifyParentOTPHash(key, mobileHash, "042718", otpHash) {
		t.Fatal("wrong OTP verified")
	}

	otherMobileHash, err := HashParentMobile(key, "+919876543211")
	if err != nil {
		t.Fatalf("HashParentMobile(other) error = %v", err)
	}
	if VerifyParentOTPHash(key, otherMobileHash, "042719", otpHash) {
		t.Fatal("OTP verified against a different mobile")
	}
	if VerifyParentOTPHash([]byte("short"), mobileHash, "042719", otpHash) {
		t.Fatal("OTP verified with an undersized HMAC key")
	}
}

func TestParentOTPHashRejectsInvalidInputs(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	if _, err := HashParentMobile(key, "9876543210"); err == nil {
		t.Fatal("expected non-E.164 mobile to be rejected")
	}
	if _, err := HashParentOTP(key, strings.Repeat("a", 64), "12345"); err == nil {
		t.Fatal("expected non-six-digit OTP to be rejected")
	}
	if _, err := HashParentOTP([]byte("short"), strings.Repeat("a", 64), "123456"); err == nil {
		t.Fatal("expected undersized key to be rejected")
	}
	if VerifyParentOTPHash(key, strings.Repeat("a", 64), "123456", "not-a-hash") {
		t.Fatal("malformed stored hash unexpectedly verified")
	}
}


type parentOTPChallengeCapture struct {
	mobileHash string
	ipHash     string
	otpHash    string
	expiresAt  time.Time
	calls         int
	err           error
	invalidations []int64
	invalidateErr  error
}

func (c *parentOTPChallengeCapture) InvalidatePreAccountChallenge(_ context.Context, challengeID int64) error {
	c.invalidations = append(c.invalidations, challengeID)
	return c.invalidateErr
}

func (c *parentOTPChallengeCapture) CreatePreAccountChallenge(
	_ context.Context, mobileHash, ipHash, otpHash string, expiresAt time.Time,
) (int64, error) {
	c.calls++
	c.mobileHash, c.ipHash, c.otpHash, c.expiresAt = mobileHash, ipHash, otpHash, expiresAt
	if c.err != nil {
		return 0, c.err
	}
	return 42, nil
}

// This fake is test-only and intentionally retains the code in memory so the
// test can verify the delivery boundary without exposing it through an API.
type parentOTPDeliveryCapture struct {
	mobile string
	code   string
	err    error
}

func (d *parentOTPDeliveryCapture) SendParentOTP(_ context.Context, mobile, code string) error {
	d.mobile, d.code = mobile, code
	return d.err
}

func TestParentOTPServiceRequestOTP(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	repo := &parentOTPChallengeCapture{}
	delivery := &parentOTPDeliveryCapture{}
	service, err := NewParentOTPService(repo, delivery, key)
	if err != nil {
		t.Fatalf("NewParentOTPService() error = %v", err)
	}

	if err := service.RequestOTP(context.Background(), "+919876543210", "192.0.2.10"); err != nil {
		t.Fatalf("RequestOTP() error = %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("challenge repository calls = %d, want 1", repo.calls)
	}
	if delivery.mobile != "+919876543210" || len(delivery.code) != 6 {
		t.Fatalf("delivery received mobile %q and code length %d", delivery.mobile, len(delivery.code))
	}
	if repo.mobileHash == "" || repo.ipHash == "" || repo.otpHash == "" {
		t.Fatal("repository did not receive all required HMAC hashes")
	}
	if repo.mobileHash == "+919876543210" || repo.ipHash == "192.0.2.10" || repo.otpHash == delivery.code {
		t.Fatal("raw mobile, IP, or OTP was passed to persistent storage")
	}
	if !VerifyParentOTPHash(key, repo.mobileHash, delivery.code, repo.otpHash) {
		t.Fatal("persisted OTP hash does not match the code delivered to provider")
	}
	remaining := time.Until(repo.expiresAt)
	if remaining <= 0 || remaining > 5*time.Minute {
		t.Fatalf("challenge expiry is outside the five-minute window: %s", remaining)
	}
}

func TestParentOTPServiceFailsClosed(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	repo := &parentOTPChallengeCapture{}
	if _, err := NewParentOTPService(repo, nil, key); !errors.Is(err, ErrParentOTPDeliveryUnavailable) {
		t.Fatalf("constructor without provider error = %v", err)
	}
	if _, err := NewParentOTPService(repo, &parentOTPDeliveryCapture{}, []byte("short")); err == nil {
		t.Fatal("constructor accepted an undersized HMAC key")
	}

	service, err := NewParentOTPService(repo, &parentOTPDeliveryCapture{}, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RequestOTP(context.Background(), "+919876543210", "not-an-ip"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("RequestOTP() invalid IP error = %v", err)
	}
	if repo.calls != 0 {
		t.Fatal("invalid IP reached persistent challenge repository")
	}
}

func TestParentOTPServicePropagatesRateLimitAndHidesDeliveryFailure(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	rateErr := errors.New("rate limited")
	repo := &parentOTPChallengeCapture{err: rateErr}
	delivery := &parentOTPDeliveryCapture{}
	service, err := NewParentOTPService(repo, delivery, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RequestOTP(context.Background(), "+919876543210", "192.0.2.11"); !errors.Is(err, rateErr) {
		t.Fatalf("RequestOTP() rate-limit error = %v", err)
	}
	if delivery.code != "" {
		t.Fatal("provider was called after repository rejected the rate-limit reservation")
	}

	repo = &parentOTPChallengeCapture{}
	delivery = &parentOTPDeliveryCapture{err: errors.New("provider private failure detail")}
	service, err = NewParentOTPService(repo, delivery, key)
	if err != nil {
		t.Fatal(err)
	}
	err = service.RequestOTP(context.Background(), "+919876543210", "192.0.2.12")
	if !errors.Is(err, ErrParentOTPDeliveryUnavailable) {
		t.Fatalf("RequestOTP() delivery error = %v", err)
	}
	if strings.Contains(err.Error(), "provider private failure detail") {
		t.Fatal("provider error detail escaped the service boundary")
	}
	if len(repo.invalidations) != 1 || repo.invalidations[0] != 42 {
		t.Fatalf("failed delivery invalidations = %v, want [42]", repo.invalidations)
	}
}

func TestHashParentIPCanonicalizesAddressesAndSeparatesDomains(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	v4, err := HashParentIP(key, "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	v4WithWhitespace, err := HashParentIP(key, " 192.0.2.1 ")
	if err != nil {
		t.Fatal(err)
	}
	if v4 != v4WithWhitespace {
		t.Fatal("equivalent IPv4 text produced different rate-limit hashes")
	}
	mobileHash, err := HashParentMobile(key, "+919876543210")
	if err != nil {
		t.Fatal(err)
	}
	if v4 == mobileHash {
		t.Fatal("IP and mobile HMAC domains unexpectedly collided")
	}
	if _, err := HashParentIP(key, ""); err == nil {
		t.Fatal("empty IP address was accepted")
	}
}


func TestParentOTPServiceReportsCleanupFailureWithoutLeakingDetails(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	repo := &parentOTPChallengeCapture{invalidateErr: errors.New("db-detail")}
	delivery := &parentOTPDeliveryCapture{err: errors.New("provider-detail")}
	service, err := NewParentOTPService(repo, delivery, key)
	if err != nil {
		t.Fatal(err)
	}

	err = service.RequestOTP(context.Background(), "+919876543210", "192.0.2.13")
	if !errors.Is(err, ErrParentOTPDeliveryUnavailable) ||
		!errors.Is(err, ErrParentOTPDeliveryCleanupFailed) {
		t.Fatalf("RequestOTP() error = %v, want delivery and cleanup sentinels", err)
	}
	if strings.Contains(err.Error(), "db-detail") || strings.Contains(err.Error(), "provider-detail") {
		t.Fatal("internal provider or database detail escaped the service boundary")
	}
	if len(repo.invalidations) != 1 || repo.invalidations[0] != 42 {
		t.Fatalf("invalidations = %v, want [42]", repo.invalidations)
	}
}
