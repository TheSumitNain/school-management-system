package config

import (
	"encoding/base64"
	"testing"
)

func validTestConfig() Config {
	return Config{
		AppEnv:     "development",
		AppPort:    "8081",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBName:     "school_management",
		DBPassword: "test-password",
		DBSSLMode:  "disable",
	}
}

func TestValidateDatabase(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid development configuration",
			wantErr: false,
		},
		{
			name: "missing database password",
			modify: func(cfg *Config) {
				cfg.DBPassword = ""
			},
			wantErr: true,
		},
		{
			name: "invalid database port",
			modify: func(cfg *Config) {
				cfg.DBPort = "abc"
			},
			wantErr: true,
		},
		{
			name: "database port out of range",
			modify: func(cfg *Config) {
				cfg.DBPort = "70000"
			},
			wantErr: true,
		},
		{
			name: "unsupported SSL mode",
			modify: func(cfg *Config) {
				cfg.DBSSLMode = "require"
			},
			wantErr: true,
		},
		{
			name: "production with SSL disabled",
			modify: func(cfg *Config) {
				cfg.AppEnv = "production"
				cfg.DBSSLMode = "disable"
			},
			wantErr: true,
		},
		{
			name: "production with verify-full",
			modify: func(cfg *Config) {
				cfg.AppEnv = "production"
				cfg.DBSSLMode = "verify-full"
			},
			wantErr: false,
		},
		{
			name: "development with verify-full",
			modify: func(cfg *Config) {
				cfg.DBSSLMode = "verify-full"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validTestConfig()

			if tt.modify != nil {
				tt.modify(&cfg)
			}

			err := cfg.ValidateDatabase()

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ValidateDatabase() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("REFRESH_TOKEN_EXPIRY_DAYS", "")
	t.Setenv("TOTP_ENCRYPTION_KEY", "")
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "")
	t.Setenv("TOTP_ENCRYPTION_PREVIOUS_KEYS", "")
	t.Setenv("PARENT_OTP_HMAC_KEY", "")

	cfg := Load()

	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want development", cfg.AppEnv)
	}

	if cfg.AppPort != "8081" {
		t.Errorf("AppPort = %q, want 8081", cfg.AppPort)
	}

	if cfg.DBSSLMode != "disable" {
		t.Errorf("DBSSLMode = %q, want disable", cfg.DBSSLMode)
	}
	if cfg.RefreshTokenExpiryDays != 7 {
		t.Errorf("RefreshTokenExpiryDays = %d, want 7", cfg.RefreshTokenExpiryDays)
	}
	if cfg.TOTPEncryptionKeyVersion != 1 {
		t.Errorf("TOTPEncryptionKeyVersion = %d, want default 1", cfg.TOTPEncryptionKeyVersion)
	}
}

func TestLoadProductionDefaultsToVerifyFull(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "8081")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_NAME", "school_management")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("DB_SSLMODE", "")

	cfg := Load()

	if cfg.DBSSLMode != "verify-full" {
		t.Errorf(
			"DBSSLMode = %q, want verify-full",
			cfg.DBSSLMode,
		)
	}

	if err := cfg.ValidateDatabase(); err != nil {
		t.Errorf("ValidateDatabase() unexpected error: %v", err)
	}
}

func TestLoadExplicitSSLMode(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "8081")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_NAME", "school_management")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("DB_SSLMODE", "verify-full")

	cfg := Load()

	if cfg.DBSSLMode != "verify-full" {
		t.Errorf(
			"DBSSLMode = %q, want verify-full",
			cfg.DBSSLMode,
		)
	}
}

func validJWTTestConfig() Config {
	return Config{
		JWTSecret:              "test-secret-with-at-least-32-characters",
		JWTIssuer:              "school-management-system",
		JWTAccessExpiryMinutes: 15,
		RefreshTokenExpiryDays: 7,
	}
}

