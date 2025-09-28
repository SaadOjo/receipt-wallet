package interfaces

// RevenueAuthorityService handles receipt hash signing with binary data
type RevenueAuthorityService interface {
	SignHash(hash []byte) ([]byte, error)
	GetPublicKey() ([]byte, error)
}

// ReceiptBankService handles encrypted receipt submission with privacy-preserving indexing
type ReceiptBankService interface {
	SubmitReceipt(userEphemeralKeyCompressed []byte, encryptedData []byte) error
	SetWebhookHandler(handler WebhookHandler)
}

// CryptoService handles cryptographic operations with binary data (privacy-preserving)
// Key validation is handled internally by the encryption method
type CryptoService interface {
	GenerateReceiptHash(binaryReceipt []byte) []byte
	EncryptWithUserEphemeralKey(binaryData []byte, userEphemeralKeyCompressed []byte) ([]byte, error)
}

// NOTE: ReceiptGenerationService has been replaced by the CashRegister class
// which provides better encapsulation and state management.

// NOTE: ReceiptIssueService has been eliminated - receipt issuing is now handled
// directly by CashRegister.IssueCurrentReceipt() for better encapsulation.

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

// NOTE: ServiceContainer has been eliminated - services are now injected directly
// into CashRegister for better encapsulation and cleaner architecture.
