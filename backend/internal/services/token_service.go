package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type TokenClaims struct {
	SchoolID int64 `json:"school_id"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret       []byte
	issuer       string
	accessExpiry time.Duration
}

func NewTokenService(
	secret string,
	issuer string,
	accessExpiryMinutes int,
) (*TokenService, error) {
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

	return &TokenService{
		secret:       []byte(secret),
		issuer:       issuer,
		accessExpiry: time.Duration(accessExpiryMinutes) * time.Minute,
	}, nil
}

func (s *TokenService) GenerateAccessToken(
	userID int64,
	schoolID int64,
) (string, error) {
	if userID <= 0 || schoolID <= 0 {
		return "", fmt.Errorf("user ID and school ID must be positive")
	}

	now := time.Now()

	claims := TokenClaims{
		SchoolID: schoolID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			ID:        fmt.Sprintf("%d-%d", userID, now.UnixNano()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signedToken, nil
}

func (s *TokenService) ValidateAccessToken(
	tokenString string,
) (*TokenClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}

	claims := &TokenClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil || token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	now := time.Now()

	// Required claims must be present.
	if claims.ExpiresAt == nil ||
		claims.IssuedAt == nil ||
		claims.NotBefore == nil ||
		strings.TrimSpace(claims.ID) == "" ||
		strings.TrimSpace(claims.Subject) == "" ||
		strings.TrimSpace(claims.Issuer) == "" {
		return nil, ErrInvalidToken
	}

	// Validate issuer and time-based claims explicitly.
	if claims.Issuer != s.issuer ||
		!claims.VerifyExpiresAt(now, true) ||
		!claims.VerifyIssuedAt(now, true) ||
		!claims.VerifyNotBefore(now, true) {
		return nil, ErrInvalidToken
	}

	// Parse the entire subject strictly; reject values like "10abc".
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return nil, ErrInvalidToken
	}

	if claims.SchoolID <= 0 {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
