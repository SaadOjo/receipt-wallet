package handlers

import (
	"log"
	"net/http"

	"fake-cash-register/internal/config"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"

	"github.com/gin-gonic/gin"
)

type CashRegisterHandler struct {
	config      *config.Config
	services    *interfaces.ServiceContainer
	currentTx   *models.Transaction
	storeInfo   interfaces.StoreInfo
	verbose     bool
}

func NewCashRegisterHandler(cfg *config.Config, services *interfaces.ServiceContainer) *CashRegisterHandler {
	storeInfo := interfaces.StoreInfo{
		VKN:     cfg.Store.VKN,
		Name:    cfg.Store.Name,
		Address: cfg.Store.Address,
	}

	return &CashRegisterHandler{
		config:    cfg,
		services:  services,
		storeInfo: storeInfo,
		verbose:   cfg.Server.Verbose,
	}
}

// GET / - Main cash register UI
func (h *CashRegisterHandler) HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"StoreName":  h.config.Store.Name,
		"StoreVKN":   h.config.Store.VKN,
		"Kisim":      h.config.Kisim,
		"Verbose":    h.config.Server.Verbose,
		"Standalone": h.config.StandaloneMode,
	})
}

// GET /api/kisim - Get kisim list
func (h *CashRegisterHandler) GetKisim(c *gin.Context) {
	kisim := make([]models.KisimInfo, len(h.config.Kisim))
	for i, k := range h.config.Kisim {
		kisim[i] = models.KisimInfo{
			ID:          k.ID,
			Name:        k.Name,
			TaxRate:     k.TaxRate,
			Description: k.Description,
		}
	}

	c.JSON(http.StatusOK, models.KisimResponse{
		Kisim: kisim,
	})
}

// POST /api/transaction/start - Start new transaction
func (h *CashRegisterHandler) StartTransaction(c *gin.Context) {
	if h.verbose {
		log.Printf("[HANDLER] Starting new transaction")
	}

	h.currentTx = h.services.Transaction.StartTransaction()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transaction started",
		"transaction_id": h.currentTx.Status,
	})
}

// POST /api/transaction/add-item - Add item to current transaction
func (h *CashRegisterHandler) AddItem(c *gin.Context) {
	var req struct {
		KisimID     int     `json:"kisim_id" binding:"required"`
		KisimName   string  `json:"kisim_name" binding:"required"`
		UnitPrice   float64 `json:"unit_price" binding:"required"`
		TaxRate     int     `json:"tax_rate" binding:"required"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	if h.currentTx == nil {
		h.currentTx = h.services.Transaction.StartTransaction()
	}

	err := h.services.Transaction.AddItem(h.currentTx, req.KisimID, req.KisimName, req.UnitPrice, req.TaxRate, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Item added",
		"items":   h.currentTx.Items,
	})
}

// POST /api/transaction/set-quantity - Set quantity for item
func (h *CashRegisterHandler) SetQuantity(c *gin.Context) {
	var req struct {
		ItemIndex int `json:"item_index" binding:"required"`
		Quantity  int `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	if h.currentTx == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No active transaction",
		})
		return
	}

	err := h.services.Transaction.SetQuantity(h.currentTx, req.ItemIndex, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quantity updated",
		"items":   h.currentTx.Items,
	})
}

// POST /api/transaction/payment - Set payment method
func (h *CashRegisterHandler) SetPaymentMethod(c *gin.Context) {
	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	if h.currentTx == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No active transaction",
		})
		return
	}

	err := h.services.Transaction.SetPaymentMethod(h.currentTx, req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment method set",
		"payment_method": req.PaymentMethod,
	})
}

// POST /api/transaction/generate-receipt - Generate receipt preview
func (h *CashRegisterHandler) GenerateReceipt(c *gin.Context) {
	if h.currentTx == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No active transaction",
		})
		return
	}

	receipt, err := h.services.Transaction.GenerateReceipt(h.currentTx, h.storeInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TransactionResponse{
		Success: true,
		Message: "Receipt generated",
		Receipt: receipt,
	})
}

// POST /api/transaction/process - Process transaction with ephemeral key
func (h *CashRegisterHandler) ProcessTransaction(c *gin.Context) {
	var req struct {
		EphemeralKey string `json:"ephemeral_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format",
		})
		return
	}

	if h.currentTx == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No active transaction",
		})
		return
	}

	// Generate receipt
	receipt, err := h.services.Transaction.GenerateReceipt(h.currentTx, h.storeInfo)
	if err != nil {
		h.cancelTransaction()
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to generate receipt: " + err.Error(),
		})
		return
	}

	// Process transaction
	err = h.services.Transaction.ProcessTransaction(receipt, req.EphemeralKey)
	if err != nil {
		h.cancelTransaction()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Transaction processing failed: " + err.Error(),
		})
		return
	}

	// Clear current transaction
	h.currentTx = nil

	c.JSON(http.StatusOK, models.TransactionResponse{
		Success: true,
		Message: "Transaction processed successfully",
		Receipt: receipt,
	})
}

// POST /api/transaction/cancel - Cancel current transaction
func (h *CashRegisterHandler) CancelTransaction(c *gin.Context) {
	h.cancelTransaction()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transaction cancelled",
	})
}

// GET /api/transaction/current - Get current transaction state
func (h *CashRegisterHandler) GetCurrentTransaction(c *gin.Context) {
	if h.currentTx == nil {
		c.JSON(http.StatusOK, gin.H{
			"transaction": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": h.currentTx,
	})
}

// POST /webhook - Receipt bank webhook endpoint
func (h *CashRegisterHandler) WebhookHandler(c *gin.Context) {
	var payload interfaces.WebhookPayload
	
	if err := c.ShouldBindJSON(&payload); err != nil {
		if h.verbose {
			log.Printf("[WEBHOOK] Invalid payload: %v", err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	if h.verbose {
		log.Printf("[WEBHOOK] Received confirmation for receipt %s: %s", 
			payload.ReceiptID, payload.Status)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook processed",
	})
}

// GET /health - Health check
func (h *CashRegisterHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "fake-cash-register",
		"standalone_mode": h.config.StandaloneMode,
	})
}

// Helper methods
func (h *CashRegisterHandler) cancelTransaction() {
	if h.verbose && h.currentTx != nil {
		log.Printf("[HANDLER] Transaction cancelled")
	}
	h.currentTx = nil
}

// WebhookHandler implementation for services
type WebhookHandlerImpl struct {
	verbose bool
}

func NewWebhookHandler(verbose bool) *WebhookHandlerImpl {
	return &WebhookHandlerImpl{verbose: verbose}
}

func (w *WebhookHandlerImpl) HandleDownloadConfirmation(receiptID string) error {
	if w.verbose {
		log.Printf("[WEBHOOK] Download confirmed for receipt: %s", receiptID)
	}
	return nil
}