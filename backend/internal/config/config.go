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

	return Config{
		AppEnv:     appEnv,
		AppPort:    appPort,
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBName:     os.Getenv("DB_NAME"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBSSLMode:  dbSSLMode,
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
