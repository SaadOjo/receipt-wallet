package cashregister

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"fake-cash-register/internal/binary"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
)

// CashRegister represents a cash register that manages complete receipt lifecycle
type CashRegister struct {
	// Core business data
	storeInfo   interfaces.StoreInfo
	kisimLookup models.KisimLookup
	verbose     bool

	// Service dependencies for complete receipt lifecycle
	revenueAuthority interfaces.RevenueAuthorityService
	receiptBank      interfaces.ReceiptBankService
	cryptoService    interfaces.CryptoService

	// Internal state management
	currentReceipt *models.Receipt
	zReportCounter int
	receiptCounter int
}

// NewCashRegister creates a new cash register with complete receipt lifecycle capabilities
func NewCashRegister(
	storeInfo interfaces.StoreInfo,
	kisimLookup models.KisimLookup,
	revenueAuthority interfaces.RevenueAuthorityService,
	receiptBank interfaces.ReceiptBankService,
	cryptoService interfaces.CryptoService,
	verbose bool,
) *CashRegister {
	return &CashRegister{
		storeInfo:        storeInfo,
		kisimLookup:      kisimLookup,
		revenueAuthority: revenueAuthority,
		receiptBank:      receiptBank,
		cryptoService:    cryptoService,
		verbose:          verbose,
		zReportCounter:   1,
		receiptCounter:   1,
	}
}

// StartNewReceipt begins a new receipt transaction
func (cr *CashRegister) StartNewReceipt() {
	if cr.verbose {
		log.Printf("[CASH-REGISTER] Starting new receipt")
	}

	cr.currentReceipt = &models.Receipt{
		Items: make([]models.Item, 0),
	}
}

// AddItem adds an item to the current receipt with optional custom unit price
func (cr *CashRegister) AddItem(kisimID int, quantity int, customUnitPrice float64) error {
	if cr.currentReceipt == nil {
		return fmt.Errorf("no active receipt - call StartNewReceipt first")
	}

	// Look up KISIM information
	kisimInfo, exists := cr.kisimLookup.GetKisimInfo(kisimID)
	if !exists {
		return fmt.Errorf("unknown KISIM ID: %d", kisimID)
	}

	// Use custom price if provided, otherwise use preset price
	unitPrice := kisimInfo.PresetPrice
	if customUnitPrice > 0 {
		unitPrice = customUnitPrice
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Adding item: %s (₺%.2f) x%d", kisimInfo.Name, unitPrice, quantity)
	}

	// Check if this kisim already exists in the receipt (same ID and same unit price)
	for i, item := range cr.currentReceipt.Items {
		if item.KisimID == kisimID && item.UnitPrice == unitPrice {
			// Increment quantity of existing item with same price
			cr.currentReceipt.Items[i].Quantity += quantity
			cr.currentReceipt.Items[i].TotalPrice = cr.currentReceipt.Items[i].UnitPrice * float64(cr.currentReceipt.Items[i].Quantity)
			if cr.verbose {
				log.Printf("[CASH-REGISTER] Incremented %s quantity to %d", kisimInfo.Name, cr.currentReceipt.Items[i].Quantity)
			}
			return nil
		}
	}

	// Add new item if not found (different kisim or different price = new line)
	totalPrice := unitPrice * float64(quantity)
	newItem := models.Item{
		KisimID:    kisimID,
		KisimName:  kisimInfo.Name,
		UnitPrice:  unitPrice,
		Quantity:   quantity,
		TotalPrice: totalPrice,
		TaxRate:    kisimInfo.TaxRate,
	}

	cr.currentReceipt.Items = append(cr.currentReceipt.Items, newItem)
	if cr.verbose {
		log.Printf("[CASH-REGISTER] Added new item: %s x%d @ ₺%.2f", kisimInfo.Name, quantity, unitPrice)
	}
	return nil
}

