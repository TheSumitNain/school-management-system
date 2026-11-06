package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type parentOTPLoginChallengeStoreFake struct {
	createCalls int
	verifyCalls int
	invalidateCalls int

	principalID int64
	mobile string
	mobileHash string
	ipHash string
	otpHash string
	expiresAt time.Time
	challengeID int64

	verifyResult bool
	createErr error
	verifyErr error
	invalidateErr error
	verifiedPrincipalID int64
	verifiedMobileHash string
	candidateOTPHash string
	invalidatedPrincipalID int64
	invalidatedChallengeID int64
}

func (f *parentOTPLoginChallengeStoreFake) CreatePrincipalChallenge(
	_ context.Context, principalID int64, mobile, mobileHash, ipHash, otpHash string, expiresAt time.Time,
) (int64, error) {
	f.createCalls++
	f.principalID, f.mobile, f.mobileHash, f.ipHash, f.otpHash, f.expiresAt =
		principalID, mobile, mobileHash, ipHash, otpHash, expiresAt
	if f.createErr != nil {
		return 0, f.createErr
	}
	if f.challengeID == 0 {
		f.challengeID = 42
	}
	return f.challengeID, nil
}

func (f *parentOTPLoginChallengeStoreFake) VerifyPrincipalChallenge(
	_ context.Context, principalID int64, mobileHash, candidateOTPHash string,
) (bool, error) {
	f.verifyCalls++
	f.verifiedPrincipalID, f.verifiedMobileHash, f.candidateOTPHash =
		principalID, mobileHash, candidateOTPHash
	return f.verifyResult, f.verifyErr
}

func (f *parentOTPLoginChallengeStoreFake) InvalidatePrincipalChallenge(
	_ context.Context, principalID, challengeID int64,
) error {
	f.invalidateCalls++
	f.invalidatedPrincipalID, f.invalidatedChallengeID = principalID, challengeID
	return f.invalidateErr
}

type parentOTPLoginDeliveryFake struct {
	calls int
	mobile string
	code string
	err error
}

func (f *parentOTPLoginDeliveryFake) SendParentOTP(_ context.Context, mobile, code string) error {
	f.calls++
	f.mobile, f.code = mobile, code
	return f.err
}

var parentOTPLoginTestKey = []byte(strings.Repeat("k", 32))

func newParentOTPLoginTestService(t *testing.T, store *parentOTPLoginChallengeStoreFake, delivery *parentOTPLoginDeliveryFake) *ParentOTPLoginService {
	t.Helper()
	service, err := NewParentOTPLoginService(store, delivery, parentOTPLoginTestKey)
	if err != nil {
		t.Fatalf("create Parent OTP login service: %v", err)
	}
	return service
}

