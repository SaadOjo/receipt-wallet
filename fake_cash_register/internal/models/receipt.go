package models

import (
	"time"
)

type Receipt struct {
	ZReportNumber string       `json:"z_report_number"`
	TransactionID string       `json:"transaction_id"`
	Timestamp     time.Time    `json:"timestamp"`
	StoreVKN      string       `json:"store_vkn"`
	StoreName     string       `json:"store_name"`
	StoreAddress  string       `json:"store_address"`
	Items         []Item       `json:"items"`
	TaxBreakdown  TaxBreakdown `json:"tax_breakdown"`
	TotalAmount   float64      `json:"total_amount"`
	PaymentMethod string       `json:"payment_method"`
	ReceiptSerial string       `json:"receipt_serial"`
}

type Item struct {
	KisimID    int     `json:"kisim_id"`
	KisimName  string  `json:"kisim_name"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	TaxRate    int     `json:"tax_rate"`
}

type TaxBreakdown struct {
	Tax10Percent TaxDetail `json:"tax_10_percent"`
	Tax20Percent TaxDetail `json:"tax_20_percent"`
	TotalTax     float64   `json:"total_tax"`
}

type TaxDetail struct {
	TaxableAmount float64 `json:"taxable_amount"`
	TaxAmount     float64 `json:"tax_amount"`
}

// NOTE: ProcessTransactionResponse removed - RESTful APIs return Receipt directly
// (renamed from /process to /issue_receipt for clarity)
// with appropriate HTTP status codes (200 for success, 400/500 for errors)

type KisimResponse struct {
	Kisim []KisimInfo `json:"kisim"`
}

type KisimInfo struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	TaxRate     int     `json:"tax_rate"`
	PresetPrice float64 `json:"preset_price"`
}

// KisimLookup provides KISIM information lookup
type KisimLookup map[int]KisimInfo

// GetKisimInfo returns KISIM information by ID
func (kl KisimLookup) GetKisimInfo(kisimID int) (KisimInfo, bool) {
	kisim, exists := kl[kisimID]
	return kisim, exists
}
