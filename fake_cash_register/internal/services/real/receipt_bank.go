package real

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"fake-cash-register/internal/api"
	"fake-cash-register/internal/config"
	"fake-cash-register/internal/interfaces"
)

type RealReceiptBank struct {
	baseURL        string
	httpClient     *http.Client
	webhookHandler interfaces.WebhookHandler
	cfg            *config.Config
	verbose        bool
}

func NewRealReceiptBank(baseURL string, cfg *config.Config, verbose bool) *RealReceiptBank {
	return &RealReceiptBank{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		cfg:     cfg,
		verbose: verbose,
	}
}

// SubmitReceipt sends encrypted receipt to external receipt bank
func (r *RealReceiptBank) SubmitReceipt(userEphemeralKeyCompressed []byte, encryptedData []byte) error {
	// Convert binary data to base64 for API transmission
	keyBase64 := base64.StdEncoding.EncodeToString(userEphemeralKeyCompressed)
	encryptedDataBase64 := base64.StdEncoding.EncodeToString(encryptedData)

	if r.verbose {
		log.Printf("[REAL] Receipt Bank: Submitting receipt (privacy-preserving)")
		log.Printf("[REAL] User Ephemeral Key: %s... (%d bytes compressed)", keyBase64[:16], len(userEphemeralKeyCompressed))
		log.Printf("[REAL] Encrypted Data: %d bytes", len(encryptedData))
	}

	// Generate receipt ID for submission tracking
	receiptID := fmt.Sprintf("%d", time.Now().Unix())

	// Construct webhook URL for receipt bank callbacks
	webhookURL := fmt.Sprintf("http://%s:%d/webhook", r.cfg.Server.WebhookHost, r.cfg.Server.WebhookPort)

	// Prepare request
	submission := api.ReceiptSubmission{
		EphemeralKey:  keyBase64,
		EncryptedData: encryptedDataBase64,
		ReceiptID:     receiptID,
		WebhookURL:    webhookURL,
	}

	requestBody, err := json.Marshal(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal receipt submission: %v", err)
	}

	// Make HTTP request
	url := r.baseURL + "/submit"
	resp, err := r.httpClient.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to call receipt bank at %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse error response
		var errorResp api.ErrorResponse
		if json.Unmarshal(responseBody, &errorResp) == nil {
			return fmt.Errorf("receipt bank error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return fmt.Errorf("receipt bank returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	var bankResp api.ReceiptBankResponse
	if err := json.Unmarshal(responseBody, &bankResp); err != nil {
		return fmt.Errorf("failed to parse receipt bank response: %v", err)
	}

	if r.verbose {
		log.Printf("[REAL] Receipt Bank: Receipt submitted successfully with ID: %s", bankResp.ReceiptID)
	}

	return nil
}

// SetWebhookHandler configures the webhook handler for receipt confirmations
func (r *RealReceiptBank) SetWebhookHandler(handler interfaces.WebhookHandler) {
	r.webhookHandler = handler
	if r.verbose {
		log.Printf("[REAL] Receipt Bank: Webhook handler registered")
	}
}
