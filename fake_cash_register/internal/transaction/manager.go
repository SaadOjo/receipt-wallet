package transaction

import (
	"log"
	"sync"
	"time"

	"fake-cash-register/internal/models"
)

// TransactionStatus represents the state of a transaction
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusConfirmed TransactionStatus = "confirmed"
	StatusExpired   TransactionStatus = "expired"
	StatusError     TransactionStatus = "error"
)

// PendingTransaction tracks transactions waiting for wallet confirmation
type PendingTransaction struct {
	ReceiptID    string
	Receipt      *models.Receipt
	Status       TransactionStatus
	SubmittedAt  time.Time
	ConfirmedAt  *time.Time
	ErrorMessage string
}

// Manager handles pending transactions and webhook confirmations
type Manager struct {
	pending map[string]*PendingTransaction
	mutex   sync.RWMutex
	verbose bool
}

// NewManager creates a new transaction manager
func NewManager(verbose bool) *Manager {
	return &Manager{
		pending: make(map[string]*PendingTransaction),
		verbose: verbose,
	}
}

// AddPendingTransaction adds a transaction waiting for confirmation
func (m *Manager) AddPendingTransaction(receiptID string, receipt *models.Receipt) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.pending[receiptID] = &PendingTransaction{
		ReceiptID:   receiptID,
		Receipt:     receipt,
		SubmittedAt: time.Now(),
	}

	if m.verbose {
		log.Printf("[TRANSACTION] Waiting for webhook confirmation: %s", receiptID)
	}
}

// ConfirmTransaction processes webhook confirmation and removes transaction
func (m *Manager) ConfirmTransaction(receiptID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.pending[receiptID]; exists {
		// Remove transaction immediately after confirmation - no need to track
		delete(m.pending, receiptID)

		if m.verbose {
			log.Printf("[TRANSACTION] Transaction confirmed and completed: %s", receiptID)
		}
		return true
	}

	if m.verbose {
		log.Printf("[TRANSACTION] Unknown transaction for confirmation: %s", receiptID)
	}
	return false
}

// CleanupExpiredTransactions removes transactions that timed out (after 5 minutes)
func (m *Manager) CleanupExpiredTransactions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)

	for receiptID, tx := range m.pending {
		if tx.SubmittedAt.Before(cutoff) {
			delete(m.pending, receiptID)
			if m.verbose {
				log.Printf("[TRANSACTION] Transaction timed out and removed: %s", receiptID)
			}
		}
	}
}
