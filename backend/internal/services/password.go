package services

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost        = 12
	minPasswordLength = 12
	maxPasswordLength = 72
)

var ErrInvalidPassword = errors.New("password must be between 12 and 72 bytes and must not be a commonly used password")

// commonPasswords is a deliberately small local denylist, not a substitute
// for a breached-password intelligence service. Keep comparisons
// case-insensitive and do not normalize the actual password before hashing.
var commonPasswords = map[string]struct{}{
	"123456789012": {},
	"1234567890":   {},
	"123456789":    {},
	"password123":  {},
	"password1234": {},
	"password@123": {},
	"qwerty123456": {},
	"qwerty123":    {},
	"admin123456":  {},
	"welcome1234":  {},
	"letmein1234":  {},
	"iloveyou123":  {},
	"changeme123":  {},
}

func HashPassword(password string) (string, error) {
	if !validNewPassword(password) {
		return "", ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcryptCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(password, passwordHash string) bool {
	if len(password) < 1 || len(password) > maxPasswordLength {
		return false
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	return err == nil
}

func validNewPassword(password string) bool {
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return false
	}
	if strings.TrimSpace(password) == "" {
		return false
	}
	_, common := commonPasswords[strings.ToLower(password)]
	return !common
}
