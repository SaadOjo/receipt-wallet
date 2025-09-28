package main

import (
	"log"
	"net"

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

	// Get LAN IP address
	lanIP := getLANIPAddress()
	log.Printf("[MAIN] Receipt Bank ready - listening on port %d", cfg.Server.Port)
	log.Printf("[MAIN] Service accessible at:")
	log.Printf("[MAIN]   Local:  http://localhost:%d", cfg.Server.Port)
	if lanIP != "" {
		log.Printf("[MAIN]   LAN:    http://%s:%d", lanIP, cfg.Server.Port)
	}
	log.Printf("[MAIN] API endpoints:")
	log.Printf("[MAIN]   POST /submit")
	log.Printf("[MAIN]   GET  /collect/{ephemeral_key}")
	log.Printf("[MAIN]   GET  /health")

	if err := srv.Start(cfg.Server.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// getLANIPAddress returns the local network IP address
func getLANIPAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
