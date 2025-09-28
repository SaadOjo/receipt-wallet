package api

// Common API types and enums

// WebhookStatus represents the status of a webhook event
type WebhookStatus string

const (
	WebhookStatusDownloaded WebhookStatus = "downloaded"
	WebhookStatusExpired    WebhookStatus = "expired"
	WebhookStatusError      WebhookStatus = "error"
)

// APIError represents RESTful error response structure
type APIError struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Common error codes
const (
	ErrorCodeInvalidRequest   = "INVALID_REQUEST"
	ErrorCodeInvalidKey       = "INVALID_KEY"
	ErrorCodeNoActiveReceipt  = "NO_ACTIVE_RECEIPT"
	ErrorCodeReceiptNotFound  = "RECEIPT_NOT_FOUND"
	ErrorCodeInternalError    = "INTERNAL_ERROR"
	ErrorCodeValidationFailed = "VALIDATION_FAILED"
)
