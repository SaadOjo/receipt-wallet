package tests

import (
	"fake-cash-register/internal/crypto"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
	"fake-cash-register/internal/services"
	"fake-cash-register/internal/services/mock"
	"testing"
)

func TestTransactionWorkflow(t *testing.T) {
	// Setup mock services with real crypto service
	// NOTE: Mock services now provide valid data for real crypto operations
	revenueAuth := mock.NewMockRevenueAuthority(true)
	receiptBank := mock.NewMockReceiptBank(true)
	cryptoService := crypto.NewCryptoService(true) // Use real crypto service

	// Create test KISIM lookup
	kisimLookup := models.KisimLookup{
		1: {ID: 1, Name: "Test Kisim", TaxRate: 20, PresetPrice: 10.50},
		2: {ID: 2, Name: "Test Kisim 2", TaxRate: 10, PresetPrice: 15.00},
	}

	// Create transaction service
	txService := services.NewTransactionService(
		revenueAuth,
		receiptBank,
		cryptoService,
		kisimLookup,
		true,
	)

	// Test 1: Start transaction
	tx := txService.StartTransaction()
	if tx == nil {
		t.Fatal("Failed to start transaction")
	}
	if tx.Status != "building" {
		t.Errorf("Expected status 'building', got '%s'", tx.Status)
	}

	// Test 2: Add items
	err := txService.AddItem(tx, 1, 1) // KisimID 1, quantity 1
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}
	if len(tx.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(tx.Items))
	}

	// Test 3: Add another item
	err = txService.AddItem(tx, 2, 1) // KisimID 2, quantity 1
	if err != nil {
		t.Fatalf("Failed to add second item: %v", err)
	}
	if len(tx.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(tx.Items))
	}

	// Test 4: Set payment method
	err = txService.SetPaymentMethod(tx, "Nakit")
	if err != nil {
		t.Fatalf("Failed to set payment method: %v", err)
	}
	if tx.PaymentMethod != "Nakit" {
		t.Errorf("Expected payment method 'Nakit', got '%s'", tx.PaymentMethod)
	}

	// Test 5: Generate receipt
	storeInfo := interfaces.StoreInfo{
		VKN:     "1234567890",
		Name:    "Test Store",
		Address: "Test Address",
	}
	receipt, err := txService.GenerateReceipt(tx, storeInfo)
	if err != nil {
		t.Fatalf("Failed to generate receipt: %v", err)
	}
	if receipt.TotalAmount != 25.5 { // 10.50 + 15.00 = 25.50
		t.Errorf("Expected total 25.50, got %.2f", receipt.TotalAmount)
	}

	// Test 6: Process transaction - get proper ephemeral key from mock QR scanner
	qrScanner := mock.NewMockQRScanner(false)
	ephemeralKeyPEMBase64, err := qrScanner.GetEphemeralKey()
	if err != nil {
		t.Fatalf("Failed to get ephemeral key: %v", err)
	}
	err = txService.ProcessTransaction(receipt, ephemeralKeyPEMBase64)
	if err != nil {
		t.Fatalf("Failed to process transaction: %v", err)
	}

	t.Log("Transaction workflow test completed successfully")
}

