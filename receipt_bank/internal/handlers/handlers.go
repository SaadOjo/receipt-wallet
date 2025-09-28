package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"receipt-bank/internal/models"
	"receipt-bank/internal/storage"
	"receipt-bank/internal/webhook"
)

// Handler contains dependencies for HTTP handlers
type Handler struct {
	storage       *storage.MemoryStorage
	webhookClient *webhook.Client
	verbose       bool
}

// NewHandler creates a new handler instance
func NewHandler(storage *storage.MemoryStorage, webhookClient *webhook.Client, verbose bool) *Handler {
	return &Handler{
		storage:       storage,
		webhookClient: webhookClient,
		verbose:       verbose,
	}
}

// SubmitHandler handles POST /submit
func (h *Handler) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	var req models.SubmitRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := req.Validate(); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create receipt
	receipt := &models.Receipt{
		EphemeralKey:  req.EphemeralKey,
		EncryptedData: req.EncryptedData,
		ReceiptID:     req.ReceiptID,
		WebhookURL:    req.WebhookURL,
		Timestamp:     time.Now(),
	}

	// Store receipt
	if err := h.storage.Store(receipt); err != nil {
		if err.Error() == "receipt_id already exists" {
			h.writeError(w, http.StatusConflict, "Receipt ID already exists")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to store receipt")
		}
		return
	}

	if h.verbose {
		log.Printf("[API] Receipt submitted successfully: %s", req.ReceiptID)
	}

	// Return success response
	resp := models.SubmitResponse{
		ReceiptID: req.ReceiptID,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// CollectHandler handles GET /collect/{ephemeral_key}
func (h *Handler) CollectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ephemeralKey := vars["ephemeral_key"]

	// Validate ephemeral key format
	if err := models.ValidateEphemeralKey(ephemeralKey); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Retrieve receipt
	receipt, err := h.storage.Retrieve(ephemeralKey)
	if err != nil {
		if err.Error() == "receipt not found" {
			h.writeError(w, http.StatusNotFound, "No receipt found for given ephemeral key")
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to retrieve receipt")
		}
		return
	}

	if h.verbose {
		log.Printf("[API] Receipt collected successfully: %s", receipt.ReceiptID)
	}

	// Send webhook notification (non-blocking)
	go func() {
		if err := h.webhookClient.NotifyCollection(receipt.WebhookURL, receipt.ReceiptID); err != nil {
			log.Printf("[WEBHOOK] Failed to notify collection: %v", err)
		}
	}()

	// Return success response
	resp := models.CollectResponse{
		EncryptedData: receipt.EncryptedData,
		ReceiptID:     receipt.ReceiptID,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HealthHandler handles GET /health
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	total, expired := h.storage.Stats()

	status := map[string]interface{}{
		"status":           "healthy",
		"receipts_stored":  total,
		"receipts_expired": expired,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, status)
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] Failed to write JSON response: %v", err)
	}
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	if h.verbose {
		log.Printf("[API] Error %d: %s", status, message)
	}

	resp := models.ErrorResponse{
		Error: message,
	}

	h.writeJSON(w, status, resp)
}
