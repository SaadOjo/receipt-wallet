package handlers

import (
	"encoding/base64"
	"log"
	"net/http"

	"fake-cash-register/internal/api"
	"fake-cash-register/internal/cashregister"
	"fake-cash-register/internal/config"
	"fake-cash-register/internal/models"

	"github.com/gin-gonic/gin"
)

type CashRegisterHandler struct {
	cashRegister *cashregister.CashRegister
	config       *config.Config
}

func NewCashRegisterHandler(
	cashReg *cashregister.CashRegister,
	cfg *config.Config,
) *CashRegisterHandler {
	return &CashRegisterHandler{
		cashRegister: cashReg,
		config:       cfg,
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
			PresetPrice: k.PresetPrice,
		}
	}

	c.JSON(http.StatusOK, models.KisimResponse{
		Kisim: kisim,
	})
}

// POST /api/transaction/start - Start new transaction
func (h *CashRegisterHandler) StartTransaction(c *gin.Context) {
	if h.config.Server.Verbose {
		log.Printf("[HANDLER] Starting new transaction")
	}

	h.cashRegister.StartNewReceipt()

	c.Status(http.StatusCreated) // 201 - Receipt created
}

// POST /api/transaction/add-item - Add item to current transaction
func (h *CashRegisterHandler) AddItem(c *gin.Context) {
	var req struct {
		KisimID   int     `json:"kisim_id" binding:"required"`
		Quantity  int     `json:"quantity" binding:"required"`
		UnitPrice float64 `json:"unit_price,omitempty"` // Optional custom price
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "Invalid request format",
			Code:  api.ErrorCodeInvalidRequest,
		})
		return
	}

	if !h.cashRegister.HasActiveReceipt() {
		h.cashRegister.StartNewReceipt()
	}

	err := h.cashRegister.AddItem(req.KisimID, req.Quantity, req.UnitPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.APIError{
			Error: err.Error(),
			Code:  api.ErrorCodeInternalError,
		})
		return
	}

	// Return current items after adding
	c.JSON(http.StatusOK, gin.H{
		"items": h.cashRegister.GetCurrentReceipt().Items,
	})
}

// POST /api/transaction/payment - Set payment method
func (h *CashRegisterHandler) SetPaymentMethod(c *gin.Context) {
	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "Invalid request format",
			Code:  api.ErrorCodeInvalidRequest,
		})
		return
	}

	if !h.cashRegister.HasActiveReceipt() {
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "No active transaction",
			Code:  api.ErrorCodeNoActiveReceipt,
		})
		return
	}

	err := h.cashRegister.SetPaymentMethod(req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.APIError{
			Error: err.Error(),
			Code:  api.ErrorCodeInternalError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_method": req.PaymentMethod,
	})
}

// POST /api/transaction/process - Process transaction with ephemeral key
func (h *CashRegisterHandler) ProcessTransaction(c *gin.Context) {
	var req struct {
		EphemeralKey string `json:"ephemeral_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "Invalid request format",
			Code:  api.ErrorCodeInvalidRequest,
		})
		return
	}

	if !h.cashRegister.HasActiveReceipt() {
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "No active transaction",
			Code:  api.ErrorCodeNoActiveReceipt,
		})
		return
	}

	// Parse ephemeral key from base64
	ephemeralKeyCompressed, err := base64.StdEncoding.DecodeString(req.EphemeralKey)
	if err != nil {
		h.cancelTransaction()
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "Invalid ephemeral key format: " + err.Error(),
			Code:  api.ErrorCodeInvalidKey,
		})
		return
	}

	// Issue receipt (finalize + issue in one atomic operation)
	receipt, err := h.cashRegister.IssueCurrentReceipt(ephemeralKeyCompressed)
	if err != nil {
		h.cancelTransaction()
		c.JSON(http.StatusInternalServerError, api.APIError{
			Error: "Receipt issuing failed: " + err.Error(),
			Code:  api.ErrorCodeInternalError,
		})
		return
	}

	// Return receipt directly with HTTP 200
	c.JSON(http.StatusOK, receipt)
}

// POST /api/transaction/cancel - Cancel current transaction
func (h *CashRegisterHandler) CancelTransaction(c *gin.Context) {
	h.cancelTransaction()

	c.Status(http.StatusNoContent) // 204 - No content, operation successful
}

// GET /api/transaction/current - Get current transaction state
func (h *CashRegisterHandler) GetCurrentTransaction(c *gin.Context) {
	if !h.cashRegister.HasActiveReceipt() {
		c.JSON(http.StatusNotFound, api.APIError{
			Error: "No active transaction",
			Code:  api.ErrorCodeNoActiveReceipt,
		})
		return
	}

	// Return receipt directly
	c.JSON(http.StatusOK, h.cashRegister.GetCurrentReceipt())
}

// POST /webhook - Receipt bank webhook endpoint
func (h *CashRegisterHandler) WebhookHandler(c *gin.Context) {
	var payload api.WebhookPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		if h.config.Server.Verbose {
			log.Printf("[WEBHOOK] Invalid payload: %v", err)
		}
		c.JSON(http.StatusBadRequest, api.APIError{
			Error: "Invalid payload",
			Code:  api.ErrorCodeInvalidRequest,
		})
		return
	}

	if h.config.Server.Verbose {
		log.Printf("[WEBHOOK] Received confirmation for receipt %s: %s",
			payload.ReceiptID, payload.Status)
	}

	c.Status(http.StatusOK) // 200 - Webhook processed successfully
}

// GET /health - Health check
func (h *CashRegisterHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"service":         "fake-cash-register",
		"standalone_mode": h.config.StandaloneMode,
	})
}

// Helper methods
func (h *CashRegisterHandler) cancelTransaction() {
	h.cashRegister.CancelCurrentReceipt()
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
