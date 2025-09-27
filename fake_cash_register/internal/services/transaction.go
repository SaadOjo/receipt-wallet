package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
)

type TransactionService struct {
	revenueAuthority interfaces.RevenueAuthorityService
	receiptBank     interfaces.ReceiptBankService
	crypto          interfaces.CryptoService
	verbose         bool
	zReportCounter  int
	receiptCounter  int
}

func NewTransactionService(
	revenueAuthority interfaces.RevenueAuthorityService,
	receiptBank interfaces.ReceiptBankService,
	crypto interfaces.CryptoService,
	verbose bool,
) *TransactionService {
	return &TransactionService{
		revenueAuthority: revenueAuthority,
		receiptBank:     receiptBank,
		crypto:          crypto,
		verbose:         verbose,
		zReportCounter:  1,
		receiptCounter:  1,
	}
}

func (t *TransactionService) StartTransaction() *models.Transaction {
	if t.verbose {
		log.Printf("[TRANSACTION] Starting new transaction")
	}
	
	return &models.Transaction{
		Items:  make([]models.TransactionItem, 0),
		Status: "building",
	}
}

func (t *TransactionService) AddItem(tx *models.Transaction, kisimID int, kisimName string, unitPrice float64, taxRate int, description string) error {
	if t.verbose {
		log.Printf("[TRANSACTION] Adding item: %s (₺%.2f)", kisimName, unitPrice)
	}
	
	// Add new item (always add new items for cash register)
	newItem := models.TransactionItem{
		KisimID:     kisimID,
		KisimName:   kisimName,
		UnitPrice:   unitPrice,
		Quantity:    1,
		TaxRate:     taxRate,
		Description: description,
	}
	
	tx.Items = append(tx.Items, newItem)
	return nil
}

func (t *TransactionService) SetQuantity(tx *models.Transaction, itemIndex, quantity int) error {
	if itemIndex < 0 || itemIndex >= len(tx.Items) {
		return fmt.Errorf("invalid item index: %d", itemIndex)
	}
	
	if quantity <= 0 {
		// Remove item
		tx.Items = append(tx.Items[:itemIndex], tx.Items[itemIndex+1:]...)
		if t.verbose {
			log.Printf("[TRANSACTION] Removed item at index %d", itemIndex)
		}
	} else {
		tx.Items[itemIndex].Quantity = quantity
		if t.verbose {
			log.Printf("[TRANSACTION] Set quantity for %s to %d", 
				tx.Items[itemIndex].KisimName, quantity)
		}
	}
	
	return nil
}

func (t *TransactionService) SetPaymentMethod(tx *models.Transaction, method string) error {
	if t.verbose {
		log.Printf("[TRANSACTION] Payment method set to: %s", method)
	}
	
	tx.PaymentMethod = method
	tx.Status = "payment"
	return nil
}

func (t *TransactionService) GenerateReceipt(tx *models.Transaction, storeInfo interfaces.StoreInfo) (*models.Receipt, error) {
	if t.verbose {
		log.Printf("[TRANSACTION] Generating receipt for transaction with %d items", len(tx.Items))
	}
	
	if len(tx.Items) == 0 {
		return nil, fmt.Errorf("cannot generate receipt for empty transaction")
	}
	
	receipt := &models.Receipt{
		ZReportNumber: fmt.Sprintf("Z%04d", t.zReportCounter),
		TransactionID: fmt.Sprintf("TX%s%04d", time.Now().Format("20060102"), t.receiptCounter),
		Timestamp:     time.Now(),
		StoreVKN:      storeInfo.VKN,
		StoreName:     storeInfo.Name,
		StoreAddress:  storeInfo.Address,
		PaymentMethod: tx.PaymentMethod,
		ReceiptSerial: fmt.Sprintf("F%04d", t.receiptCounter),
		Items:         make([]models.Item, 0),
	}
	
	// Convert transaction items to receipt items
	for _, txItem := range tx.Items {
		receiptItem := models.Item{
			KisimName:   txItem.KisimName,
			Quantity:    txItem.Quantity,
			UnitPrice:   txItem.UnitPrice,
			TotalPrice:  float64(txItem.Quantity) * txItem.UnitPrice,
			TaxRate:     txItem.TaxRate,
			Description: txItem.Description,
		}
		receipt.Items = append(receipt.Items, receiptItem)
	}
	
	// Calculate totals
	receipt.CalculateTotals()
	
	t.receiptCounter++
	
	if t.verbose {
		log.Printf("[TRANSACTION] Generated receipt %s with total ₺%.2f", 
			receipt.TransactionID, receipt.TotalAmount)
	}
	
	return receipt, nil
}

func (t *TransactionService) ProcessTransaction(receipt *models.Receipt, ephemeralKey string) error {
	if t.verbose {
		log.Printf("[TRANSACTION] Processing transaction %s", receipt.TransactionID)
	}
	
	// Step 1: Generate hash
	hash, err := t.crypto.GenerateReceiptHash(receipt)
	if err != nil {
		return fmt.Errorf("failed to generate receipt hash: %v", err)
	}
	
	if t.verbose {
		log.Printf("[TRANSACTION] Generated receipt hash: %s", hash[:16]+"...")
	}
	
	// Step 2: Get signature from revenue authority
	signature, err := t.revenueAuthority.SignHash(hash)
	if err != nil {
		return fmt.Errorf("failed to get signature from revenue authority: %v", err)
	}
	
	if t.verbose {
		log.Printf("[TRANSACTION] Received signature from revenue authority")
	}
	
	// Step 3: Prepare combined data (receipt + signature)
	combinedData := struct {
		Receipt   *models.Receipt `json:"receipt"`
		Signature string          `json:"signature"`
		Hash      string          `json:"hash"`
	}{
		Receipt:   receipt,
		Signature: signature,
		Hash:      hash,
	}
	
	// Marshal to JSON
	dataJSON, err := json.Marshal(combinedData)
	if err != nil {
		return fmt.Errorf("failed to marshal combined data: %v", err)
	}
	
	// Step 4: Encrypt with ephemeral key
	encryptedData, err := t.crypto.EncryptWithEphemeralKey(dataJSON, ephemeralKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt receipt data: %v", err)
	}
	
	if t.verbose {
		log.Printf("[TRANSACTION] Encrypted receipt data")
	}
	
	// Step 5: Submit to receipt bank
	err = t.receiptBank.SubmitReceipt(ephemeralKey, encryptedData)
	if err != nil {
		return fmt.Errorf("failed to submit to receipt bank: %v", err)
	}
	
	if t.verbose {
		log.Printf("[TRANSACTION] Successfully submitted to receipt bank")
	}
	
	return nil
}