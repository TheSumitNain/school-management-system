package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/database"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"
)

func main() {
	if os.Getenv("APP_ENV") != "development" {
		log.Fatal("devseed can only run when APP_ENV=development")
	}

	schoolID := int64(1)
	username := strings.TrimSpace(os.Getenv("DEVSEED_USERNAME"))
	password := os.Getenv("DEVSEED_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("Set DEVSEED_USERNAME and DEVSEED_PASSWORD first")
	}

	cfg := config.Load()

	if err := cfg.ValidateDatabase(); err != nil {
		log.Fatal("Database configuration is invalid")
	}

	if len(password) < 8 || len(password) > 72 {
		log.Fatal("Password must be between 8 and 72 bytes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatal("Could not connect to database")
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	schoolRepo := repository.NewSchoolRepository(db)
	authService := services.NewAuthService(userRepo, schoolRepo)

	user, err := authService.CreateUser(ctx, schoolID, username, password)
	if err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			log.Fatal("This username already exists in the school")
		}
		if errors.Is(err, services.ErrSchoolInactive) {
			log.Fatal("The school is not active")
		}
		if errors.Is(err, services.ErrInvalidPassword) ||
			errors.Is(err, services.ErrInvalidRegistration) {
			log.Fatal("Username or password does not meet validation rules")
		}
		log.Fatal("Could not create development user")
	}

	fmt.Printf("Development user created successfully: ID=%d, SchoolID=%d, Username=%s\n",
		user.ID, user.SchoolID, user.Username)
}
