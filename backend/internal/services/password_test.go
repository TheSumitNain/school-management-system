package services

import (
	"errors"
	"strings"
	"testing"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password := "School@Secure2026"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned error: %v", err)
	}

	if hash == password {
		t.Fatal("password must not be stored as plain text")
	}

	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}

	if !CheckPassword(password, hash) {
		t.Fatal("CheckPassword() rejected the correct password")
	}
}

func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("School@Secure2026")
	if err != nil {
		t.Fatalf("HashPassword() returned error: %v", err)
	}

	if CheckPassword("Wrong@Secure2026", hash) {
		t.Fatal("CheckPassword() accepted an incorrect password")
	}
}

func TestHashPasswordRejectsInvalidPasswords(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "empty", password: ""},
		{name: "too short", password: "Abc123"},
		{name: "eleven bytes", password: "Abcdef12345"},
		{name: "whitespace only", password: "            "},
		{name: "too long", password: strings.Repeat("a", 73)},
		{name: "common password", password: "Password1234"},
		{name: "common password case insensitive", password: "PASSWORD1234"},
		{name: "common password with symbol", password: "Password@123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashPassword(tt.password)

			if !errors.Is(err, ErrInvalidPassword) {
				t.Fatalf("expected ErrInvalidPassword, got %v", err)
			}
		})
	}
}

func TestCheckPasswordRejectsInvalidInput(t *testing.T) {
	if CheckPassword("", "$2a$12$invalid") {
		t.Fatal("empty password must be rejected")
	}

	if CheckPassword(strings.Repeat("a", 73), "$2a$12$invalid") {
		t.Fatal("password longer than 72 bytes must be rejected")
	}

	if CheckPassword("valid-password", "not-a-bcrypt-hash") {
		t.Fatal("invalid hash must not be accepted")
	}
}

func TestCheckPasswordStillVerifiesLegacyShortPasswordHash(t *testing.T) {
	legacyHash, err := bcrypt.GenerateFromPassword([]byte("Legacy8!"), bcryptCost)
	if err != nil {
		t.Fatalf("generate legacy hash: %v", err)
	}

	if !CheckPassword("Legacy8!", string(legacyHash)) {
		t.Fatal("existing bcrypt hash for a legacy password must remain verifiable")
	}
}
