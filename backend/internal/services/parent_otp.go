package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var (
	ErrParentOTPInvalid       = errors.New("invalid or expired parent OTP")
	ErrParentOTPDeliveryUnavailable = errors.New("parent OTP delivery is unavailable")
)

var (
	parentE164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	parentHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	parentOTPPattern  = regexp.MustCompile(`^[0-9]{6}$`)
)

// ParentOTPDelivery sends a one-time code through an explicitly configured
// delivery provider. Implementations must never log or persist the raw code.
// Automated tests may provide an in-memory fake; production must use a real
// provider and must not expose the code through an API response.
type ParentOTPDelivery interface {
	SendParentOTP(ctx context.Context, mobileE164, code string) error
}

// NormalizeParentMobile accepts only canonical E.164 numbers. It deliberately
// does not guess country codes or rewrite local/national-format numbers.
func NormalizeParentMobile(value string) (string, error) {
	mobile := strings.TrimSpace(value)
	if !parentE164Pattern.MatchString(mobile) {
		return "", fmt.Errorf("mobile number must be in canonical E.164 format")
	}
	return mobile, nil
}

// GenerateParentOTP returns a uniformly distributed six-digit decimal code.
// The caller must pass it only to ParentOTPDelivery and must never log or
// return it to the client.
func GenerateParentOTP() (string, error) {
	const upperBound = 1000000
	n, err := rand.Int(rand.Reader, big.NewInt(upperBound))
	if err != nil {
		return "", fmt.Errorf("generate parent OTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashParentMobile creates a domain-separated HMAC lookup key. Raw mobile
// numbers must never be stored in OTP challenge or rate-limit lookup columns.
func HashParentMobile(key []byte, mobileE164 string) (string, error) {
	if len(key) < 32 {
		return "", fmt.Errorf("parent OTP HMAC key must contain at least 32 bytes")
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return "", err
	}
	return parentHMAC(key, "school-management:parent-otp:mobile:v1:"+mobile), nil
}

// HashParentOTP binds the six-digit code to its mobile lookup hash and uses a
// server-side HMAC key so a database-only compromise cannot cheaply enumerate
// the small OTP keyspace. The key must be independently managed from JWT/TOTP.
func HashParentOTP(key []byte, mobileLookupHash, code string) (string, error) {
	if len(key) < 32 {
		return "", fmt.Errorf("parent OTP HMAC key must contain at least 32 bytes")
	}
	if !parentHashPattern.MatchString(mobileLookupHash) || !parentOTPPattern.MatchString(code) {
		return "", ErrParentOTPInvalid
	}
	return parentHMAC(key, "school-management:parent-otp:code:v1:"+mobileLookupHash+":"+code), nil
}

// VerifyParentOTPHash performs a constant-time comparison of the persisted
// challenge hash with a candidate code's HMAC.
func VerifyParentOTPHash(key []byte, mobileLookupHash, code, expectedHash string) bool {
	if !parentHashPattern.MatchString(expectedHash) {
		return false
	}
	actual, err := HashParentOTP(key, mobileLookupHash, code)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(actual), []byte(expectedHash))
}

func parentHMAC(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
