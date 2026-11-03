package main

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

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
	if email == "" {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL environment variable is required")
	}
	if len(email) > 254 {
		return fmt.Errorf("Platform Admin email must not exceed 254 characters")
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL must be a single valid email address")
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("run this command from an interactive terminal so the initial password can be entered securely")
	}
	fmt.Print("Initial Platform Admin password (input hidden): ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("read initial Platform Admin password: %w", err)
	}
	passwordHash, err := services.HashPassword(string(passwordBytes))
	for i := range passwordBytes {
		passwordBytes[i] = 0
	}
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