func TestNewParentOTPLoginServiceValidatesDependencies(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{}
	delivery := &parentOTPLoginDeliveryFake{}

	tests := []struct {
		name string
		store ParentOTPLoginChallengeStore
		delivery ParentOTPDelivery
		key []byte
		wantErr error
	}{
		{name: "missing store", delivery: delivery, key: parentOTPLoginTestKey},
		{name: "missing delivery", store: store, key: parentOTPLoginTestKey, wantErr: ErrParentOTPDeliveryUnavailable},
		{name: "short HMAC key", store: store, delivery: delivery, key: []byte("short")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewParentOTPLoginService(tt.store, tt.delivery, tt.key); err == nil {
				t.Fatal("expected constructor error")
			} else if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestParentOTPLoginRequestOTPCreatesAndDeliversBoundChallenge(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{}
	delivery := &parentOTPLoginDeliveryFake{}
	service := newParentOTPLoginTestService(t, store, delivery)

	before := time.Now()
	if err := service.RequestOTP(context.Background(), 73, " +919876543210 ", "203.0.113.8"); err != nil {
		t.Fatalf("request Parent OTP: %v", err)
	}

	if store.createCalls != 1 || delivery.calls != 1 {
		t.Fatalf("unexpected calls: create=%d delivery=%d", store.createCalls, delivery.calls)
	}
	if store.principalID != 73 || store.mobile != "+919876543210" || delivery.mobile != store.mobile {
		t.Fatalf("mobile/principal binding mismatch: store=%+v delivery=%+v", store, delivery)
	}
	if len(delivery.code) != 6 {
		t.Fatalf("delivered OTP length = %d, want 6", len(delivery.code))
	}
	for _, digit := range delivery.code {
		if digit < '0' || digit > '9' {
			t.Fatalf("delivered OTP is not numeric: %q", delivery.code)
		}
	}
	expectedMobileHash, err := HashParentMobile(parentOTPLoginTestKey, store.mobile)
	if err != nil {
		t.Fatalf("hash expected mobile: %v", err)
	}
	expectedIPHash, err := HashParentIP(parentOTPLoginTestKey, "203.0.113.8")
	if err != nil {
		t.Fatalf("hash expected IP: %v", err)
	}
	expectedOTPHash, err := HashParentOTP(parentOTPLoginTestKey, expectedMobileHash, delivery.code)
	if err != nil {
		t.Fatalf("hash delivered OTP: %v", err)
	}
	if store.mobileHash != expectedMobileHash || store.ipHash != expectedIPHash || store.otpHash != expectedOTPHash {
		t.Fatal("repository did not receive the expected HMAC-bound values")
	}
	if store.expiresAt.Before(before.Add(4*time.Minute)) || store.expiresAt.After(time.Now().Add(5*time.Minute)) {
		t.Fatalf("challenge expiry is outside the five-minute window: %v", store.expiresAt)
	}
}

func TestParentOTPLoginRequestOTPRejectsInvalidInputBeforePersistence(t *testing.T) {
	tests := []struct {
		name string
		principalID int64
		mobile string
		ip string
	}{
		{name: "zero principal", mobile: "+919876543210", ip: "203.0.113.8"},
		{name: "invalid mobile", principalID: 73, mobile: "9876543210", ip: "203.0.113.8"},
		{name: "invalid IP", principalID: 73, mobile: "+919876543210", ip: "not-an-ip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &parentOTPLoginChallengeStoreFake{}
			delivery := &parentOTPLoginDeliveryFake{}
			service := newParentOTPLoginTestService(t, store, delivery)
			if err := service.RequestOTP(context.Background(), tt.principalID, tt.mobile, tt.ip); err == nil {
				t.Fatal("expected invalid request error")
			}
			if store.createCalls != 0 || delivery.calls != 0 {
				t.Fatalf("invalid request reached dependencies: create=%d delivery=%d", store.createCalls, delivery.calls)
			}
		})
	}
}

func TestParentOTPLoginRequestOTPDoesNotDeliverWhenChallengeCreationFails(t *testing.T) {
	storeErr := errors.New("database unavailable")
	store := &parentOTPLoginChallengeStoreFake{createErr: storeErr}
	delivery := &parentOTPLoginDeliveryFake{}
	service := newParentOTPLoginTestService(t, store, delivery)

	err := service.RequestOTP(context.Background(), 73, "+919876543210", "203.0.113.8")
	if !errors.Is(err, storeErr) {
		t.Fatalf("error = %v, want wrapped repository error", err)
	}
	if delivery.calls != 0 {
		t.Fatalf("delivery called %d times after persistence failure", delivery.calls)
	}
}

func TestParentOTPLoginRequestOTPInvalidatesChallengeAfterDeliveryFailure(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{challengeID: 84}
	delivery := &parentOTPLoginDeliveryFake{err: errors.New("provider details must not escape")}
	service := newParentOTPLoginTestService(t, store, delivery)

	err := service.RequestOTP(context.Background(), 73, "+919876543210", "203.0.113.8")
	if !errors.Is(err, ErrParentOTPDeliveryUnavailable) {
		t.Fatalf("error = %v, want generic delivery error", err)
	}
	if strings.Contains(err.Error(), "provider details") {
		t.Fatalf("provider error leaked: %v", err)
	}
	if store.invalidateCalls != 1 || store.invalidatedPrincipalID != 73 || store.invalidatedChallengeID != 84 {
		t.Fatalf("delivery failure did not invalidate exact challenge: %+v", store)
	}
}

func TestParentOTPLoginRequestOTPReportsCleanupFailureWithoutProviderDetails(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{
		challengeID: 84,
		invalidateErr: errors.New("database cleanup details"),
	}
	delivery := &parentOTPLoginDeliveryFake{err: errors.New("provider details")}
	service := newParentOTPLoginTestService(t, store, delivery)

	err := service.RequestOTP(context.Background(), 73, "+919876543210", "203.0.113.8")
	if !errors.Is(err, ErrParentOTPDeliveryUnavailable) || !errors.Is(err, ErrParentOTPDeliveryCleanupFailed) {
		t.Fatalf("error = %v, want delivery and cleanup sentinels", err)
	}
	if strings.Contains(err.Error(), "provider details") || strings.Contains(err.Error(), "database cleanup details") {
		t.Fatalf("internal error details leaked: %v", err)
	}
}

func TestParentOTPLoginVerifyOTPSuccess(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{verifyResult: true}
	service := newParentOTPLoginTestService(t, store, &parentOTPLoginDeliveryFake{})

	if err := service.VerifyOTP(context.Background(), 73, " +919876543210 ", "123456"); err != nil {
		t.Fatalf("verify valid Parent OTP: %v", err)
	}
	expectedMobileHash, err := HashParentMobile(parentOTPLoginTestKey, "+919876543210")
	if err != nil {
		t.Fatalf("hash expected mobile: %v", err)
	}
	expectedOTPHash, err := HashParentOTP(parentOTPLoginTestKey, expectedMobileHash, "123456")
	if err != nil {
		t.Fatalf("hash expected OTP: %v", err)
	}
	if store.verifyCalls != 1 || store.verifiedPrincipalID != 73 ||
		store.verifiedMobileHash != expectedMobileHash || store.candidateOTPHash != expectedOTPHash {
		t.Fatalf("unexpected verification arguments: %+v", store)
	}
}

func TestParentOTPLoginVerifyOTPRejectsInvalidOrUnmatchedCode(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{}
	service := newParentOTPLoginTestService(t, store, &parentOTPLoginDeliveryFake{})

	if err := service.VerifyOTP(context.Background(), 73, "+919876543210", "000000"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if store.verifyCalls != 1 {
		t.Fatalf("verify calls = %d, want 1", store.verifyCalls)
	}

	if err := service.VerifyOTP(context.Background(), 73, "+919876543210", "12ab"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("malformed OTP error = %v, want invalid OTP", err)
	}
	if store.verifyCalls != 1 {
		t.Fatalf("malformed OTP reached repository; verify calls = %d", store.verifyCalls)
	}
}

func TestParentOTPLoginVerifyOTPWrapsRepositoryError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	store := &parentOTPLoginChallengeStoreFake{verifyErr: storeErr}
	service := newParentOTPLoginTestService(t, store, &parentOTPLoginDeliveryFake{})

	err := service.VerifyOTP(context.Background(), 73, "+919876543210", "123456")
	if !errors.Is(err, storeErr) {
		t.Fatalf("error = %v, want wrapped repository error", err)
	}
	if errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("repository failure was incorrectly reported as invalid OTP: %v", err)
	}
}

func TestParentOTPLoginVerifyOTPRejectsInvalidIdentityBeforeRepository(t *testing.T) {
	store := &parentOTPLoginChallengeStoreFake{}
	service := newParentOTPLoginTestService(t, store, &parentOTPLoginDeliveryFake{})

	if err := service.VerifyOTP(context.Background(), 0, "+919876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("error = %v, want invalid OTP", err)
	}
	if err := service.VerifyOTP(context.Background(), 73, "9876543210", "123456"); !errors.Is(err, ErrParentOTPInvalid) {
		t.Fatalf("invalid mobile error = %v, want invalid OTP", err)
	}
	if store.verifyCalls != 0 {
		t.Fatalf("invalid identity reached repository %d times", store.verifyCalls)
	}
}
