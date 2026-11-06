package services

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ParentOTPLoginChallengeStore persists OTP challenges bound to an existing
// Parent principal and shares the registration flow's durable send limits.
type ParentOTPLoginChallengeStore interface {
	CreatePrincipalChallenge(ctx context.Context, principalID int64, mobileE164, mobileHash, ipHash, otpHash string, expiresAt time.Time) (int64, error)
	VerifyPrincipalChallenge(ctx context.Context, principalID int64, mobileHash, candidateOTPHash string) (bool, error)
	InvalidatePrincipalChallenge(ctx context.Context, principalID, challengeID int64) error
}

// ParentOTPLoginService handles OTP delivery and verification for an existing
// Parent account. The caller resolves the principal from the submitted mobile;
// the repository transaction rechecks the exact principal/mobile binding.
type ParentOTPLoginService struct {
	challenges ParentOTPLoginChallengeStore
	delivery   ParentOTPDelivery
	hmacKey    []byte
}

func NewParentOTPLoginService(challenges ParentOTPLoginChallengeStore, delivery ParentOTPDelivery, hmacKey []byte) (*ParentOTPLoginService, error) {
	if challenges == nil {
		return nil, fmt.Errorf("Parent OTP login challenge store is required")
	}
	if delivery == nil {
		return nil, ErrParentOTPDeliveryUnavailable
	}
	if len(hmacKey) < 32 {
		return nil, fmt.Errorf("Parent OTP HMAC key must contain at least 32 bytes")
	}
	return &ParentOTPLoginService{challenges: challenges, delivery: delivery, hmacKey: append([]byte(nil), hmacKey...)}, nil
}

func (s *ParentOTPLoginService) RequestOTP(ctx context.Context, principalID int64, mobileE164, clientIP string) error {
	if s == nil || s.challenges == nil || s.delivery == nil || len(s.hmacKey) < 32 || principalID <= 0 {
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
		return fmt.Errorf("generate Parent login OTP: %w", err)
	}
	otpHash, err := HashParentOTP(s.hmacKey, mobileHash, code)
	if err != nil {
		return fmt.Errorf("hash Parent login OTP: %w", err)
	}
	challengeID, err := s.challenges.CreatePrincipalChallenge(ctx, principalID, mobile, mobileHash, ipHash, otpHash, time.Now().Add(5*time.Minute))
	if err != nil {
		return err
	}
	if err := s.delivery.SendParentOTP(ctx, mobile, code); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if cleanupErr := s.challenges.InvalidatePrincipalChallenge(cleanupCtx, principalID, challengeID); cleanupErr != nil {
			return errors.Join(ErrParentOTPDeliveryUnavailable, ErrParentOTPDeliveryCleanupFailed)
		}
		return ErrParentOTPDeliveryUnavailable
	}
	return nil
}

func (s *ParentOTPLoginService) VerifyOTP(ctx context.Context, principalID int64, mobileE164, code string) error {
	if s == nil || s.challenges == nil || len(s.hmacKey) < 32 || principalID <= 0 {
		return ErrParentOTPInvalid
	}
	mobile, err := NormalizeParentMobile(mobileE164)
	if err != nil {
		return ErrParentOTPInvalid
	}
	mobileHash, err := HashParentMobile(s.hmacKey, mobile)
	if err != nil {
		return ErrParentOTPInvalid
	}
	candidateHash, err := HashParentOTP(s.hmacKey, mobileHash, code)
	if err != nil {
		return ErrParentOTPInvalid
	}
	verified, err := s.challenges.VerifyPrincipalChallenge(ctx, principalID, mobileHash, candidateHash)
	if err != nil {
		return fmt.Errorf("verify Parent login OTP challenge: %w", err)
	}
	if !verified {
		return ErrParentOTPInvalid
	}
	return nil
}
