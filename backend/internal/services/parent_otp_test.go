package services

import (
	"strings"
	"testing"
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
