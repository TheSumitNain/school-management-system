package config

import (
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

	return nil
}
