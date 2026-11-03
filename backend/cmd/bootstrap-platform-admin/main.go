package main

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/database"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"
)

const confirmationValue = "CREATE_FIRST_PLATFORM_ADMIN"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if os.Getenv("PLATFORM_ADMIN_BOOTSTRAP_CONFIRM") != confirmationValue {
		return fmt.Errorf("set PLATFORM_ADMIN_BOOTSTRAP_CONFIRM=%s to confirm this one-time operation", confirmationValue)
	}

	email := strings.ToLower(strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_EMAIL")))
	password := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL and PLATFORM_ADMIN_PASSWORD environment variables are required")
	}
	if len(email) > 254 {
		return fmt.Errorf("Platform Admin email must not exceed 254 characters")
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL must be a single valid email address")
	}

	passwordHash, err := services.HashPassword(password)
	// Remove the plaintext from this process environment as soon as it is hashed.
	_ = os.Unsetenv("PLATFORM_ADMIN_PASSWORD")
	password = ""
	if err != nil {
		return fmt.Errorf("initial Platform Admin password does not meet policy: %w", err)
	}

	cfg := config.Load()
	if err := cfg.ValidateDatabase(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	principalID, err := repository.NewPlatformAdminRepository(db).BootstrapFirst(ctx, email, passwordHash)
	if err != nil {
		return err
	}

	fmt.Printf("Initial Platform Admin created successfully. Principal ID: %d; Email: %s\n", principalID, email)
	fmt.Println("Next: start the API and complete mandatory TOTP MFA enrollment at /auth/platform-admin/login.")
	return nil
}