func TestReceiptCalculations(t *testing.T) {
	receipt := &models.Receipt{
		Items: []models.Item{
			{KisimID: 1, Quantity: 2, UnitPrice: 10.0, TotalPrice: 20.0, TaxRate: 10},
			{KisimID: 2, Quantity: 1, UnitPrice: 24.0, TotalPrice: 24.0, TaxRate: 20},
		},
	}

	receipt.CalculateTotals()

	// Check total amount
	expectedTotal := 44.0
	if receipt.TotalAmount != expectedTotal {
		t.Errorf("Expected total amount %.2f, got %.2f", expectedTotal, receipt.TotalAmount)
	}

	// Check 10% tax calculation
	// Item 1: 20.0 total with 10% tax means base = 20/1.1 = 18.18, tax = 1.82
	expectedTax10Base := 20.0 / 1.1
	if receipt.TaxBreakdown.Tax10Percent.TaxableAmount < expectedTax10Base-0.01 ||
		receipt.TaxBreakdown.Tax10Percent.TaxableAmount > expectedTax10Base+0.01 {
		t.Errorf("Expected 10%% tax base %.2f, got %.2f", 
			expectedTax10Base, receipt.TaxBreakdown.Tax10Percent.TaxableAmount)
	}

	// Check 20% tax calculation  
	// Item 2: 24.0 total with 20% tax means base = 24/1.2 = 20.0, tax = 4.0
	expectedTax20Base := 24.0 / 1.2
	if receipt.TaxBreakdown.Tax20Percent.TaxableAmount < expectedTax20Base-0.01 ||
		receipt.TaxBreakdown.Tax20Percent.TaxableAmount > expectedTax20Base+0.01 {
		t.Errorf("Expected 20%% tax base %.2f, got %.2f",
			expectedTax20Base, receipt.TaxBreakdown.Tax20Percent.TaxableAmount)
	}

	t.Log("Receipt calculation test completed successfully")
}

func TestMockServices(t *testing.T) {
	// Test Mock Revenue Authority
	revenueAuth := mock.NewMockRevenueAuthority(false)
	
	// Use a proper 32-byte hash (SHA-256)
	binaryTestHash := make([]byte, 32)
	for i := range binaryTestHash {
		binaryTestHash[i] = byte(i) // Fill with test data
	}
	binarySignature, err := revenueAuth.SignHash(binaryTestHash)
	if err != nil {
		t.Fatalf("Mock revenue authority sign failed: %v", err)
	}
	if len(binarySignature) != 64 {
		t.Errorf("Expected 64-byte signature, got %d bytes", len(binarySignature))
	}

	publicKeyPEMBase64, err := revenueAuth.GetPublicKey()
	if err != nil {
		t.Fatalf("Mock revenue authority get public key failed: %v", err)
	}
	if publicKeyPEMBase64 == "" {
		t.Error("Expected non-empty public key")
	}

	// Test Mock Receipt Bank
	receiptBank := mock.NewMockReceiptBank(false)
	
	err = receiptBank.SubmitReceipt("mock_key", "mock_encrypted_data")
	if err != nil {
		t.Fatalf("Mock receipt bank submit failed: %v", err)
	}

	// Test Mock QR Scanner
	qrScanner := mock.NewMockQRScanner(false)
	
	ephemeralKeyPEMBase64, err := qrScanner.GetEphemeralKey()
	if err != nil {
		t.Fatalf("Mock QR scanner failed: %v", err)
	}
	if ephemeralKeyPEMBase64 == "" {
		t.Error("Expected non-empty ephemeral key")
	}

	err = qrScanner.ValidateKey(ephemeralKeyPEMBase64)
	if err != nil {
		t.Fatalf("Mock QR scanner validation failed: %v", err)
	}

	t.Log("Mock services test completed successfully")
}

