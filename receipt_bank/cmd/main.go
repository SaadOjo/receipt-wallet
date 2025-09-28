package main

import (
	"log"

	"receipt-bank/internal/config"
	"receipt-bank/internal/handlers"
	"receipt-bank/internal/server"
	"receipt-bank/internal/storage"
	"receipt-bank/internal/webhook"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.Server.Verbose {
		log.Printf("[MAIN] Receipt Bank starting...")
		log.Printf("[MAIN] Configuration loaded from: config.yaml")
		log.Printf("[MAIN] Server port: %d", cfg.Server.Port)
		log.Printf("[MAIN] Cleanup interval: %v", cfg.CleanupInterval)
		log.Printf("[MAIN] Max receipt age: %v", cfg.MaxReceiptAge)
		log.Printf("[MAIN] Webhook timeout: %v", cfg.WebhookTimeout)
		log.Printf("[MAIN] Webhook max retries: %d", cfg.Webhooks.MaxRetries)
	}

	// Initialize storage
	storage := storage.NewMemoryStorage(cfg.MaxReceiptAge, cfg.Server.Verbose)
	storage.StartCleanupRoutine(cfg.CleanupInterval)

	// Initialize webhook client
	webhookClient := webhook.NewClient(cfg.WebhookTimeout, cfg.Webhooks.MaxRetries, cfg.Server.Verbose)

	// Initialize handlers
	handler := handlers.NewHandler(storage, webhookClient, cfg.Server.Verbose)

	// Initialize and start server
	srv := server.NewServer(handler, cfg.Server.Verbose)

	log.Printf("[MAIN] Receipt Bank ready - listening on port %d", cfg.Server.Port)
	if err := srv.Start(cfg.Server.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
