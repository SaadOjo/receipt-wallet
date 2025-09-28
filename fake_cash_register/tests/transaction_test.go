package tests

import (
	"testing"

	"fake-cash-register/internal/cashregister"
	"fake-cash-register/internal/crypto"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/models"
	"fake-cash-register/internal/services/mock"
)

// Setup shared data for all tests
var (
	kisimLookup = models.KisimLookup{
		1: {ID: 1, Name: "Test Kisim", TaxRate: 20, PresetPrice: 10.50},
		2: {ID: 2, Name: "Test Kisim 2", TaxRate: 10, PresetPrice: 15.00},
		3: {ID: 3, Name: "Custom Item", TaxRate: 10, PresetPrice: 8.25},
	}
	storeInfo = interfaces.StoreInfo{
		VKN:     "1234567890",
		Name:    "Test Store",
		Address: "Test Address",
	}
)

// createTestCashRegister creates a new cash register for testing with all services
func createTestCashRegister(verbose bool) *cashregister.CashRegister {
	// Import mock package for other services
	revenueAuth := mock.NewMockRevenueAuthority(verbose)
	receiptBank := mock.NewMockReceiptBank(verbose)
	cryptoService := crypto.NewCryptoService(verbose)

	return cashregister.NewCashRegister(
		storeInfo,
		kisimLookup,
		revenueAuth,
		receiptBank,
		cryptoService,
		verbose,
	)
}