func TestSpecificationCompliantWorkflow(t *testing.T) {
	// Setup services with real crypto service
	// NOTE: Mock services now provide valid data for real crypto operations
	revenueAuth := mock.NewMockRevenueAuthority(false)
	receiptBank := mock.NewMockReceiptBank(false)
	cryptoService := crypto.NewCryptoService(false) // Use real crypto service
	
	// Create test KISIM lookup
	kisimLookup := models.KisimLookup{
		1: {ID: 1, Name: "Temel Gıda", TaxRate: 10, PresetPrice: 5.50},
		2: {ID: 2, Name: "Yemek", TaxRate: 20, PresetPrice: 12.75},
		3: {ID: 3, Name: "Custom Item", TaxRate: 10, PresetPrice: 8.25},
	}
	
	txService := services.NewTransactionService(
		revenueAuth,
		receiptBank,
		cryptoService,
		kisimLookup,
		true,
	)
	
	// Test 1: Standard Transaction Flow - Basic KISIM presses
	tx := txService.StartTransaction()
	if tx.Status != "building" {
		t.Errorf("Expected status 'building', got '%s'", tx.Status)
	}
	
	// Add first Temel Gıda item (should create new item)
	err := txService.AddItem(tx, 1, 1) // KisimID 1, quantity 1
	if err != nil {
		t.Fatalf("Failed to add first item: %v", err)
	}
	if len(tx.Items) != 1 || tx.Items[0].Quantity != 1 {
		t.Errorf("Expected 1 item with quantity 1, got %d items with quantity %d", 
			len(tx.Items), tx.Items[0].Quantity)
	}
	
	// Press same KISIM button again (should increment quantity)
	err = txService.AddItem(tx, 1, 1) // KisimID 1, quantity 1 (will be added to existing)
	if err != nil {
		t.Fatalf("Failed to add second instance: %v", err)
	}
	if len(tx.Items) != 1 || tx.Items[0].Quantity != 2 {
		t.Errorf("Expected 1 item with quantity 2, got %d items with quantity %d", 
			len(tx.Items), tx.Items[0].Quantity)
	}
	
	// Press same KISIM button third time (should increment to 3)
	err = txService.AddItem(tx, 1, 1) // KisimID 1, quantity 1 (will be added to existing)
	if err != nil {
		t.Fatalf("Failed to add third instance: %v", err)
	}
	if len(tx.Items) != 1 || tx.Items[0].Quantity != 3 {
		t.Errorf("Expected 1 item with quantity 3, got %d items with quantity %d", 
			len(tx.Items), tx.Items[0].Quantity)
	}
	
	// Add different KISIM (should create second item)
	err = txService.AddItem(tx, 2, 1) // KisimID 2, quantity 1
	if err != nil {
		t.Fatalf("Failed to add different KISIM: %v", err)
	}
	if len(tx.Items) != 2 {
		t.Errorf("Expected 2 different items, got %d", len(tx.Items))
	}
	
	// Test 2: Custom quantity (MIKTAR functionality)
	err = txService.AddItem(tx, 3, 5) // KisimID 3, quantity 5
	if err != nil {
		t.Fatalf("Failed to add item with custom quantity: %v", err)
	}
	if len(tx.Items) != 3 || tx.Items[2].Quantity != 5 {
		t.Errorf("Expected 3 items with last having quantity 5, got %d items", len(tx.Items))
	}
	
	// Test 4: Payment and immediate processing
	err = txService.SetPaymentMethod(tx, "Nakit")
	if err != nil {
		t.Fatalf("Failed to set payment: %v", err)
	}
	
	// Generate receipt
	storeInfo := interfaces.StoreInfo{
		VKN:     "1234567890",
		Name:    "Test Store",
		Address: "Test Address",
	}
	receipt, err := txService.GenerateReceipt(tx, storeInfo)
	if err != nil {
		t.Fatalf("Failed to generate receipt: %v", err)
	}
	
	// Verify receipt calculations
	expectedTotal := (5.50 * 3) + (12.75 * 1) + (8.25 * 5) // 16.5 + 12.75 + 41.25 = 70.50
	if receipt.TotalAmount != expectedTotal {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, receipt.TotalAmount)
	}
	
	// Process transaction - get proper ephemeral key from mock QR scanner  
	qrScanner := mock.NewMockQRScanner(false)
	ephemeralKeyPEMBase64, err := qrScanner.GetEphemeralKey()
	if err != nil {
		t.Fatalf("Failed to get ephemeral key: %v", err)
	}
	err = txService.ProcessTransaction(receipt, ephemeralKeyPEMBase64)
	if err != nil {
		t.Fatalf("Failed to process transaction: %v", err)
	}
	
	t.Log("Specification compliant workflow test completed successfully")
	t.Logf("Final transaction had %d item types with total ₺%.2f", 
		len(tx.Items), receipt.TotalAmount)
}