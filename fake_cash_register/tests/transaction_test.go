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
	// Setup mock services
	revenueAuth := mock.NewMockRevenueAuthority(true)
	receiptBank := mock.NewMockReceiptBank(true)
	cryptoService := crypto.NewMockCryptoService(true)

	// Create transaction service
	txService := services.NewTransactionService(
		revenueAuth,
		receiptBank,
		cryptoService,
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
	err := txService.AddItem(tx, 1, "Test Kisim", 10.50, 20, "Test description")
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}
	if len(tx.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(tx.Items))
	}

	// Test 3: Add another item
	err = txService.AddItem(tx, 2, "Test Kisim 2", 15.00, 10, "Test description 2")
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

	// Test 6: Process transaction
	err = txService.ProcessTransaction(receipt, "mock_ephemeral_key")
	if err != nil {
		t.Fatalf("Failed to process transaction: %v", err)
	}

	t.Log("Transaction workflow test completed successfully")
}

func TestReceiptCalculations(t *testing.T) {
	receipt := &models.Receipt{
		Items: []models.Item{
			{KisimName: "Item 1", Quantity: 2, UnitPrice: 10.0, TotalPrice: 20.0, TaxRate: 10},
			{KisimName: "Item 2", Quantity: 1, UnitPrice: 24.0, TotalPrice: 24.0, TaxRate: 20},
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
	
	// Use a proper 44-character base64 hash (SHA-256 in base64 is always 44 chars)
	signature, err := revenueAuth.SignHash("dGVzdCBoYXNoIGZvciB0ZXN0aW5nIHB1cnBvc2VzPQ==")
	if err != nil {
		t.Fatalf("Mock revenue authority sign failed: %v", err)
	}
	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	publicKey, err := revenueAuth.GetPublicKey()
	if err != nil {
		t.Fatalf("Mock revenue authority get public key failed: %v", err)
	}
	if publicKey == "" {
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
	
	key, err := qrScanner.GetEphemeralKey()
	if err != nil {
		t.Fatalf("Mock QR scanner failed: %v", err)
	}
	if key == "" {
		t.Error("Expected non-empty ephemeral key")
	}

	err = qrScanner.ValidateKey(key)
	if err != nil {
		t.Fatalf("Mock QR scanner validation failed: %v", err)
	}

	t.Log("Mock services test completed successfully")
}