// SetPaymentMethod sets the payment method for the current receipt
func (cr *CashRegister) SetPaymentMethod(method string) error {
	if cr.currentReceipt == nil {
		return fmt.Errorf("no active receipt - call StartNewReceipt first")
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Payment method set to: %s", method)
	}

	cr.currentReceipt.PaymentMethod = method
	return nil
}

// FinalizeCurrentReceipt completes the current receipt and returns it
func (cr *CashRegister) FinalizeCurrentReceipt() (*models.Receipt, error) {
	if cr.currentReceipt == nil {
		return nil, fmt.Errorf("no active receipt - call StartNewReceipt first")
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Finalizing receipt with %d items", len(cr.currentReceipt.Items))
	}

	if len(cr.currentReceipt.Items) == 0 {
		return nil, fmt.Errorf("cannot finalize receipt with no items")
	}

	// Add metadata to the receipt
	cr.currentReceipt.ZReportNumber = fmt.Sprintf("Z%04d", cr.zReportCounter)
	cr.currentReceipt.TransactionID = fmt.Sprintf("TX%s%04d", time.Now().Format("20060102"), cr.receiptCounter)
	cr.currentReceipt.Timestamp = time.Now()
	cr.currentReceipt.StoreVKN = cr.storeInfo.VKN
	cr.currentReceipt.StoreName = cr.storeInfo.Name
	cr.currentReceipt.StoreAddress = cr.storeInfo.Address
	cr.currentReceipt.ReceiptSerial = fmt.Sprintf("F%04d", cr.receiptCounter)

	// Calculate totals
	cr.calculateTotals(cr.currentReceipt)

	cr.receiptCounter++

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Finalized receipt %s with total ₺%.2f",
			cr.currentReceipt.TransactionID, cr.currentReceipt.TotalAmount)
	}

	// Return the finalized receipt and clear current state
	finalizedReceipt := cr.currentReceipt
	cr.currentReceipt = nil

	return finalizedReceipt, nil
}

// CancelCurrentReceipt cancels the current receipt
func (cr *CashRegister) CancelCurrentReceipt() {
	if cr.verbose && cr.currentReceipt != nil {
		log.Printf("[CASH-REGISTER] Canceling current receipt")
	}
	cr.currentReceipt = nil
}

// HasActiveReceipt returns true if there's an active receipt
func (cr *CashRegister) HasActiveReceipt() bool {
	return cr.currentReceipt != nil
}

// GetCurrentReceipt returns the current receipt (for testing/debugging)
func (cr *CashRegister) GetCurrentReceipt() *models.Receipt {
	return cr.currentReceipt
}

// calculateTotals calculates tax breakdown and total amount for a receipt
// This is moved from Receipt.CalculateTotals() to keep Receipt as pure data
func (cr *CashRegister) calculateTotals(receipt *models.Receipt) {
	var total float64
	var tax10Base, tax20Base float64

	for _, item := range receipt.Items {
		total += item.TotalPrice

		baseAmount := item.TotalPrice / (1 + float64(item.TaxRate)/100)
		switch item.TaxRate {
		case 10:
			tax10Base += baseAmount
		case 20:
			tax20Base += baseAmount
		}
	}

	receipt.TaxBreakdown.Tax10Percent = models.TaxDetail{
		TaxableAmount: tax10Base,
		TaxAmount:     tax10Base * 0.10,
	}

	receipt.TaxBreakdown.Tax20Percent = models.TaxDetail{
		TaxableAmount: tax20Base,
		TaxAmount:     tax20Base * 0.20,
	}

	receipt.TaxBreakdown.TotalTax = receipt.TaxBreakdown.Tax10Percent.TaxAmount + receipt.TaxBreakdown.Tax20Percent.TaxAmount
	receipt.TotalAmount = total
}

