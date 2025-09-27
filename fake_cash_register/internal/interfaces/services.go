package interfaces

import "fake-cash-register/internal/models"

// RevenueAuthorityService handles receipt hash signing
type RevenueAuthorityService interface {
	SignHash(hash string) (string, error)
	GetPublicKey() (string, error)
}

// ReceiptBankService handles encrypted receipt submission
type ReceiptBankService interface {
	SubmitReceipt(ephemeralKey, encryptedData string) error
	SetWebhookHandler(handler WebhookHandler)
}

// QRScannerService handles ephemeral key input
type QRScannerService interface {
	GetEphemeralKey() (string, error)
	ValidateKey(key string) error
}

// CryptoService handles cryptographic operations
type CryptoService interface {
	GenerateReceiptHash(receipt *models.Receipt) (string, error)
	EncryptWithEphemeralKey(data []byte, ephemeralKeyPEM string) (string, error)
	ValidateEphemeralKey(keyPEM string) error
}

// TransactionService handles transaction workflow
type TransactionService interface {
	StartTransaction() *models.Transaction
	AddItem(tx *models.Transaction, kisimID int, kisimName string, unitPrice float64, taxRate int, description string) error
	SetQuantity(tx *models.Transaction, itemIndex, quantity int) error
	SetPaymentMethod(tx *models.Transaction, method string) error
	GenerateReceipt(tx *models.Transaction, storeInfo StoreInfo) (*models.Receipt, error)
	ProcessTransaction(receipt *models.Receipt, ephemeralKey string) error
}

// WebhookHandler handles receipt bank confirmations
type WebhookHandler interface {
	HandleDownloadConfirmation(receiptID string) error
}

// StoreInfo contains store configuration data
type StoreInfo struct {
	VKN     string
	Name    string
	Address string
}

// ServiceContainer holds all service implementations
type ServiceContainer struct {
	RevenueAuthority RevenueAuthorityService
	ReceiptBank     ReceiptBankService
	QRScanner       QRScannerService
	Crypto          CryptoService
	Transaction     TransactionService
}

// Revenue Authority API models
type SignRequest struct {
	Hash string `json:"hash"`
}

type SignResponse struct {
	Signature string `json:"signature"`
}

type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Receipt Bank API models
type ReceiptSubmission struct {
	EphemeralKey  string `json:"ephemeral_key"`
	EncryptedData string `json:"encrypted_data"`
	ReceiptID     string `json:"receipt_id"`
}

type BankResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ReceiptID string `json:"receipt_id,omitempty"`
}

// Webhook payload
type WebhookPayload struct {
	ReceiptID string `json:"receipt_id"`
	Status    string `json:"status"` // "downloaded", "expired", "error"
	Timestamp string `json:"timestamp"`
}