package main

import (
	"fmt"
	"log"

	"fake-cash-register/internal/cashregister"
	"fake-cash-register/internal/config"
	"fake-cash-register/internal/crypto"
	"fake-cash-register/internal/handlers"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
	"fake-cash-register/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create store info
	storeInfo := interfaces.StoreInfo{
		VKN:     cfg.Store.VKN,
		Name:    cfg.Store.Name,
		Address: cfg.Store.Address,
	}

	// Create KISIM lookup
	kisimLookup := make(models.KisimLookup)
	for _, k := range cfg.Kisim {
		kisimLookup[k.ID] = models.KisimInfo{
			ID:          k.ID,
			Name:        k.Name,
			TaxRate:     k.TaxRate,
			PresetPrice: k.PresetPrice,
		}
	}

	// Initialize services based on configuration (factory pattern)
	cryptoService := crypto.NewCryptoService(cfg.Server.Verbose)
	revenueAuthority, receiptBank, err := services.CreateServices(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Set up webhook handlers for online mode
	if !cfg.StandaloneMode {
		// TODO: Implement webhook handler for real receipt bank confirmations
		// For now, we'll use the mock webhook handler
		// webhookHandler := real.NewWebhookHandler(cfg.Server.Verbose)
		// receiptBank.SetWebhookHandler(webhookHandler)
	}

	if cfg.Server.Verbose {
		if cfg.StandaloneMode {
			log.Printf("Initialized MOCK services for standalone mode")
		} else {
			log.Printf("Initialized REAL services for online mode")
		}
	}

	// Initialize CashRegister with all services directly
	cashReg := cashregister.NewCashRegister(
		storeInfo,
		kisimLookup,
		revenueAuthority,
		receiptBank,
		cryptoService,
		cfg.Server.Verbose,
	)

	// Initialize handlers
	handler := handlers.NewCashRegisterHandler(cashReg, cfg)

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
			tx.POST("/payment", handler.SetPaymentMethod)
			tx.POST("/issue_receipt", handler.IssueReceipt)
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
