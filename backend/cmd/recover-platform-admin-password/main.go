package main

import (
	"bufio"
	"context"
	"crypto/subtle"
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

const recoveryConfirmation = "RESET_EXISTING_PLATFORM_ADMIN"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if os.Getenv("PLATFORM_ADMIN_RECOVERY_CONFIRM") != recoveryConfirmation {
		return fmt.Errorf("set PLATFORM_ADMIN_RECOVERY_CONFIRM=%s to confirm this recovery operation", recoveryConfirmation)
	}

	email := strings.ToLower(strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_EMAIL")))
	if email == "" || len(email) > 254 {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL must contain the existing Platform Admin email")
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL must be a single valid email address")
	}

	cfg := config.Load()
	if err := cfg.ValidateDatabase(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("run this command from an interactive terminal; passwords must not be piped or supplied as environment variables")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("To confirm recovery, type the existing Platform Admin email (%s): ", email)
	typedEmail, err := reader.ReadString('\n')
	if err != nil && len(typedEmail) == 0 {
		return fmt.Errorf("read email confirmation: %w", err)
	}
	if strings.TrimSpace(typedEmail) != email {
		return fmt.Errorf("email confirmation did not match; no changes were made")
	}

	password, err := readHiddenPassword("New Platform Admin password (input hidden): ")
	if err != nil {
		return err
	}
	defer clear(password)

	confirmation, err := readHiddenPassword("Re-enter new Platform Admin password: ")
	if err != nil {
		clear(confirmation)
		return err
	}
	defer clear(confirmation)

	if subtle.ConstantTimeCompare(password, confirmation) != 1 {
		return fmt.Errorf("passwords did not match; no changes were made")
	}

	passwordCtx, passwordCancel := context.WithTimeout(ctx, 45*time.Second)
	passwordHash, err := services.HashPasswordContext(passwordCtx, string(password))
	passwordCancel()
	if err != nil {
		return fmt.Errorf("new password does not meet the approved password policy: %w", err)
	}

	principalID, err := repository.NewPlatformAdminRepository(db).RecoverPasswordByEmail(ctx, email, passwordHash)
	if err != nil {
		return err
	}

	fmt.Printf("Existing Platform Admin password recovered successfully. Principal ID: %d; Email: %s\n", principalID, email)
	fmt.Println("All previous access and refresh sessions were invalidated.")
	fmt.Println("Next: sign in at /auth/platform-admin/login and complete the mandatory MFA challenge or enrollment.")
	return nil
}

func readHiddenPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("read password securely: %w", err)
	}
	return password, nil
}

func clear(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