// IssueCurrentReceipt finalizes and issues the current receipt in one atomic operation
func (cr *CashRegister) IssueCurrentReceipt(userEphemeralKeyCompressed []byte) (*models.Receipt, error) {
	if cr.currentReceipt == nil {
		return nil, fmt.Errorf("no active receipt - call StartNewReceipt first")
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Issuing receipt with %d items", len(cr.currentReceipt.Items))
	}

	if len(cr.currentReceipt.Items) == 0 {
		return nil, fmt.Errorf("cannot issue receipt with no items")
	}

	// Step 1: Finalize receipt with metadata and calculations
	cr.currentReceipt.ZReportNumber = fmt.Sprintf("Z%04d", cr.zReportCounter)
	cr.currentReceipt.TransactionID = fmt.Sprintf("TX%s%04d", time.Now().Format("20060102"), cr.receiptCounter)
	cr.currentReceipt.Timestamp = time.Now()
	cr.currentReceipt.StoreVKN = cr.storeInfo.VKN
	cr.currentReceipt.StoreName = cr.storeInfo.Name
	cr.currentReceipt.StoreAddress = cr.storeInfo.Address
	cr.currentReceipt.ReceiptSerial = fmt.Sprintf("F%04d", cr.receiptCounter)

	// Calculate totals
	cr.calculateTotals(cr.currentReceipt)
	cr.receiptCounter++

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Finalized receipt %s with total ₺%.2f",
			cr.currentReceipt.TransactionID, cr.currentReceipt.TotalAmount)
	}

	// Step 2: Validate receipt
	if err := cr.validateReceipt(cr.currentReceipt); err != nil {
		return nil, fmt.Errorf("receipt validation failed: %v", err)
	}

	// Step 3: Serialize receipt to binary format
	binaryReceipt, err := binary.SerializeReceipt(cr.currentReceipt)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize receipt: %v", err)
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Serialized receipt to %d bytes", len(binaryReceipt))
	}

	// Step 4: Generate hash of binary receipt
	binaryHash := cr.cryptoService.GenerateReceiptHash(binaryReceipt)
	hashBase64 := base64.StdEncoding.EncodeToString(binaryHash)

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Generated receipt hash: %s", hashBase64[:16]+"...")
	}

	// Step 5: Get signature from revenue authority
	binarySignature, err := cr.revenueAuthority.SignHash(binaryHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature from revenue authority: %v", err)
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Received signature from revenue authority")
	}

	// Step 6: Create signed receipt (binary receipt + signature)
	binarySignedReceipt, err := binary.CreateSignedReceipt(binaryReceipt, binarySignature)
	if err != nil {
		return nil, fmt.Errorf("failed to create signed receipt: %v", err)
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Created signed receipt: %d bytes", len(binarySignedReceipt))
	}

	// Step 7: Encrypt signed receipt with user's ephemeral key (privacy-preserving)
	binaryEncrypted, err := cr.cryptoService.EncryptWithUserEphemeralKey(binarySignedReceipt, userEphemeralKeyCompressed)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt receipt data: %v", err)
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Privacy-preserving encryption completed")
	}

	// Step 8: Submit to receipt bank using user's ephemeral key as index
	err = cr.receiptBank.SubmitReceipt(userEphemeralKeyCompressed, binaryEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to receipt bank: %v", err)
	}

	if cr.verbose {
		log.Printf("[CASH-REGISTER] Successfully submitted to receipt bank (user anonymous)")
	}

	// Step 9: Return finalized receipt and clear current state
	finalizedReceipt := cr.currentReceipt
	cr.currentReceipt = nil

	return finalizedReceipt, nil
}

// validateReceipt ensures the receipt is complete and valid before issuing
func (cr *CashRegister) validateReceipt(receipt *models.Receipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt cannot be nil")
	}
	if len(receipt.Items) == 0 {
		return fmt.Errorf("receipt must have at least one item")
	}
	if receipt.PaymentMethod == "" {
		return fmt.Errorf("receipt must have a payment method")
	}
	if receipt.TotalAmount <= 0 {
		return fmt.Errorf("receipt total must be greater than zero")
	}
	return nil
}
