package api

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
	WebhookURL    string `json:"webhook_url"`
}

type ReceiptBankResponse struct {
	ReceiptID string `json:"receipt_id"`
}

// Webhook payload
type WebhookPayload struct {
	ReceiptID string `json:"receipt_id"`
	Status    string `json:"status"` // "downloaded", "expired", "error"
	Timestamp string `json:"timestamp"`
}
