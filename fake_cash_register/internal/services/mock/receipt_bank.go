package mock

import (
	"encoding/base64"
	"log"
	"time"

	"fake-cash-register/internal/interfaces"
)

type MockReceiptBank struct {
	verbose        bool
	webhookHandler interfaces.WebhookHandler
	storage        map[string]string // ephemeral key -> encrypted receipt storage
}

func NewMockReceiptBank(verbose bool) *MockReceiptBank {
	return &MockReceiptBank{
		verbose: verbose,
		storage: make(map[string]string),
	}
}

func (m *MockReceiptBank) SubmitReceipt(userEphemeralKeyCompressed []byte, encryptedData []byte) error {
	// Convert compressed key to base64 for internal indexing
	keyBase64 := base64.StdEncoding.EncodeToString(userEphemeralKeyCompressed)
	// Convert encrypted data to base64 for internal storage
	encryptedDataBase64 := base64.StdEncoding.EncodeToString(encryptedData)

	if m.verbose {
		log.Printf("[MOCK] Receipt Bank: Submitting receipt (privacy-preserving)")
		log.Printf("[MOCK] User Ephemeral Key: %s... (%d bytes compressed)", keyBase64[:16], len(userEphemeralKeyCompressed))
		log.Printf("[MOCK] Encrypted Data: %d bytes", len(encryptedData))
	}

	// Store encrypted receipt indexed by user's ephemeral key (privacy-preserving)
	m.storage[keyBase64] = encryptedDataBase64

	// Simulate network delay
	time.Sleep(200 * time.Millisecond)

	if m.verbose {
		log.Printf("[MOCK] Receipt Bank: Receipt submitted successfully (user anonymous)")
		log.Printf("[MOCK] Storage contains %d receipts", len(m.storage))
	}

	// Simulate webhook callback after a short delay
	if m.webhookHandler != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			receiptID := generateMockReceiptID()
			if m.verbose {
				log.Printf("[MOCK] Receipt Bank: Sending webhook confirmation for %s", receiptID)
			}
			m.webhookHandler.HandleDownloadConfirmation(receiptID)
		}()
	}

	return nil
}

func (m *MockReceiptBank) SetWebhookHandler(handler interfaces.WebhookHandler) {
	m.webhookHandler = handler
	if m.verbose {
		log.Printf("[MOCK] Receipt Bank: Webhook handler registered")
	}
}

func generateMockReceiptID() string {
	return "mock_receipt_" + time.Now().Format("20060102150405")
}
