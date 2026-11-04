package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"school-management-system/internal/models"
	"school-management-system/internal/repository"
)

type fakeRefreshSessionRepository struct {
	principal         *models.AuthPrincipal
	createPrincipalID int64
	expectedTokenVersion int64
	createHash        string
	createExpiry      time.Time
	oldHash           string
	newHash           string
	newExpiry         time.Time
	revokeHash        string
	createErr         error
	rotateErr         error
	revokeErr         error
}

func (f *fakeRefreshSessionRepository) CreatePrincipal(_ context.Context, principalID int64, tokenHash string, expiresAt time.Time) error {
	f.createPrincipalID, f.createHash, f.createExpiry = principalID, tokenHash, expiresAt
	return f.createErr
}

func (f *fakeRefreshSessionRepository) CreatePrincipalForVersion(_ context.Context, principalID, expectedTokenVersion int64, tokenHash string, expiresAt time.Time) error {
	f.createPrincipalID, f.createHash, f.createExpiry = principalID, tokenHash, expiresAt
	f.expectedTokenVersion = expectedTokenVersion
	return f.createErr
}

func (f *fakeRefreshSessionRepository) RotatePrincipal(_ context.Context, oldHash, newHash string, expiresAt time.Time) (*models.AuthPrincipal, error) {
	f.oldHash, f.newHash, f.newExpiry = oldHash, newHash, expiresAt
	return f.principal, f.rotateErr
}

func (f *fakeRefreshSessionRepository) RevokePrincipal(_ context.Context, tokenHash string) error {
	f.revokeHash = tokenHash
	return f.revokeErr
}

func TestNewSessionServiceValidatesExpiry(t *testing.T) {
	repo := &fakeRefreshSessionRepository{}
	for _, days := range []int{0, 31} {
		if _, err := NewSessionService(repo, days); err == nil {
			t.Errorf("expected expiry %d to be rejected", days)
		}
	}
	if _, err := NewSessionService(repo, 7); err != nil {
		t.Fatalf("expected valid expiry, got %v", err)
	}
}

func TestIssueStoresOnlyTokenHash(t *testing.T) {
	repo := &fakeRefreshSessionRepository{}
	service, err := NewSessionService(repo, 7)
	if err != nil {
		t.Fatal(err)
	}

	rawToken, err := service.Issue(context.Background(), &models.AuthPrincipal{ID: 12, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 3})
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected a non-empty refresh token")
	}
	if repo.createPrincipalID != 12 {
		t.Fatalf("stored user ID = %d, want 12", repo.createPrincipalID)
	}
	if repo.createHash != HashRefreshToken(rawToken) || repo.createHash == rawToken {
		t.Fatal("repository must receive only the token hash")
	}
	if !repo.createExpiry.After(time.Now().Add(6 * 24 * time.Hour)) {
		t.Fatal("refresh token expiry should be approximately seven days away")
	}
}

func TestIssueBindsExpectedTokenVersion(t *testing.T) {
	repo := &fakeRefreshSessionRepository{}
	service, err := NewSessionService(repo, 7)
	if err != nil {
		t.Fatal(err)
	}
	principal := &models.AuthPrincipal{ID: 42, Type: "staff", Status: "active", TokenVersion: 7, SchoolID: 3}
	rawToken, err := service.Issue(context.Background(), principal)
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}
	if rawToken == "" || repo.createPrincipalID != principal.ID ||
		repo.expectedTokenVersion != principal.TokenVersion {
		t.Fatalf("session issuance was not bound to authenticated principal version: token=%v principal=%d version=%d",
			rawToken != "", repo.createPrincipalID, repo.expectedTokenVersion)
	}
	if repo.createHash != HashRefreshToken(rawToken) || repo.createHash == rawToken {
		t.Fatal("repository must receive only the hashed refresh token")
	}
}

func TestIssueRejectsMissingOrStaleIdentity(t *testing.T) {
	service, err := NewSessionService(&fakeRefreshSessionRepository{}, 7)
	if err != nil {
		t.Fatal(err)
	}
	for _, principal := range []*models.AuthPrincipal{
		nil,
		{ID: 1, Type: "staff", Status: "active", TokenVersion: 0, SchoolID: 2},
		{ID: 1, Type: "staff", Status: "disabled", TokenVersion: 1, SchoolID: 2},
	} {
		if _, err := service.Issue(context.Background(), principal); err == nil {
			t.Fatalf("Issue(%+v) should reject invalid principal", principal)
		}
	}
}

func TestRotateHashesBothTokensAndReturnsReplacement(t *testing.T) {
	repo := &fakeRefreshSessionRepository{
		principal: &models.AuthPrincipal{ID: 12, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 3, Username: "user"},
	}
	service, err := NewSessionService(repo, 7)
	if err != nil {
		t.Fatal(err)
	}

	principal, replacement, err := service.Rotate(context.Background(), "old-token")
	if err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}
	if principal.ID != 12 || replacement == "" || replacement == "old-token" {
		t.Fatal("expected user and a new refresh token")
	}
	if repo.oldHash != HashRefreshToken("old-token") {
		t.Fatal("repository must receive the hash of the old token")
	}
	if repo.newHash != HashRefreshToken(replacement) || repo.newHash == replacement {
		t.Fatal("repository must receive only the replacement token hash")
	}
}

func TestRotateRejectsEmptyToken(t *testing.T) {
	service, err := NewSessionService(&fakeRefreshSessionRepository{}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Rotate(context.Background(), "  "); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected invalid refresh token error, got %v", err)
	}
}

func TestRotatePropagatesReuseDetection(t *testing.T) {
	repo := &fakeRefreshSessionRepository{rotateErr: repository.ErrRefreshTokenReuse}
	service, err := NewSessionService(repo, 7)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.Rotate(context.Background(), "replayed-token")
	if !errors.Is(err, ErrRefreshTokenReuse) {
		t.Fatalf("expected reuse error, got %v", err)
	}
}

func TestRevokeStoresOnlyTokenHash(t *testing.T) {
	repo := &fakeRefreshSessionRepository{}
	service, err := NewSessionService(repo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Revoke(context.Background(), "raw-token"); err != nil {
		t.Fatal(err)
	}
	if repo.revokeHash != HashRefreshToken("raw-token") || repo.revokeHash == "raw-token" {
		t.Fatal("repository must receive only the token hash")
	}
}

func TestHashRefreshTokenIsStableSHA256Hex(t *testing.T) {
	first := HashRefreshToken("example-token")
	second := HashRefreshToken("example-token")
	if first != second || len(first) != 64 {
		t.Fatalf("expected stable 64-character hash, got %q and %q", first, second)
	}
	if HashRefreshToken("different-token") == first {
		t.Fatal("different raw tokens must not have the same hash")
	}
}
