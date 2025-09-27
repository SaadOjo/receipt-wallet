package mock

import (
	"encoding/base64"
	"log"
	"time"
)

type MockQRScanner struct {
	verbose bool
	testKeys []string
	keyIndex int
}

func NewMockQRScanner(verbose bool) *MockQRScanner {
	// Generate some test ephemeral keys
	testKeys := []string{
		generateMockKey("test_ephemeral_key_1"),
		generateMockKey("test_ephemeral_key_2"),
		generateMockKey("test_ephemeral_key_3"),
	}
	
	return &MockQRScanner{
		verbose:  verbose,
		testKeys: testKeys,
		keyIndex: 0,
	}
}

func (m *MockQRScanner) GetEphemeralKey() (string, error) {
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Simulating QR code scan...")
	}
	
	// Simulate scanning delay
	time.Sleep(300 * time.Millisecond)
	
	// Cycle through test keys
	key := m.testKeys[m.keyIndex]
	m.keyIndex = (m.keyIndex + 1) % len(m.testKeys)
	
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Scanned key %s...", key[:16])
	}
	
	return key, nil
}

func (m *MockQRScanner) ValidateKey(key string) error {
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Validating key %s...", key[:16])
	}
	
	// Basic validation - check if it's base64
	_, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Key validation successful")
	}
	
	return nil
}

func generateMockKey(seed string) string {
	mockKeyData := "mock_ephemeral_public_key_" + seed + "_" + time.Now().Format("20060102")
	return base64.StdEncoding.EncodeToString([]byte(mockKeyData))
}