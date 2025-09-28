package services

import (
	"fmt"
	"log"
	"time"

	"fake-cash-register/internal/binary"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
)

type TransactionService struct {
	revenueAuthority interfaces.RevenueAuthorityService
	receiptBank      interfaces.ReceiptBankService
	crypto           interfaces.CryptoService
	kisimLookup      models.KisimLookup
	verbose          bool
	zReportCounter   int
	receiptCounter   int
}

func NewTransactionService(
	revenueAuthority interfaces.RevenueAuthorityService,
	receiptBank interfaces.ReceiptBankService,
	crypto interfaces.CryptoService,
	kisimLookup models.KisimLookup,
	verbose bool,
) *TransactionService {
	return &TransactionService{
		revenueAuthority: revenueAuthority,
		receiptBank:      receiptBank,
		crypto:           crypto,
		kisimLookup:      kisimLookup,
		verbose:          verbose,
		zReportCounter:   1,
		receiptCounter:   1,
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

func (t *TransactionService) AddItem(tx *models.Transaction, kisimID int, quantity int) error {
	// Look up KISIM information
	kisimInfo, exists := t.kisimLookup.GetKisimInfo(kisimID)
	if !exists {
		return fmt.Errorf("unknown KISIM ID: %d", kisimID)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Adding item: %s (₺%.2f) x%d", kisimInfo.Name, kisimInfo.PresetPrice, quantity)
	}

	// Check if this kisim already exists in the transaction (same ID and price)
	for i, item := range tx.Items {
		if item.KisimID == kisimID && item.UnitPrice == kisimInfo.PresetPrice {
			// Increment quantity of existing item
			tx.Items[i].Quantity += quantity
			if t.verbose {
				log.Printf("[TRANSACTION] Incremented %s quantity to %d", kisimInfo.Name, tx.Items[i].Quantity)
			}
			return nil
		}
	}

	// Add new item if not found
	newItem := models.TransactionItem{
		KisimID:   kisimID,
		UnitPrice: kisimInfo.PresetPrice,
		Quantity:  quantity,
		TaxRate:   kisimInfo.TaxRate,
	}

	tx.Items = append(tx.Items, newItem)
	if t.verbose {
		log.Printf("[TRANSACTION] Added new item: %s x%d", kisimInfo.Name, quantity)
	}
	return nil
}


func (t *TransactionService) UpdateItemQuantity(tx *models.Transaction, kisimID int, quantity int) error {
	if quantity <= 0 {
		// Individual item deletion not allowed per specification
		return fmt.Errorf("individual item deletion not allowed - use cancel transaction instead")
	}

	// Find the item by KisimID
	for i, item := range tx.Items {
		if item.KisimID == kisimID {
			tx.Items[i].Quantity = quantity
			if t.verbose {
				kisimName := t.kisimLookup.GetKisimName(kisimID)
				log.Printf("[TRANSACTION] Updated quantity for %s to %d", kisimName, quantity)
			}
			return nil
		}
	}

	return fmt.Errorf("item with KISIM ID %d not found in transaction", kisimID)
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
			KisimID:    txItem.KisimID,
			Quantity:   txItem.Quantity,
			UnitPrice:  txItem.UnitPrice,
			TotalPrice: float64(txItem.Quantity) * txItem.UnitPrice,
			TaxRate:    txItem.TaxRate,
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

func (t *TransactionService) ProcessTransaction(receipt *models.Receipt, ephemeralKeyPEMBase64 string) error {
	if t.verbose {
		log.Printf("[TRANSACTION] Processing transaction %s", receipt.TransactionID)
	}

	// Step 1: Serialize receipt to binary format
	binaryReceipt, err := binary.SerializeReceipt(receipt)
	if err != nil {
		return fmt.Errorf("failed to serialize receipt: %v", err)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Serialized receipt to %d bytes", len(binaryReceipt))
	}

	// Step 2: Generate hash of binary receipt
	binaryHash := t.crypto.GenerateReceiptHash(binaryReceipt)

	if t.verbose {
		hashBase64 := binary.ToBase64(binaryHash)
		log.Printf("[TRANSACTION] Generated receipt hash: %s", hashBase64[:16]+"...")
	}

	// Step 3: Get signature from revenue authority
	binarySignature, err := t.revenueAuthority.SignHash(binaryHash)
	if err != nil {
		return fmt.Errorf("failed to get signature from revenue authority: %v", err)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Received signature from revenue authority")
	}

	// Step 4: Create signed receipt (binary receipt + signature)
	binarySignedReceipt, err := binary.CreateSignedReceipt(binaryReceipt, binarySignature)
	if err != nil {
		return fmt.Errorf("failed to create signed receipt: %v", err)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Created signed receipt: %d bytes", len(binarySignedReceipt))
	}

	// Step 5: Encrypt signed receipt with ephemeral key
	binaryEncrypted, err := t.crypto.EncryptWithEphemeralKey(binarySignedReceipt, ephemeralKeyPEMBase64)
	if err != nil {
		return fmt.Errorf("failed to encrypt receipt data: %v", err)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Encrypted receipt data")
	}

	// Step 6: Submit to receipt bank (encode to base64 for transmission)
	encryptedBase64 := binary.ToBase64(binaryEncrypted)
	err = t.receiptBank.SubmitReceipt(ephemeralKeyPEMBase64, encryptedBase64)
	if err != nil {
		return fmt.Errorf("failed to submit to receipt bank: %v", err)
	}

	if t.verbose {
		log.Printf("[TRANSACTION] Successfully submitted to receipt bank")
	}

	return nil
}
