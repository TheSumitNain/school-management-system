package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/database"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected successfully")

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "School Management System API is running",
		})
	})

	address := ":" + cfg.AppPort

	log.Printf("Starting API in %s mode on port %s", cfg.AppEnv, cfg.AppPort)

	if err := router.Run(address); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
