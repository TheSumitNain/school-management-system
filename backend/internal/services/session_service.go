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
	ErrSessionPrincipalVersionChanged = repository.ErrPrincipalVersionChanged
)

type refreshSessionRepository interface {
	CreatePrincipal(ctx context.Context, principalID int64, tokenHash string, expiresAt time.Time) error
	CreatePrincipalForVersion(ctx context.Context, principalID, expectedTokenVersion int64, tokenHash string, expiresAt time.Time) error
	RotatePrincipal(ctx context.Context, oldTokenHash, newTokenHash string, newExpiresAt time.Time) (*models.AuthPrincipal, error)
	RevokePrincipal(ctx context.Context, tokenHash string) error
}

// SessionService stores only SHA-256 hashes of opaque, principal-bound tokens.
type SessionService struct {
	sessions refreshSessionRepository
	expiry   time.Duration
}

func NewSessionService(sessions refreshSessionRepository, refreshExpiryDays int) (*SessionService, error) {
	if sessions == nil {
		return nil, fmt.Errorf("refresh session repository is required")
	}
	if refreshExpiryDays < 1 || refreshExpiryDays > 30 {
		return nil, fmt.Errorf("refresh token expiry must be between 1 and 30 days")
	}
	return &SessionService{sessions: sessions, expiry: time.Duration(refreshExpiryDays) * 24 * time.Hour}, nil
}

func (s *SessionService) Issue(ctx context.Context, principalID int64) (string, error) {
	if principalID <= 0 {
		return "", fmt.Errorf("principal ID must be positive")
	}
	raw, err := generateOpaqueRefreshToken()
	if err != nil {
		return "", err
	}
	if err := s.sessions.CreatePrincipal(ctx, principalID, HashRefreshToken(raw), time.Now().Add(s.expiry)); err != nil {
		return "", err
	}
	return raw, nil
}

// IssueForPrincipal binds session creation to the exact principal version
// returned by password/MFA authentication. A concurrent credential reset or
// account-security change therefore invalidates the in-flight login.
func (s *SessionService) IssueForPrincipal(ctx context.Context, principal *models.AuthPrincipal) (string, error) {
	if principal == nil || principal.ID <= 0 || principal.TokenVersion <= 0 || principal.Status != "active" {
		return "", fmt.Errorf("valid authenticated principal is required")
	}
	raw, err := generateOpaqueRefreshToken()
	if err != nil {
		return "", err
	}
	if err := s.sessions.CreatePrincipalForVersion(
		ctx, principal.ID, principal.TokenVersion, HashRefreshToken(raw), time.Now().Add(s.expiry),
	); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *SessionService) Rotate(ctx context.Context, rawToken string) (*models.AuthPrincipal, string, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, "", ErrInvalidRefreshToken
	}
	replacement, err := generateOpaqueRefreshToken()
	if err != nil {
		return nil, "", err
	}
	principal, err := s.sessions.RotatePrincipal(ctx, HashRefreshToken(rawToken), HashRefreshToken(replacement), time.Now().Add(s.expiry))
	if err != nil {
		return nil, "", err
	}
	if principal == nil || principal.ID <= 0 || principal.TokenVersion <= 0 || principal.Status != "active" ||
		(principal.Type != "staff" && principal.Type != "platform_admin" && principal.Type != "parent") ||
		(principal.Type == "staff" && principal.SchoolID <= 0) || (principal.Type == "parent" && !principal.MobileVerified) {
		return nil, "", fmt.Errorf("refresh session repository returned invalid principal identity")
	}
	return principal, replacement, nil
}

func (s *SessionService) Revoke(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return ErrInvalidRefreshToken
	}
	return s.sessions.RevokePrincipal(ctx, HashRefreshToken(rawToken))
}

func generateOpaqueRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
