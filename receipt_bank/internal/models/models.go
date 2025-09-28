package models

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"time"
)

// SubmitRequest represents the receipt submission request
type SubmitRequest struct {
	EphemeralKey  string `json:"ephemeral_key"`
	EncryptedData string `json:"encrypted_data"`
	ReceiptID     string `json:"receipt_id"`
	WebhookURL    string `json:"webhook_url"`
}

// SubmitResponse represents the receipt submission response
type SubmitResponse struct {
	ReceiptID string `json:"receipt_id"`
}

// CollectResponse represents the receipt collection response
type CollectResponse struct {
	EncryptedData string `json:"encrypted_data"`
	ReceiptID     string `json:"receipt_id"`
}

// WebhookPayload represents the payload sent to cash register webhook
type WebhookPayload struct {
	ReceiptID string `json:"receipt_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Receipt represents a stored receipt
type Receipt struct {
	EphemeralKey  string    `json:"ephemeral_key"`
	EncryptedData string    `json:"encrypted_data"`
	ReceiptID     string    `json:"receipt_id"`
	WebhookURL    string    `json:"webhook_url"`
	Timestamp     time.Time `json:"timestamp"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// receiptIDRegex matches alphanumeric characters and hyphens only
var receiptIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// ValidateSubmitRequest validates a submit request
func (req *SubmitRequest) Validate() error {
	// Validate ephemeral key
	if req.EphemeralKey == "" {
		return fmt.Errorf("ephemeral_key is required")
	}

	ephemeralKeyBytes, err := base64.StdEncoding.DecodeString(req.EphemeralKey)
	if err != nil {
		return fmt.Errorf("ephemeral_key must be valid base64")
	}

	if len(ephemeralKeyBytes) != 33 {
		return fmt.Errorf("ephemeral_key must decode to exactly 33 bytes")
	}

	// Validate encrypted data
	if req.EncryptedData == "" {
		return fmt.Errorf("encrypted_data is required")
	}

	if _, err := base64.StdEncoding.DecodeString(req.EncryptedData); err != nil {
		return fmt.Errorf("encrypted_data must be valid base64")
	}

	// Validate receipt ID
	if req.ReceiptID == "" {
		return fmt.Errorf("receipt_id is required")
	}

	if !receiptIDRegex.MatchString(req.ReceiptID) {
		return fmt.Errorf("receipt_id must contain only alphanumeric characters and hyphens")
	}

	// Validate webhook URL
	if req.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	webhookURL, err := url.Parse(req.WebhookURL)
	if err != nil {
		return fmt.Errorf("webhook_url must be a valid URL")
	}

	if webhookURL.Scheme != "http" && webhookURL.Scheme != "https" {
		return fmt.Errorf("webhook_url must use HTTP or HTTPS")
	}

	return nil
}

// ValidateEphemeralKey validates an ephemeral key for collection
func ValidateEphemeralKey(ephemeralKey string) error {
	if ephemeralKey == "" {
		return fmt.Errorf("ephemeral_key is required")
	}

	ephemeralKeyBytes, err := base64.StdEncoding.DecodeString(ephemeralKey)
	if err != nil {
		return fmt.Errorf("ephemeral_key must be valid base64")
	}

	if len(ephemeralKeyBytes) != 33 {
		return fmt.Errorf("ephemeral_key must decode to exactly 33 bytes")
	}

	return nil
}
