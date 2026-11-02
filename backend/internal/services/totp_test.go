package services

import (
	"bytes"
	"encoding/base32"
	"net/url"
	"testing"
	"time"
)

func TestGenerateTOTPSecret(t *testing.T) {
	first, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error = %v", err)
	}
	second, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() second error = %v", err)
	}
	if len(first) != 32 {
		t.Fatalf("secret length = %d, want 32 Base32 characters", len(first))
	}
	if first == second {
		t.Fatal("two generated TOTP secrets unexpectedly match")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(first)
	if err != nil {
		t.Fatalf("generated secret is not valid Base32: %v", err)
	}
	if len(decoded) != totpSecretBytes {
		t.Fatalf("decoded secret length = %d, want %d", len(decoded), totpSecretBytes)
	}
}

func TestVerifyTOTPUsesRFC4226HOTPVectorAtUnixTime59(t *testing.T) {
	secret := []byte("12345678901234567890")
	if !VerifyTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("VerifyTOTP() rejected RFC HOTP counter-1 six-digit vector")
	}
	if VerifyTOTP(secret, "287083", time.Unix(59, 0)) {
		t.Fatal("VerifyTOTP() accepted an incorrect code")
	}
}

func TestVerifyTOTPAcceptsAdjacentTimeStepsOnly(t *testing.T) {
	secret := []byte("12345678901234567890")
	// Counter 2 is accepted while the current counter is 1 (one step ahead).
	if !VerifyTOTP(secret, "359152", time.Unix(59, 0)) {
		t.Fatal("VerifyTOTP() rejected the adjacent future time step")
	}
	// Counter 3 is two steps ahead and must not be accepted.
	if VerifyTOTP(secret, "969429", time.Unix(59, 0)) {
		t.Fatal("VerifyTOTP() accepted a code outside the configured time window")
	}
}

func TestVerifyTOTPAndGetStepReturnsMatchedCounter(t *testing.T) {
	secret := []byte("12345678901234567890")
	step, ok := VerifyTOTPAndGetStep(secret, "287082", time.Unix(59, 0))
	if !ok || step != 1 {
		t.Fatalf("VerifyTOTPAndGetStep() = (%d, %v), want (1, true)", step, ok)
	}
	step, ok = VerifyTOTPAndGetStep(secret, "359152", time.Unix(59, 0))
	if !ok || step != 2 {
		t.Fatalf("adjacent-step match = (%d, %v), want (2, true)", step, ok)
	}
	if step, ok = VerifyTOTPAndGetStep(secret, "000000", time.Unix(59, 0)); ok {
		t.Fatalf("unexpectedly matched invalid code at step %d", step)
	}
}

func TestVerifyTOTPRejectsMalformedInputs(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(59, 0)
	for _, code := range []string{"", "12345", "1234567", "12a456", " 23456", "１２３４５６"} {
		if VerifyTOTP(secret, code, now) {
			t.Errorf("VerifyTOTP() accepted malformed code %q", code)
		}
	}
	if VerifyTOTP([]byte("short"), "287082", now) {
		t.Fatal("VerifyTOTP() accepted a short secret")
	}
	if VerifyTOTP(secret, "287082", time.Unix(-1, 0)) {
		t.Fatal("VerifyTOTP() accepted a negative timestamp")
	}
}

func TestEncryptDecryptTOTPSecret(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	secret := []byte("12345678901234567890")
	encrypted, err := EncryptTOTPSecret(secret, key)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret() error = %v", err)
	}
	if bytes.Contains(encrypted, secret) {
		t.Fatal("encrypted output contains plaintext secret")
	}
	decrypted, err := DecryptTOTPSecret(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptTOTPSecret() error = %v", err)
	}
	if !bytes.Equal(decrypted, secret) {
		t.Fatal("decrypted secret does not match original")
	}
	if _, err := DecryptTOTPSecret(encrypted, bytes.Repeat([]byte{0x24}, 32)); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted the wrong key")
	}
	tampered := append([]byte(nil), encrypted...)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := DecryptTOTPSecret(tampered, key); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted tampered ciphertext")
	}
}

func TestTOTPEncryptionRejectsInvalidKeyAndSecret(t *testing.T) {
	if _, err := EncryptTOTPSecret([]byte("1234567890123456"), []byte("short")); err == nil {
		t.Fatal("EncryptTOTPSecret() accepted an invalid key")
	}
	if _, err := EncryptTOTPSecret([]byte("short"), bytes.Repeat([]byte{1}, 32)); err == nil {
		t.Fatal("EncryptTOTPSecret() accepted a short secret")
	}
	if _, err := DecryptTOTPSecret([]byte("short"), bytes.Repeat([]byte{1}, 32)); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted truncated ciphertext")
	}
}

func TestTOTPProvisioningURI(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	got, err := TOTPProvisioningURI("School Management", "admin@example.com", secret)
	if err != nil {
		t.Fatalf("TOTPProvisioningURI() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
		t.Fatalf("unexpected URI scheme/host: %q %q", parsed.Scheme, parsed.Host)
	}
	query := parsed.Query()
	if query.Get("secret") != secret || query.Get("issuer") != "School Management" ||
		query.Get("algorithm") != "SHA1" || query.Get("digits") != "6" || query.Get("period") != "30" {
		t.Fatalf("unexpected provisioning query: %v", query)
	}
	if _, err := TOTPProvisioningURI("", "admin@example.com", secret); err == nil {
		t.Fatal("TOTPProvisioningURI() accepted an empty issuer")
	}
	if _, err := TOTPProvisioningURI("Issuer", "admin@example.com", "not-base32!"); err == nil {
		t.Fatal("TOTPProvisioningURI() accepted an invalid Base32 secret")
	}
}
