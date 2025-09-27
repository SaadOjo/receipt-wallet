package models

import (
	"fmt"
	"time"
)

type Receipt struct {
	ZReportNumber  string       `json:"z_report_number"`
	TransactionID  string       `json:"transaction_id"`
	Timestamp      time.Time    `json:"timestamp"`
	StoreVKN       string       `json:"store_vkn"`
	StoreName      string       `json:"store_name"`
	StoreAddress   string       `json:"store_address"`
	Items          []Item       `json:"items"`
	TaxBreakdown   TaxBreakdown `json:"tax_breakdown"`
	TotalAmount    float64      `json:"total_amount"`
	PaymentMethod  string       `json:"payment_method"`
	ReceiptSerial  string       `json:"receipt_serial"`
}

type Item struct {
	KisimName   string  `json:"kisim_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	TaxRate     int     `json:"tax_rate"`
	Description string  `json:"description,omitempty"`
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

// Transaction represents the current transaction state
type Transaction struct {
	Items         []TransactionItem `json:"items"`
	PaymentMethod string           `json:"payment_method"`
	Status        string           `json:"status"` // "building", "payment", "qr_scan", "processing", "complete"
}

type TransactionItem struct {
	KisimID     int     `json:"kisim_id"`
	KisimName   string  `json:"kisim_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	TaxRate     int     `json:"tax_rate"`
	Description string  `json:"description,omitempty"`
}

// API Response models
type TransactionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Receipt *Receipt `json:"receipt,omitempty"`
}

type KisimResponse struct {
	Kisim []KisimInfo `json:"kisim"`
}

type KisimInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	TaxRate     int    `json:"tax_rate"`
	Description string `json:"description"`
}

// Helper methods
func (r *Receipt) CalculateTotals() {
	var total float64
	var tax10Base, tax20Base float64
	
	for _, item := range r.Items {
		total += item.TotalPrice
		
		baseAmount := item.TotalPrice / (1 + float64(item.TaxRate)/100)
		if item.TaxRate == 10 {
			tax10Base += baseAmount
		} else if item.TaxRate == 20 {
			tax20Base += baseAmount
		}
	}
	
	r.TaxBreakdown.Tax10Percent = TaxDetail{
		TaxableAmount: tax10Base,
		TaxAmount:     tax10Base * 0.10,
	}
	
	r.TaxBreakdown.Tax20Percent = TaxDetail{
		TaxableAmount: tax20Base,
		TaxAmount:     tax20Base * 0.20,
	}
	
	r.TaxBreakdown.TotalTax = r.TaxBreakdown.Tax10Percent.TaxAmount + r.TaxBreakdown.Tax20Percent.TaxAmount
	r.TotalAmount = total
}

func (r *Receipt) FormatForDisplay() string {
	layout := `
========================================
         %s
========================================
VKN: %s
%s
========================================
Tarih: %s
İşlem No: %s
Fiş No: %s
========================================

`
	
	header := fmt.Sprintf(layout, 
		r.StoreName,
		r.StoreVKN,
		r.StoreAddress,
		r.Timestamp.Format("02.01.2006 15:04"),
		r.TransactionID,
		r.ReceiptSerial,
	)
	
	items := ""
	for _, item := range r.Items {
		items += fmt.Sprintf("%-20s %dx%.2f ₺%.2f\n", 
			item.KisimName, 
			item.Quantity, 
			item.UnitPrice, 
			item.TotalPrice,
		)
	}
	
	footer := fmt.Sprintf(`
----------------------------------------
KDV %%10: ₺%.2f
KDV %%20: ₺%.2f
Toplam KDV: ₺%.2f

GENEL TOPLAM: ₺%.2f
Ödeme: %s
========================================
Z Rapor No: %s
========================================
`, 
		r.TaxBreakdown.Tax10Percent.TaxAmount,
		r.TaxBreakdown.Tax20Percent.TaxAmount,
		r.TaxBreakdown.TotalTax,
		r.TotalAmount,
		r.PaymentMethod,
		r.ZReportNumber,
	)
	
	return header + items + footer
}