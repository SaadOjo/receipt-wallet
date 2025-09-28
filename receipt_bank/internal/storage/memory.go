package storage

import (
	"fmt"
	"log"
	"sync"
	"time"

	"receipt-bank/internal/models"
)

// MemoryStorage provides thread-safe in-memory storage for receipts
type MemoryStorage struct {
	mu            sync.RWMutex
	receipts      map[string]*models.Receipt // key: ephemeral_key
	maxReceiptAge time.Duration
	verbose       bool
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage(maxReceiptAge time.Duration, verbose bool) *MemoryStorage {
	return &MemoryStorage{
		receipts:      make(map[string]*models.Receipt),
		maxReceiptAge: maxReceiptAge,
		verbose:       verbose,
	}
}

// Store stores a receipt indexed by ephemeral key
func (ms *MemoryStorage) Store(receipt *models.Receipt) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check for duplicate receipt ID
	for _, existingReceipt := range ms.receipts {
		if existingReceipt.ReceiptID == receipt.ReceiptID {
			return fmt.Errorf("receipt_id already exists")
		}
	}

	ms.receipts[receipt.EphemeralKey] = receipt

	if ms.verbose {
		log.Printf("[STORAGE] Stored receipt %s (ephemeral key: %s)",
			receipt.ReceiptID, receipt.EphemeralKey)
	}

	return nil
}

// Retrieve retrieves and deletes a receipt by ephemeral key
func (ms *MemoryStorage) Retrieve(ephemeralKey string) (*models.Receipt, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	receipt, exists := ms.receipts[ephemeralKey]
	if !exists {
		if ms.verbose {
			log.Printf("[STORAGE] Receipt not found for ephemeral key: %s", ephemeralKey)
			log.Printf("[STORAGE] Available keys: %d", len(ms.receipts))
			for key := range ms.receipts {
				log.Printf("[STORAGE]   Available key: %s", key)
			}
		}
		return nil, fmt.Errorf("receipt not found")
	}

	// Delete the receipt after retrieval (one-time collection)
	delete(ms.receipts, ephemeralKey)

	if ms.verbose {
		log.Printf("[STORAGE] Retrieved and deleted receipt %s (ephemeral key: %s)",
			receipt.ReceiptID, ephemeralKey)
	}

	return receipt, nil
}

// Cleanup removes expired receipts
func (ms *MemoryStorage) Cleanup() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	removed := 0

	for ephemeralKey, receipt := range ms.receipts {
		if now.Sub(receipt.Timestamp) > ms.maxReceiptAge {
			delete(ms.receipts, ephemeralKey)
			removed++

			if ms.verbose {
				log.Printf("[STORAGE] Cleaned up expired receipt %s (age: %v)",
					receipt.ReceiptID, now.Sub(receipt.Timestamp))
			}
		}
	}

	if ms.verbose && removed > 0 {
		log.Printf("[STORAGE] Cleanup completed: removed %d expired receipts", removed)
	}
}

// StartCleanupRoutine starts a background routine to clean up expired receipts
func (ms *MemoryStorage) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			ms.Cleanup()
		}
	}()

	if ms.verbose {
		log.Printf("[STORAGE] Started cleanup routine (interval: %v)", interval)
	}
}

// Stats returns storage statistics
func (ms *MemoryStorage) Stats() (int, int) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	now := time.Now()
	total := len(ms.receipts)
	expired := 0

	for _, receipt := range ms.receipts {
		if now.Sub(receipt.Timestamp) > ms.maxReceiptAge {
			expired++
		}
	}

	return total, expired
}