func TestTransactionWorkflow(t *testing.T) {
	// Create a new cash register for this test
	cashReg := createTestCashRegister(true)

	// Test 1: Start transaction
	cashReg.StartNewReceipt()
	if !cashReg.HasActiveReceipt() {
		t.Fatal("Failed to start receipt")
	}

	// Test 2: Add items
	err := cashReg.AddItem(1, 1, 0) // KisimID 1, quantity 1
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	currentReceipt := cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(currentReceipt.Items))
	}

	// Test 3: Add another item
	err = cashReg.AddItem(2, 1, 0) // KisimID 2, quantity 1
	if err != nil {
		t.Fatalf("Failed to add second item: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(currentReceipt.Items))
	}

	// Test 4: Set payment method
	err = cashReg.SetPaymentMethod("Nakit")
	if err != nil {
		t.Fatalf("Failed to set payment method: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if currentReceipt.PaymentMethod != "Nakit" {
		t.Errorf("Expected payment method 'Nakit', got '%s'", currentReceipt.PaymentMethod)
	}

	// Test 5: Verify receipt is ready for issuing
	currentReceipt = cashReg.GetCurrentReceipt()
	if currentReceipt == nil {
		t.Fatal("Expected current receipt to exist before issuing")
	}

	// Test 7: Issue receipt (privacy-preserving) - Use the new unified workflow
	// Use QR scanner to generate a proper test ephemeral key
	// Generate test ephemeral key directly (simulating frontend QR scan)
	userEphemeralKeyCompressed := []byte("test_ephemeral_key_32_bytes_long")
	if len(userEphemeralKeyCompressed) != 32 {
		// Pad to 32 bytes for consistency
		userEphemeralKeyCompressed = append(userEphemeralKeyCompressed, make([]byte, 32-len(userEphemeralKeyCompressed))...)
	}

	// Start a new receipt for issuing test
	cashReg.StartNewReceipt()
	err = cashReg.AddItem(1, 1, 0)
	if err != nil {
		t.Fatalf("Failed to add item for issuing test: %v", err)
	}
	err = cashReg.SetPaymentMethod("Nakit")
	if err != nil {
		t.Fatalf("Failed to set payment method for issuing test: %v", err)
	}

	// Test the unified IssueCurrentReceipt method
	issuedReceipt, err := cashReg.IssueCurrentReceipt(userEphemeralKeyCompressed)
	if err != nil {
		t.Fatalf("Failed to issue receipt: %v", err)
	}

	if issuedReceipt == nil {
		t.Fatal("Expected issued receipt, got nil")
	}

	t.Log("Transaction workflow test completed successfully")
}

func TestReceiptCalculations(t *testing.T) {
	// Create a new cash register for this test
	cashReg := createTestCashRegister(false)

	// Start receipt and add test items
	cashReg.StartNewReceipt()

	// Add items with different tax rates to test calculations
	err := cashReg.AddItem(1, 2, 0) // 2x Test Kisim (20% tax, ₺10.50 each)
	if err != nil {
		t.Fatalf("Failed to add item 1: %v", err)
	}

	// Manually create a test item for different tax calculation
	currentReceipt := cashReg.GetCurrentReceipt()
	currentReceipt.Items = append(currentReceipt.Items, models.Item{
		KisimID: 2, Quantity: 1, UnitPrice: 24.0, TotalPrice: 24.0, TaxRate: 20,
	})

	// Finalize to trigger tax calculations
	receipt, err := cashReg.FinalizeCurrentReceipt()
	if err != nil {
		t.Fatalf("Failed to finalize receipt: %v", err)
	}

	// Check total amount (2x10.50 + 24.0 = 45.0)
	expectedTotal := 45.0
	if receipt.TotalAmount != expectedTotal {
		t.Errorf("Expected total amount %.2f, got %.2f", expectedTotal, receipt.TotalAmount)
	}

	// Check that tax breakdown was calculated
	if receipt.TaxBreakdown.TotalTax <= 0 {
		t.Error("Expected tax breakdown to be calculated")
	}

	t.Log("Receipt calculation test completed successfully")
}

func TestMockServices(t *testing.T) {
	// Create services for testing
	revenueAuth := mock.NewMockRevenueAuthority(false)
	receiptBank := mock.NewMockReceiptBank(false)

	// Test revenue authority mock
	// Create a proper 32-byte hash for testing
	hash := []byte("this_is_a_test_hash_32_bytes_lng")
	signature, err := revenueAuth.SignHash(hash)
	if err != nil {
		t.Fatalf("Revenue authority signing failed: %v", err)
	}
	if len(signature) == 0 {
		t.Error("Expected signature from revenue authority")
	}

	// Test revenue authority public key
	publicKey, err := revenueAuth.GetPublicKey()
	if err != nil {
		t.Fatalf("Failed to get public key: %v", err)
	}
	if len(publicKey) == 0 {
		t.Error("Expected public key from revenue authority")
	}

	// Test receipt bank mock - generate a proper ephemeral key
	// Generate test ephemeral key directly (simulating frontend QR scan)
	userEphemeralKeyCompressed := []byte("test_ephemeral_key_32_bytes_long")
	if len(userEphemeralKeyCompressed) != 32 {
		// Pad to 32 bytes for consistency
		userEphemeralKeyCompressed = append(userEphemeralKeyCompressed, make([]byte, 32-len(userEphemeralKeyCompressed))...)
	}

	err = receiptBank.SubmitReceipt(userEphemeralKeyCompressed, []byte("mock_encrypted_data"))
	if err != nil {
		t.Fatalf("Receipt bank submission failed: %v", err)
	}

	t.Log("Mock services test completed successfully")
}

func TestSpecificationCompliantWorkflow(t *testing.T) {
	// Create a new cash register for this test
	cashReg := createTestCashRegister(true)

	// Start receipt
	cashReg.StartNewReceipt()

	// Add multiple items with quantity increments (as per specification)
	err := cashReg.AddItem(1, 1, 0) // KisimID 1, quantity 1
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	currentReceipt := cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 1 || currentReceipt.Items[0].Quantity != 1 {
		t.Errorf("Expected 1 item with quantity 1, got %d items with quantity %d",
			len(currentReceipt.Items), currentReceipt.Items[0].Quantity)
	}

	// Add same item again - should increment quantity
	err = cashReg.AddItem(1, 1, 0) // KisimID 1, quantity 1 (will be added to existing)
	if err != nil {
		t.Fatalf("Failed to increment item quantity: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 1 || currentReceipt.Items[0].Quantity != 2 {
		t.Errorf("Expected 1 item with quantity 2, got %d items with quantity %d",
			len(currentReceipt.Items), currentReceipt.Items[0].Quantity)
	}

	// Add same item once more
	err = cashReg.AddItem(1, 1, 0) // KisimID 1, quantity 1 (will be added to existing)
	if err != nil {
		t.Fatalf("Failed to increment item quantity again: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 1 || currentReceipt.Items[0].Quantity != 3 {
		t.Errorf("Expected 1 item with quantity 3, got %d items with quantity %d",
			len(currentReceipt.Items), currentReceipt.Items[0].Quantity)
	}

	// Add different item
	err = cashReg.AddItem(2, 1, 0) // KisimID 2, quantity 1
	if err != nil {
		t.Fatalf("Failed to add different item: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 2 {
		t.Errorf("Expected 2 different items, got %d", len(currentReceipt.Items))
	}

	// Add third item with higher quantity
	err = cashReg.AddItem(3, 5, 0) // KisimID 3, quantity 5
	if err != nil {
		t.Fatalf("Failed to add third item: %v", err)
	}

	currentReceipt = cashReg.GetCurrentReceipt()
	if len(currentReceipt.Items) != 3 || currentReceipt.Items[2].Quantity != 5 {
		t.Errorf("Expected 3 items with last having quantity 5, got %d items", len(currentReceipt.Items))
	}

	// Set payment method
	err = cashReg.SetPaymentMethod("Nakit")
	if err != nil {
		t.Fatalf("Failed to set payment method: %v", err)
	}

	// Issue receipt (privacy-preserving) using unified workflow - generate proper ephemeral key
	// Generate test ephemeral key directly (simulating frontend QR scan)
	userEphemeralKeyCompressed := []byte("test_ephemeral_key_32_bytes_long")
	if len(userEphemeralKeyCompressed) != 32 {
		// Pad to 32 bytes for consistency
		userEphemeralKeyCompressed = append(userEphemeralKeyCompressed, make([]byte, 32-len(userEphemeralKeyCompressed))...)
	}

	receipt, err := cashReg.IssueCurrentReceipt(userEphemeralKeyCompressed)
	if err != nil {
		t.Fatalf("Failed to issue receipt: %v", err)
	}

	t.Log("Specification compliant workflow test completed successfully")
	t.Logf("Final transaction had 3 item types with total ₺%.2f", receipt.TotalAmount)
}
