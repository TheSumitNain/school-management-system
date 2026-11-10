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
	CreatePrincipalForVersion(ctx context.Context, principalID, expectedTokenVersion int64, tokenHash string, expiresAt time.Time) error
	RotatePrincipal(ctx context.Context, oldTokenHash, newTokenHash string, newExpiresAt time.Time) (*models.AuthPrincipal, error)
	RevokePrincipal(ctx context.Context, tokenHash string) error
}

// SessionService stores only SHA-256 hashes of opaque, principal-bound tokens.
type SessionService struct {
	sessions refreshSessionRepository
}

func NewSessionService(sessions refreshSessionRepository) (*SessionService, error) {
	if sessions == nil {
		return nil, fmt.Errorf("refresh session repository is required")
	}
	return &SessionService{sessions: sessions}, nil
}

// Issue binds session creation to the exact principal version
// returned by password/MFA authentication. A concurrent credential reset or
// account-security change therefore invalidates the in-flight login.
func (s *SessionService) Issue(ctx context.Context, principal *models.AuthPrincipal) (string, error) {
	if principal == nil || principal.ID <= 0 || principal.TokenVersion <= 0 || principal.Status != "active" {
		return "", fmt.Errorf("valid authenticated principal is required")
	}
	raw, err := generateOpaqueRefreshToken()
	if err != nil {
		return "", err
	}
	if err := s.sessions.CreatePrincipalForVersion(
		ctx, principal.ID, principal.TokenVersion, HashRefreshToken(raw), time.Time{},
	); err != nil {
		return "", err
	}
	return raw, nil
}

type refreshSessionFamilyIssuer interface {
	CreatePrincipalForVersionWithFamily(ctx context.Context, principalID, expectedTokenVersion int64, tokenHash string, expiresAt time.Time) (string, error)
}

// IssueWithFamily creates the refresh family before access-token issuance and
// returns its stable UUID for the JWT sid claim.
func (s *SessionService) IssueWithFamily(ctx context.Context, principal *models.AuthPrincipal) (string, string, error) {
	if principal == nil || principal.ID <= 0 || principal.TokenVersion <= 0 || principal.Status != "active" {
		return "", "", fmt.Errorf("valid authenticated principal is required")
	}
	repo, ok := s.sessions.(refreshSessionFamilyIssuer)
	if !ok {
		return "", "", fmt.Errorf("refresh repository does not support session-family binding")
	}
	raw, err := generateOpaqueRefreshToken()
	if err != nil { return "", "", err }
	familyID, err := repo.CreatePrincipalForVersionWithFamily(ctx, principal.ID, principal.TokenVersion, HashRefreshToken(raw), time.Time{})
	if err != nil { return "", "", err }
	if !validRefreshFamilyID(familyID) { return "", "", fmt.Errorf("refresh repository returned invalid family ID") }
	return raw, familyID, nil
}

func validRefreshFamilyID(value string) bool {
	if len(value) != 36 { return false }
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' { return false }
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) { return false }
		}
	}
	return true
}

func (s *SessionService) Rotate(ctx context.Context, rawToken string) (*models.AuthPrincipal, string, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, "", ErrInvalidRefreshToken
	}
	replacement, err := generateOpaqueRefreshToken()
	if err != nil {
		return nil, "", err
	}
	principal, err := s.sessions.RotatePrincipal(ctx, HashRefreshToken(rawToken), HashRefreshToken(replacement), time.Time{})
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

type refreshSessionFamilyLookup interface {
	FamilyIDForTokenHash(ctx context.Context, tokenHash string) (string, error)
}

// RotateWithFamily preserves the repository's transactional rotation/replay
// protections, then resolves the immutable family UUID for the replacement.
func (s *SessionService) RotateWithFamily(ctx context.Context, rawToken string) (*models.AuthPrincipal, string, string, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, "", "", ErrInvalidRefreshToken
	}
	repo, ok := s.sessions.(refreshSessionFamilyLookup)
	if !ok {
		return nil, "", "", fmt.Errorf("refresh repository does not support session-family lookup")
	}

	// Resolve the immutable family ID before rotation commits. A post-commit
	// lookup failure would otherwise strand an active replacement token that
	// was never returned to the client.
	familyID, err := repo.FamilyIDForTokenHash(ctx, HashRefreshToken(rawToken))
	if err != nil {
		return nil, "", "", err
	}
	if !validRefreshFamilyID(familyID) {
		return nil, "", "", fmt.Errorf("refresh repository returned invalid family ID")
	}

	principal, replacement, err := s.Rotate(ctx, rawToken)
	if err != nil {
		return nil, "", "", err
	}
	return principal, replacement, familyID, nil
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