func TestValidateJWT(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid configuration",
			wantErr: false,
		},
		{
			name: "secret too short",
			modify: func(cfg *Config) {
				cfg.JWTSecret = "short"
			},
			wantErr: true,
		},
		{
			name: "empty issuer",
			modify: func(cfg *Config) {
				cfg.JWTIssuer = ""
			},
			wantErr: true,
		},
		{
			name: "expiry too low",
			modify: func(cfg *Config) {
				cfg.JWTAccessExpiryMinutes = 0
			},
			wantErr: true,
		},
		{
			name: "expiry too high",
			modify: func(cfg *Config) {
				cfg.JWTAccessExpiryMinutes = 61
			},
			wantErr: true,
		},
		{
			name: "refresh expiry too low",
			modify: func(cfg *Config) {
				cfg.RefreshTokenExpiryDays = 0
			},
			wantErr: true,
		},
		{
			name: "refresh expiry too high",
			modify: func(cfg *Config) {
				cfg.RefreshTokenExpiryDays = 31
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validJWTTestConfig()

			if tt.modify != nil {
				tt.modify(&cfg)
			}

			err := cfg.ValidateJWT()

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"ValidateJWT() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestTOTPEncryptionKeyConfiguration(t *testing.T) {
	t.Setenv("TOTP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "3")
	cfg := Load()
	if err := cfg.ValidateTOTPEncryption(); err != nil {
		t.Fatalf("valid TOTP encryption configuration rejected: %v", err)
	}
	if cfg.TOTPEncryptionKeyVersion != 3 {
		t.Fatalf("key version = %d, want 3", cfg.TOTPEncryptionKeyVersion)
	}
}

func TestTOTPEncryptionKeyConfigurationRejectsInvalidValues(t *testing.T) {
	for _, encoded := range []string{"", "not-valid-base64!", base64.StdEncoding.EncodeToString(make([]byte, 31))} {
		t.Setenv("TOTP_ENCRYPTION_KEY", encoded)
		t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "1")
		if err := Load().ValidateTOTPEncryption(); err == nil {
			t.Fatalf("expected invalid key %q to be rejected", encoded)
		}
	}
	t.Setenv("TOTP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "0")
	if err := Load().ValidateTOTPEncryption(); err == nil {
		t.Fatal("expected zero key version to be rejected")
	}
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "2147483648")
	if err := Load().ValidateTOTPEncryption(); err == nil {
		t.Fatal("expected key version outside PostgreSQL INTEGER range to be rejected")
	}
}

func TestParentOTPHMACKeyConfiguration(t *testing.T) {
	key := make([]byte, 32)
	t.Setenv("PARENT_OTP_HMAC_KEY", base64.StdEncoding.EncodeToString(key))
	cfg := Load()
	if err := cfg.ValidateParentOTPHMAC(); err != nil {
		t.Fatalf("valid Parent OTP HMAC key rejected: %v", err)
	}
	if len(cfg.ParentOTPHMACKey) != 32 {
		t.Fatalf("Parent OTP HMAC key length = %d, want 32", len(cfg.ParentOTPHMACKey))
	}
	if string(cfg.ParentOTPHMACKey) == string(cfg.TOTPEncryptionKey) {
		t.Fatal("Parent OTP HMAC key must be separately configured from TOTP encryption key")
	}
}

func TestParentOTPHMACKeyConfigurationRejectsMissingOrInvalidValues(t *testing.T) {
	for _, encoded := range []string{"", "invalid-base64!", base64.StdEncoding.EncodeToString(make([]byte, 31))} {
		t.Setenv("PARENT_OTP_HMAC_KEY", encoded)
		if err := Load().ValidateParentOTPHMAC(); err == nil {
			t.Fatalf("expected invalid Parent OTP HMAC key %q to be rejected", encoded)
		}
	}
}

func TestTOTPPreviousEncryptionKeyConfiguration(t *testing.T) {
	current := make([]byte, 32)
	previous := make([]byte, 32)
	previous[0] = 1
	t.Setenv("TOTP_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(current))
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "4")
	t.Setenv("TOTP_ENCRYPTION_PREVIOUS_KEYS", "2:"+base64.StdEncoding.EncodeToString(previous))
	cfg := Load()
	if err := cfg.ValidateTOTPEncryption(); err != nil {
		t.Fatalf("valid previous TOTP key rejected: %v", err)
	}
	if len(cfg.TOTPEncryptionPreviousKeys) != 1 || cfg.TOTPEncryptionPreviousKeys[0].Version != 2 {
		t.Fatalf("previous keys not loaded correctly: %#v", cfg.TOTPEncryptionPreviousKeys)
	}
}

func TestTOTPPreviousEncryptionKeyConfigurationRejectsMalformedEntries(t *testing.T) {
	current := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("TOTP_ENCRYPTION_KEY", current)
	t.Setenv("TOTP_ENCRYPTION_KEY_VERSION", "4")
	validPrevious := make([]byte, 32)
	encoded := base64.StdEncoding.EncodeToString(validPrevious)
	for _, raw := range []string{
		"broken",
		"0:"+encoded,
		"2:not-base64!",
		"2:"+base64.StdEncoding.EncodeToString(make([]byte, 31)),
		"2:"+encoded+",2:"+encoded,
		"4:"+encoded,
	} {
		t.Setenv("TOTP_ENCRYPTION_PREVIOUS_KEYS", raw)
		if err := Load().ValidateTOTPEncryption(); err == nil {
			t.Errorf("expected previous key configuration %q to be rejected", raw)
		}
	}
}
