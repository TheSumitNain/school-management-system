package services

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func newTestTokenService(t *testing.T) *TokenService {
	t.Helper()

	service, err := NewTokenService(
		"test-secret-with-at-least-32-characters",
		"school-management-system",
		15,
	)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	return service
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	service := newTestTokenService(t)

	tokenString, err := service.GenerateAccessToken(12, 5)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := service.ValidateAccessToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.Subject != "12" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "12")
	}

	if claims.SchoolID != 5 {
		t.Errorf("SchoolID = %d, want 5", claims.SchoolID)
	}

	if claims.Issuer != "school-management-system" {
		t.Errorf("Issuer = %q, unexpected issuer", claims.Issuer)
	}
}

func TestGenerateAccessTokenRejectsInvalidIDs(t *testing.T) {
	service := newTestTokenService(t)

	tests := []struct {
		name     string
		userID   int64
		schoolID int64
	}{
		{name: "invalid user ID", userID: 0, schoolID: 5},
		{name: "invalid school ID", userID: 12, schoolID: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GenerateAccessToken(tt.userID, tt.schoolID)
			if err == nil {
				t.Fatal("expected error for invalid IDs")
			}
		})
	}
}

func TestValidateAccessTokenRejectsTamperedToken(t *testing.T) {
	service := newTestTokenService(t)

	tokenString, err := service.GenerateAccessToken(12, 5)
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Fatalf("expected JWT to have 3 parts, got %d", len(parts))
	}

	// Signature ke pehle character ko badalte hain.
	signature := []byte(parts[2])
	if signature[0] == 'A' {
		signature[0] = 'B'
	} else {
		signature[0] = 'A'
	}
	parts[2] = string(signature)

	tampered := strings.Join(parts, ".")

	if _, err := service.ValidateAccessToken(tampered); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
}

func TestValidateAccessTokenRejectsWrongIssuer(t *testing.T) {
	service := newTestTokenService(t)

	claims := TokenClaims{
		SchoolID: 5,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "12",
			Issuer:    "wrong-issuer",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}

	tokenString, err := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	).SignedString(service.secret)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected wrong issuer to be rejected")
	}
}

func TestValidateAccessTokenRejectsExpiredToken(t *testing.T) {
	service := newTestTokenService(t)

	claims := TokenClaims{
		SchoolID: 5,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "12",
			Issuer:    service.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}

	tokenString, err := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	).SignedString(service.secret)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ValidateAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestValidateAccessTokenRejectsMalformedSubject(t *testing.T) {
	service := newTestTokenService(t)

	tests := []struct {
		name    string
		subject string
	}{
		{name: "trailing characters", subject: "12abc"},
		{name: "zero user ID", subject: "0"},
		{name: "negative user ID", subject: "-12"},
		{name: "empty subject", subject: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()

			claims := TokenClaims{
				SchoolID: 5,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   tt.subject,
					Issuer:    service.issuer,
					IssuedAt:  jwt.NewNumericDate(now),
					NotBefore: jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
					ID:        "test-token-id",
				},
			}

			tokenString, err := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				claims,
			).SignedString(service.secret)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := service.ValidateAccessToken(tokenString); err == nil {
				t.Fatal("expected malformed subject to be rejected")
			}
		})
	}
}

func TestValidateAccessTokenRejectsMissingRequiredClaims(t *testing.T) {
	service := newTestTokenService(t)
	now := time.Now()

	tests := []struct {
		name   string
		claims TokenClaims
	}{
		{
			name: "missing expiration",
			claims: TokenClaims{
				SchoolID: 5,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "12",
					Issuer:    service.issuer,
					IssuedAt:  jwt.NewNumericDate(now),
					NotBefore: jwt.NewNumericDate(now),
					ID:        "test-token-id",
				},
			},
		},
		{
			name: "missing issued at",
			claims: TokenClaims{
				SchoolID: 5,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "12",
					Issuer:    service.issuer,
					NotBefore: jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
					ID:        "test-token-id",
				},
			},
		},
		{
			name: "missing not before",
			claims: TokenClaims{
				SchoolID: 5,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "12",
					Issuer:    service.issuer,
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
					ID:        "test-token-id",
				},
			},
		},
		{
			name: "missing token ID",
			claims: TokenClaims{
				SchoolID: 5,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "12",
					Issuer:    service.issuer,
					IssuedAt:  jwt.NewNumericDate(now),
					NotBefore: jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, err := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				tt.claims,
			).SignedString(service.secret)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := service.ValidateAccessToken(tokenString); err == nil {
				t.Fatal("expected token with missing claim to be rejected")
			}
		})
	}
}
