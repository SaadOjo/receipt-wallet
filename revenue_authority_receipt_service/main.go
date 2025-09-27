package main

import (
	"fmt"
	"log"

	"revenue-authority-receipt-service/config"
	"revenue-authority-receipt-service/crypto"
	"revenue-authority-receipt-service/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize crypto service
	cryptoService := crypto.NewCryptoService(
		cfg.Keys.PrivateKeyPath,
		cfg.Keys.PublicKeyPath,
	)

	// Initialize handlers
	handler := handlers.NewHandler(cryptoService)

	// Set up Gin router with logging based on verbose config
	var router *gin.Engine
	if cfg.Server.Verbose {
		gin.SetMode(gin.DebugMode)
		router = gin.Default() // Includes Logger() and Recovery() middleware
		log.Printf("Verbose mode enabled - HTTP requests will be logged")
	} else {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New() // No default middleware in production
		router.Use(gin.Recovery()) // Still use recovery middleware for safety
	}

	// Define routes
	router.POST("/sign", handler.SignHash)
	router.GET("/public-key", handler.GetPublicKey)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting revenue authority receipt service on port %d", cfg.Server.Port)
	
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}