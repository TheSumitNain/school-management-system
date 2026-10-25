package main

import (
	"log"
	"net/http"

	"school-management-system/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

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
