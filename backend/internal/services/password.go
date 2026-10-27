package services

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var ErrInvalidPassword = errors.New("password must be between 8 and 72 bytes")

func HashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 72 {
		return "", ErrInvalidPassword
	}

	if strings.TrimSpace(password) == "" {
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
	if len(password) < 1 || len(password) > 72 {
		return false
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	)

	return err == nil
}
