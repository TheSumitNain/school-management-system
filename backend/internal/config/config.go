package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBName     string
	DBPassword string
	DBSSLMode  string

	JWTSecret              string
	JWTIssuer              string
	JWTAccessExpiryMinutes int
	RefreshTokenExpiryDays int

	// TOTPEncryptionKey is loaded from Base64-encoded secret material.
	// It must never be logged or exposed in API responses.
	TOTPEncryptionKey        []byte
	TOTPEncryptionKeyVersion int

	// ParentOTPHMACKey is independent secret material used only to derive
	// keyed lookup and OTP hashes for the Parent portal.
	ParentOTPHMACKey []byte
}

func Load() Config {
	_ = godotenv.Load()

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "development"
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8081"
	}

	dbSSLMode := strings.ToLower(strings.TrimSpace(os.Getenv("DB_SSLMODE")))
	if dbSSLMode == "" {
		dbSSLMode = "disable"
		if appEnv == "production" {
			dbSSLMode = "verify-full"
		}
	}

	jwtIssuer := strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	if jwtIssuer == "" {
		jwtIssuer = "school-management-system"
	}

	jwtAccessExpiry := 15
	if value := strings.TrimSpace(os.Getenv("JWT_ACCESS_EXPIRY_MINUTES")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			jwtAccessExpiry = parsed
		}
	}

	refreshExpiryDays := 7
	if value := strings.TrimSpace(os.Getenv("REFRESH_TOKEN_EXPIRY_DAYS")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			refreshExpiryDays = parsed
		}
	}

	totpEncryptionKey := []byte(nil)
	if encoded := strings.TrimSpace(os.Getenv("TOTP_ENCRYPTION_KEY")); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil {
			totpEncryptionKey = decoded
		}
	}

	totpEncryptionKeyVersion := 1
	if value := strings.TrimSpace(os.Getenv("TOTP_ENCRYPTION_KEY_VERSION")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			totpEncryptionKeyVersion = 0
		} else {
			totpEncryptionKeyVersion = parsed
		}
	}

	parentOTPHMACKey := []byte(nil)
	if encoded := strings.TrimSpace(os.Getenv("PARENT_OTP_HMAC_KEY")); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil {
			parentOTPHMACKey = decoded
		}
	}

	return Config{
		AppEnv:                 appEnv,
		AppPort:                appPort,
		DBHost:                 os.Getenv("DB_HOST"),
		DBPort:                 os.Getenv("DB_PORT"),
		DBUser:                 os.Getenv("DB_USER"),
		DBName:                 os.Getenv("DB_NAME"),
		DBPassword:             os.Getenv("DB_PASSWORD"),
		DBSSLMode:              dbSSLMode,
		JWTSecret:              os.Getenv("JWT_SECRET"),
		JWTIssuer:              jwtIssuer,
		JWTAccessExpiryMinutes: jwtAccessExpiry,
		RefreshTokenExpiryDays: refreshExpiryDays,
		TOTPEncryptionKey:        totpEncryptionKey,
		TOTPEncryptionKeyVersion: totpEncryptionKeyVersion,
		ParentOTPHMACKey:         parentOTPHMACKey,
	}
}

func (cfg Config) ValidateDatabase() error {
	if cfg.DBHost == "" ||
		cfg.DBPort == "" ||
		cfg.DBUser == "" ||
		cfg.DBName == "" ||
		cfg.DBPassword == "" {
		return fmt.Errorf("database configuration is incomplete")
	}

	port, err := strconv.Atoi(cfg.DBPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid database port")
	}

	switch cfg.DBSSLMode {
	case "disable", "verify-full":
	default:
		return fmt.Errorf(
			"unsupported DB_SSLMODE %q: use disable or verify-full",
			cfg.DBSSLMode,
		)
	}

	if cfg.AppEnv == "production" && cfg.DBSSLMode != "verify-full" {
		return fmt.Errorf(
			"production requires DB_SSLMODE=verify-full",
		)
	}

	return nil
}

func (cfg Config) ValidateJWT() error {
	secret := strings.TrimSpace(cfg.JWTSecret)

	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if strings.TrimSpace(cfg.JWTIssuer) == "" {
		return fmt.Errorf("JWT_ISSUER is required")
	}

	if cfg.JWTAccessExpiryMinutes < 1 ||
		cfg.JWTAccessExpiryMinutes > 60 {
		return fmt.Errorf(
			"JWT_ACCESS_EXPIRY_MINUTES must be between 1 and 60",
		)
	}

	if cfg.RefreshTokenExpiryDays < 1 || cfg.RefreshTokenExpiryDays > 30 {
		return fmt.Errorf("REFRESH_TOKEN_EXPIRY_DAYS must be between 1 and 30")
	}

	return nil
}


// ValidateTOTPEncryption validates the key material required by the MFA
// service. It is intentionally separate from ValidateJWT so deployments can
// roll out the MFA capability without silently inventing a local key.
func (cfg Config) ValidateTOTPEncryption() error {
	if len(cfg.TOTPEncryptionKey) != 32 {
		return fmt.Errorf("TOTP_ENCRYPTION_KEY must be valid Base64 encoding of exactly 32 bytes")
	}
	if cfg.TOTPEncryptionKeyVersion < 1 || cfg.TOTPEncryptionKeyVersion > 2147483647 {
		return fmt.Errorf("TOTP_ENCRYPTION_KEY_VERSION must be between 1 and 2147483647")
	}
	return nil
}

// ValidateParentOTPHMAC validates the independent secret required by Parent
// OTP. It is deliberately separate from TOTP encryption configuration.
func (cfg Config) ValidateParentOTPHMAC() error {
	if len(cfg.ParentOTPHMACKey) < 32 {
		return fmt.Errorf("PARENT_OTP_HMAC_KEY must be valid Base64 encoding of at least 32 bytes")
	}
	return nil
}
