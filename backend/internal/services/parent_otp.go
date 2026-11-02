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
	"net"
	"time"
	"regexp"
	"strings"
)

var (
	ErrParentOTPInvalid             = errors.New("invalid or expired parent OTP")
	ErrParentOTPDeliveryUnavailable = errors.New("parent OTP delivery is unavailable")
	ErrParentOTPDeliveryCleanupFailed = errors.New("parent OTP challenge cleanup failed after delivery failure")
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


// ParentOTPChallengeCreator is the persistence boundary used by the request
// flow. The repository stores only HMACs and enforces persistent send limits.
type ParentOTPChallengeCreator interface {
	CreatePreAccountChallenge(ctx context.Context, mobileHash, ipHash, otpHash string, expiresAt time.Time) (int64, error)
	InvalidatePreAccountChallenge(ctx context.Context, challengeID int64) error
}

// ParentOTPService coordinates OTP generation, durable rate-limit reservation,
// and provider delivery. The raw code exists only in this method's memory and
// is passed only to the configured delivery provider.
type ParentOTPService struct {
	challenges ParentOTPChallengeCreator
	delivery   ParentOTPDelivery
	hmacKey    []byte
}

func NewParentOTPService(challenges ParentOTPChallengeCreator, delivery ParentOTPDelivery, hmacKey []byte) (*ParentOTPService, error) {
	if challenges == nil {
		return nil, fmt.Errorf("parent OTP challenge repository is required")
	}
	if delivery == nil {
		return nil, ErrParentOTPDeliveryUnavailable
	}
	if len(hmacKey) < 32 {
		return nil, fmt.Errorf("parent OTP HMAC key must contain at least 32 bytes")
	}
	return &ParentOTPService{
		challenges: challenges,
		delivery: delivery,
		hmacKey: append([]byte(nil), hmacKey...),
	}, nil
}

// RequestOTP starts a pre-account challenge. clientIP must be the address
// resolved by the HTTP server's trusted-proxy policy, never a raw forwarded
// header supplied directly by the caller.
func (s *ParentOTPService) RequestOTP(ctx context.Context, mobileE164, clientIP string) error {
	if s == nil || s.challenges == nil || s.delivery == nil || len(s.hmacKey) < 32 {
		return ErrParentOTPDeliveryUnavailable
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return ErrParentOTPInvalid
	}
	mobileHash, err := HashParentMobile(s.hmacKey, mobile)
	if err != nil {
		return ErrParentOTPInvalid
	}
	ipHash, err := HashParentIP(s.hmacKey, clientIP)
	if err != nil {
		return ErrParentOTPInvalid
	}
	code, err := GenerateParentOTP()
	if err != nil {
		return fmt.Errorf("generate parent OTP: %w", err)
	}
	otpHash, err := HashParentOTP(s.hmacKey, mobileHash, code)
	if err != nil {
		return fmt.Errorf("hash parent OTP: %w", err)
	}
	challengeID, err := s.challenges.CreatePreAccountChallenge(
		ctx, mobileHash, ipHash, otpHash, time.Now().Add(5*time.Minute),
	)
	if err != nil {
		return err
	}
	if err := s.delivery.SendParentOTP(ctx, mobile, code); err != nil {
		// Delivery may have partially succeeded. Consume the challenge using
		// a bounded context detached from caller cancellation, while retaining
		// the durable rate-limit reservation.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if cleanupErr := s.challenges.InvalidatePreAccountChallenge(cleanupCtx, challengeID); cleanupErr != nil {
			// Do not expose provider or database details to the caller. The
			// additional sentinel lets internal callers distinguish a failed
			// cleanup and trigger operational alerting without leaking secrets.
			return errors.Join(ErrParentOTPDeliveryUnavailable, ErrParentOTPDeliveryCleanupFailed)
		}
		return ErrParentOTPDeliveryUnavailable
	}
	return nil
}

// HashParentIP produces a domain-separated lookup HMAC for persistent IP
// throttling. It canonicalizes IPv4/IPv6 text to prevent equivalent textual
// representations from bypassing the same rate-limit bucket.
func HashParentIP(key []byte, address string) (string, error) {
	if len(key) < 32 {
		return "", fmt.Errorf("parent OTP HMAC key must contain at least 32 bytes")
	}
	ip := net.ParseIP(strings.TrimSpace(address))
	if ip == nil {
		return "", fmt.Errorf("invalid client IP address")
	}
	canonical := ip.String()
	if ipv4 := ip.To4(); ipv4 != nil {
		canonical = ipv4.String()
	}
	return parentHMAC(key, "school-management:parent-otp:ip:v1:"+canonical), nil
}
