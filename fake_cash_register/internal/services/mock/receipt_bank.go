package mock

import (
	"fake-cash-register/internal/interfaces"
	"log"
	"time"
)

type MockReceiptBank struct {
	verbose        bool
	webhookHandler interfaces.WebhookHandler
}

func NewMockReceiptBank(verbose bool) *MockReceiptBank {
	return &MockReceiptBank{
		verbose: verbose,
	}
}

func (m *MockReceiptBank) SubmitReceipt(ephemeralKey, encryptedData string) error {
	if m.verbose {
		log.Printf("[MOCK] Receipt Bank: Submitting receipt")
		log.Printf("[MOCK] Ephemeral Key: %s...", ephemeralKey[:16])
		log.Printf("[MOCK] Encrypted Data: %d bytes", len(encryptedData))
	}
	
	// Simulate network delay
	time.Sleep(200 * time.Millisecond)
	
	if m.verbose {
		log.Printf("[MOCK] Receipt Bank: Receipt submitted successfully")
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