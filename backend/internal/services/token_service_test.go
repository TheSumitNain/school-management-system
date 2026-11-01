package services

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const testJWTSecret = "test-secret-with-at-least-32-characters"

func newTestTokenService(t *testing.T) *TokenService {
	t.Helper()

	service, err := NewTokenService(
		testJWTSecret,
		"school-management-system",
		15,
	)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}

	return service
}

func TestNewTokenServiceValidation(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		issuer  string
		expiry  int
		wantErr bool
	}{
		{"valid configuration", testJWTSecret, "school-management", 15, false},
		{"short secret", "too-short", "school-management", 15, true},
		{"empty issuer", testJWTSecret, " ", 15, true},
		{"zero expiry", testJWTSecret, "school-management", 0, true},
		{"expiry too long", testJWTSecret, "school-management", 61, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTokenService(tt.secret, tt.issuer, tt.expiry)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewTokenService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPrincipalAccessTokenRoundTrip(t *testing.T) {
	service := newTestTokenService(t)

	for _, principalType := range []string{"staff", "platform_admin", "parent"} {
		t.Run(principalType, func(t *testing.T) {
			raw, err := service.GenerateAccessToken(42, 3, principalType)
			if err != nil {
				t.Fatalf("GenerateAccessToken() error = %v", err)
			}

			claims, err := service.ValidateAccessToken(raw)
			if err != nil {
				t.Fatalf("ValidateAccessToken() error = %v", err)
			}

			if claims.Subject != "42" {
				t.Errorf("Subject = %q, want %q", claims.Subject, "42")
			}
			if claims.TokenVersion != 3 {
				t.Errorf("TokenVersion = %d, want 3", claims.TokenVersion)
			}
			if claims.PrincipalType != principalType {
				t.Errorf("PrincipalType = %q, want %q", claims.PrincipalType, principalType)
			}
			if claims.Issuer != "school-management-system" {
				t.Errorf("Issuer = %q, unexpected issuer", claims.Issuer)
			}
			if claims.ID == "" {
				t.Error("expected token ID")
			}
		})
	}
}

func TestGenerateAccessTokenRejectsInvalidIdentity(t *testing.T) {
	service := newTestTokenService(t)

	tests := []struct {
		name          string
		principalID   int64
		tokenVersion  int64
		principalType string
	}{
		{"zero principal ID", 0, 1, "staff"},
		{"negative principal ID", -1, 1, "staff"},
		{"zero token version", 1, 0, "staff"},
		{"negative token version", 1, -1, "staff"},
		{"invalid principal type", 1, 1, "student"},
		{"empty principal type", 1, 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GenerateAccessToken(
				tt.principalID,
				tt.tokenVersion,
				tt.principalType,
			)
			if err == nil {
				t.Fatal("expected invalid identity to be rejected")
			}
		})
	}
}

func signTestClaims(t *testing.T, service *TokenService, claims TokenClaims, method jwt.SigningMethod) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	raw, err := token.SignedString(service.secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return raw
}

func validTestClaims(service *TokenService) TokenClaims {
	now := time.Now()

	return TokenClaims{
		PrincipalType: "staff",
		TokenVersion:  1,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "12",
			Issuer:    service.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			ID:        "test-token-id",
		},
	}
}

func TestValidateAccessTokenRejectsTamperedToken(t *testing.T) {
	service := newTestTokenService(t)

	raw, err := service.GenerateAccessToken(12, 1, "staff")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	signature := []byte(parts[2])
	if signature[0] == 'A' {
		signature[0] = 'B'
	} else {
		signature[0] = 'A'
	}
	parts[2] = string(signature)

	if _, err := service.ValidateAccessToken(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
}

func TestValidateAccessTokenRejectsWrongIssuer(t *testing.T) {
	service := newTestTokenService(t)

	claims := validTestClaims(service)
	claims.Issuer = "wrong-issuer"

	raw := signTestClaims(t, service, claims, jwt.SigningMethodHS256)

	if _, err := service.ValidateAccessToken(raw); err == nil {
		t.Fatal("expected wrong issuer to be rejected")
	}
}

func TestValidateAccessTokenRejectsExpiredToken(t *testing.T) {
	service := newTestTokenService(t)

	now := time.Now()
	claims := validTestClaims(service)
	claims.IssuedAt = jwt.NewNumericDate(now.Add(-2 * time.Hour))
	claims.NotBefore = jwt.NewNumericDate(now.Add(-2 * time.Hour))
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Hour))

	raw := signTestClaims(t, service, claims, jwt.SigningMethodHS256)

	if _, err := service.ValidateAccessToken(raw); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestValidateAccessTokenRejectsMalformedSubject(t *testing.T) {
	service := newTestTokenService(t)

	subjects := []string{
		"",
		"12abc",
		"0",
		"-12",
	}

	for _, subject := range subjects {
		t.Run(subject, func(t *testing.T) {
			claims := validTestClaims(service)
			claims.Subject = subject

			raw := signTestClaims(t, service, claims, jwt.SigningMethodHS256)

			if _, err := service.ValidateAccessToken(raw); err == nil {
				t.Fatalf("expected malformed subject %q to be rejected", subject)
			}
		})
	}
}

func TestValidateAccessTokenRejectsMissingRequiredClaims(t *testing.T) {
	service := newTestTokenService(t)

	tests := []struct {
		name   string
		mutate func(*TokenClaims)
	}{
		{"missing expiration", func(c *TokenClaims) { c.ExpiresAt = nil }},
		{"missing issued at", func(c *TokenClaims) { c.IssuedAt = nil }},
		{"missing not before", func(c *TokenClaims) { c.NotBefore = nil }},
		{"missing token ID", func(c *TokenClaims) { c.ID = "" }},
		{"missing subject", func(c *TokenClaims) { c.Subject = "" }},
		{"invalid token version", func(c *TokenClaims) { c.TokenVersion = 0 }},
		{"invalid principal type", func(c *TokenClaims) { c.PrincipalType = "student" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validTestClaims(service)
			tt.mutate(&claims)

			raw := signTestClaims(t, service, claims, jwt.SigningMethodHS256)

			if _, err := service.ValidateAccessToken(raw); err == nil {
				t.Fatal("expected token with invalid required claims to be rejected")
			}
		})
	}
}

func TestValidateAccessTokenRejectsWrongSigningAlgorithm(t *testing.T) {
	service := newTestTokenService(t)

	claims := validTestClaims(service)
	raw := signTestClaims(t, service, claims, jwt.SigningMethodHS384)

	if _, err := service.ValidateAccessToken(raw); err == nil {
		t.Fatal("expected non-HS256 token to be rejected")
	}
}

func TestValidateAccessTokenRejectsEmptyToken(t *testing.T) {
	service := newTestTokenService(t)

	if _, err := service.ValidateAccessToken(""); err == nil {
		t.Fatal("expected empty token to be rejected")
	}
}
