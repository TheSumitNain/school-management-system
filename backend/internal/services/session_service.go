package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

var (
	ErrInvalidRefreshToken    = repository.ErrRefreshTokenInvalid
	ErrRefreshTokenReuse      = repository.ErrRefreshTokenReuse
	ErrRefreshSessionInactive = repository.ErrRefreshSessionInactive
)

type refreshSessionRepository interface {
	Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	Rotate(ctx context.Context, oldTokenHash, newTokenHash string, newExpiresAt time.Time) (*models.User, error)
	Revoke(ctx context.Context, tokenHash string) error
}

// SessionService manages opaque refresh tokens and their persisted lifecycle.
type SessionService struct {
	sessions refreshSessionRepository
	expiry   time.Duration
}

func NewSessionService(
	sessions refreshSessionRepository,
	refreshExpiryDays int,
) (*SessionService, error) {
	if sessions == nil {
		return nil, fmt.Errorf("refresh session repository is required")
	}
	if refreshExpiryDays < 1 || refreshExpiryDays > 30 {
		return nil, fmt.Errorf("refresh token expiry must be between 1 and 30 days")
	}

	return &SessionService{
		sessions: sessions,
		expiry:   time.Duration(refreshExpiryDays) * 24 * time.Hour,
	}, nil
}

// Issue creates a cryptographically random refresh token and stores only its SHA-256 hash.
func (s *SessionService) Issue(ctx context.Context, userID int64) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("user ID must be positive")
	}

	rawToken, err := generateOpaqueRefreshToken()
	if err != nil {
		return "", err
	}

	if err := s.sessions.Create(
		ctx,
		userID,
		HashRefreshToken(rawToken),
		time.Now().Add(s.expiry),
	); err != nil {
		return "", err
	}

	return rawToken, nil
}

// Rotate invalidates the presented token and returns a one-time replacement.
func (s *SessionService) Rotate(
	ctx context.Context,
	rawToken string,
) (*models.User, string, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, "", ErrInvalidRefreshToken
	}

	replacement, err := generateOpaqueRefreshToken()
	if err != nil {
		return nil, "", err
	}

	user, err := s.sessions.Rotate(
		ctx,
		HashRefreshToken(rawToken),
		HashRefreshToken(replacement),
		time.Now().Add(s.expiry),
	)
	if err != nil {
		return nil, "", err
	}
	if user == nil || user.ID <= 0 || user.SchoolID <= 0 || user.Status != "active" {
		return nil, "", fmt.Errorf("refresh session repository returned invalid user identity")
	}

	return user, replacement, nil
}

// Revoke is safe to call more than once; unknown token hashes are no-ops.
func (s *SessionService) Revoke(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return ErrInvalidRefreshToken
	}
	return s.sessions.Revoke(ctx, HashRefreshToken(rawToken))
}

func generateOpaqueRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashRefreshToken returns the lowercase hexadecimal SHA-256 digest of a raw token.
func HashRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
