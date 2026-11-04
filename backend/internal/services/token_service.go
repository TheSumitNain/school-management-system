package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// TokenClaims deliberately contains identity only. Roles and permissions are
// never embedded in a JWT; protected requests must resolve live RBAC state.
type TokenClaims struct {
	PrincipalType string `json:"principal_type"`
	TokenVersion  int64  `json:"token_version"`
	SessionID     string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret       []byte
	issuer       string
	accessExpiry time.Duration
}

func NewTokenService(secret, issuer string, accessExpiryMinutes int) (*TokenService, error) {
	secret = strings.TrimSpace(secret)
	issuer = strings.TrimSpace(issuer)
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters")
	}
	if issuer == "" {
		return nil, fmt.Errorf("JWT issuer is required")
	}
	if accessExpiryMinutes < 1 || accessExpiryMinutes > 60 {
		return nil, fmt.Errorf("access token expiry must be between 1 and 60 minutes")
	}
	return &TokenService{secret: []byte(secret), issuer: issuer, accessExpiry: time.Duration(accessExpiryMinutes) * time.Minute}, nil
}

// GenerateAccessToken signs a principal-based token. principalID is the
// auth_principals.id, not users.id. tokenVersion supports immediate global
// access-token invalidation when the principal's version is incremented.
func (s *TokenService) GenerateAccessToken(principalID, tokenVersion int64, principalType string) (string, error) {
	return s.GenerateAccessTokenForSession(principalID, tokenVersion, principalType, "")
}

// GenerateAccessTokenForSession binds an access token to the refresh-session
// family that issued it. sid is deliberately optional for compatibility with
// non-session-scoped callers; admin login flows will require it at cutover.
func (s *TokenService) GenerateAccessTokenForSession(principalID, tokenVersion int64, principalType, sessionID string) (string, error) {
	if principalID <= 0 || tokenVersion <= 0 || !validPrincipalType(principalType) {
		return "", fmt.Errorf("invalid principal identity")
	}
	if sessionID != "" && !validSessionID(sessionID) {
		return "", fmt.Errorf("invalid refresh-session family ID")
	}
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("generate access-token ID: %w", err)
	}
	now := time.Now()
	claims := TokenClaims{
		PrincipalType: principalType,
		TokenVersion:  tokenVersion,
		SessionID:     sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(principalID, 10),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			ID:        base64.RawURLEncoding.EncodeToString(jtiBytes),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func validSessionID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' { return false }
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

func (s *TokenService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}
	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.NotBefore == nil ||
		strings.TrimSpace(claims.ID) == "" || strings.TrimSpace(claims.Subject) == "" ||
		claims.Issuer != s.issuer || claims.TokenVersion <= 0 || !validPrincipalType(claims.PrincipalType) ||
		(claims.SessionID != "" && !validSessionID(claims.SessionID)) {
		return nil, ErrInvalidToken
	}
	now := time.Now()
	if !claims.VerifyExpiresAt(now, true) || !claims.VerifyIssuedAt(now, true) || !claims.VerifyNotBefore(now, true) {
		return nil, ErrInvalidToken
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func validPrincipalType(value string) bool {
	switch value {
	case "staff", "platform_admin", "parent":
		return true
	default:
		return false
	}
}
