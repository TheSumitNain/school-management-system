package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/database"

	"school-management-system/internal/handlers"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type databasePinger interface {
	Ping(ctx context.Context) error
}

func setupRouter(
	db databasePinger,
	authHandler *handlers.AuthHandler,
) (*gin.Engine, error) {
	router := gin.Default()

	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	if authHandler != nil {
		router.POST("/auth/login", authHandler.Login)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "School Management System API is running",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Database is not ready",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "API and database are ready",
		})
	})

	return router, nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	if err := cfg.ValidateDatabase(); err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	if err := cfg.ValidateJWT(); err != nil {
		return fmt.Errorf("invalid JWT configuration: %w", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.Connect(ctx, cfg)
	cancel()

	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	userRepo := repository.NewUserRepository(db)
	schoolRepo := repository.NewSchoolRepository(db)

	authService := services.NewAuthService(userRepo, schoolRepo)

	tokenService, err := services.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAccessExpiryMinutes,
	)
	if err != nil {
		return fmt.Errorf("initialize token service: %w", err)
	}

	authHandler := handlers.NewAuthHandler(
		authService,
		tokenService,
	)

	router, err := setupRouter(db, authHandler)

	if err != nil {
		return fmt.Errorf("setup router: %w", err)
	}

	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.AppPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	log.Printf("Starting API in %s mode on port %s", cfg.AppEnv, cfg.AppPort)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil

	case <-stop:
		log.Println("Shutting down API server...")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil &&
			!errors.Is(closeErr, net.ErrClosed) {
			log.Printf("Failed to force close server: %v", closeErr)
		}
	}

	err = <-serverErr
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server stopped unexpectedly: %w", err)
	}

	log.Println("API server stopped")
	return nil
}
