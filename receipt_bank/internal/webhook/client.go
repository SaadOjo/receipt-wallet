package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"receipt-bank/internal/models"
)

// Client handles webhook notifications to cash registers
type Client struct {
	httpClient *http.Client
	maxRetries int
	verbose    bool
}

// NewClient creates a new webhook client
func NewClient(timeout time.Duration, maxRetries int, verbose bool) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
		verbose:    verbose,
	}
}

// NotifyCollection sends a webhook notification about receipt collection
func (c *Client) NotifyCollection(webhookURL, receiptID string) error {
	payload := models.WebhookPayload{
		ReceiptID: receiptID,
		Status:    "downloaded",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	return c.sendWebhook(webhookURL, payload)
}

// sendWebhook sends a webhook with retry logic
func (c *Client) sendWebhook(webhookURL string, payload models.WebhookPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff for retries
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)

			if c.verbose {
				log.Printf("[WEBHOOK] Retry attempt %d for receipt %s", attempt, payload.ReceiptID)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create webhook request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("webhook request failed: %v", err)
			if c.verbose {
				log.Printf("[WEBHOOK] Request failed for receipt %s: %v", payload.ReceiptID, err)
			}
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if c.verbose {
				log.Printf("[WEBHOOK] Successfully notified receipt collection: %s", payload.ReceiptID)
			}
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		if c.verbose {
			log.Printf("[WEBHOOK] Bad status %d for receipt %s", resp.StatusCode, payload.ReceiptID)
		}
	}

	// All retries failed
	log.Printf("[WEBHOOK] Failed to notify receipt collection after %d attempts: %s (last error: %v)",
		c.maxRetries+1, payload.ReceiptID, lastErr)

	return lastErr
}
