package config

import "testing"

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
