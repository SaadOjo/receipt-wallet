package main

import (
	"fmt"
	"log"

	"fake-cash-register/internal/config"
	"fake-cash-register/internal/crypto"
	"fake-cash-register/internal/handlers"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/services"
	"fake-cash-register/internal/services/mock"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize services based on standalone mode
	serviceContainer := initializeServices(cfg)

	// Initialize handlers
	handler := handlers.NewCashRegisterHandler(cfg, serviceContainer)

	// Set up Gin router with logging based on verbose config
	var router *gin.Engine
	if cfg.Server.Verbose {
		gin.SetMode(gin.DebugMode)
		router = gin.Default()
		log.Printf("Verbose mode enabled - HTTP requests will be logged")
	} else {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()
		router.Use(gin.Recovery())
	}

	// Load HTML templates
	router.LoadHTMLGlob("web/templates/*")
	router.Static("/static", "./web/static")

	// Define routes
	// Web UI
	router.GET("/", handler.HomePage)
	
	// API routes
	api := router.Group("/api")
	{
		// Kisim management  
		api.GET("/kisim", handler.GetKisim)
		
		// Transaction management
		tx := api.Group("/transaction")
		{
			tx.POST("/start", handler.StartTransaction)
			tx.POST("/add-item", handler.AddItem)
			tx.POST("/set-quantity", handler.SetQuantity)
			tx.POST("/payment", handler.SetPaymentMethod)
			tx.POST("/generate-receipt", handler.GenerateReceipt)
			tx.POST("/process", handler.ProcessTransaction)
			tx.POST("/cancel", handler.CancelTransaction)
			tx.GET("/current", handler.GetCurrentTransaction)
		}
	}
	
	// Webhook endpoint
	router.POST("/webhook", handler.WebhookHandler)
	
	// Health check
	router.GET("/health", handler.HealthCheck)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting fake cash register on port %d", cfg.Server.Port)
	
	if cfg.StandaloneMode {
		log.Printf("Running in STANDALONE mode - no external services required")
	} else {
		log.Printf("Running in ONLINE mode - connecting to external services")
		log.Printf("  Revenue Authority: %s", cfg.RevenueAuthority.URL)
		log.Printf("  Receipt Bank: %s", cfg.ReceiptBank.URL)
	}

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initializeServices(cfg *config.Config) *interfaces.ServiceContainer {
	var revenueAuthority interfaces.RevenueAuthorityService
	var receiptBank interfaces.ReceiptBankService
	var qrScanner interfaces.QRScannerService
	var cryptoService interfaces.CryptoService

	if cfg.StandaloneMode {
		// Use mock services
		revenueAuthority = mock.NewMockRevenueAuthority(cfg.Server.Verbose)
		receiptBank = mock.NewMockReceiptBank(cfg.Server.Verbose)
		qrScanner = mock.NewMockQRScanner(cfg.Server.Verbose)
		cryptoService = crypto.NewMockCryptoService(cfg.Server.Verbose)
		
		// Set up webhook handler for mock receipt bank
		webhookHandler := handlers.NewWebhookHandler(cfg.Server.Verbose)
		receiptBank.SetWebhookHandler(webhookHandler)
		
		if cfg.Server.Verbose {
			log.Printf("Initialized MOCK services for standalone mode")
		}
	} else {
		// Use real services (to be implemented)
		// For now, fall back to mock services
		log.Printf("WARNING: Real service implementations not yet available, using mocks")
		
		revenueAuthority = mock.NewMockRevenueAuthority(cfg.Server.Verbose)
		receiptBank = mock.NewMockReceiptBank(cfg.Server.Verbose)
		qrScanner = mock.NewMockQRScanner(cfg.Server.Verbose)
		cryptoService = crypto.NewCryptoService(cfg.Server.Verbose)
		
		webhookHandler := handlers.NewWebhookHandler(cfg.Server.Verbose)
		receiptBank.SetWebhookHandler(webhookHandler)
	}

	// Initialize transaction service
	transactionService := services.NewTransactionService(
		revenueAuthority,
		receiptBank,
		cryptoService,
		cfg.Server.Verbose,
	)

	return &interfaces.ServiceContainer{
		RevenueAuthority: revenueAuthority,
		ReceiptBank:     receiptBank,
		QRScanner:       qrScanner,
		Crypto:          cryptoService,
		Transaction:     transactionService,
	}
}