package mock

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"fake-cash-register/internal/binary"
)

type MockQRScanner struct {
	verbose  bool
	testKeys [][]byte
	keyIndex int
}

func NewMockQRScanner(verbose bool) *MockQRScanner {
	// Generate valid compressed ECDSA public keys for testing
	// NOTE FOR CODE REVIEW: These are MOCK test keys generated at runtime
	// They are NOT production keys and are only used for testing/standalone mode
	testKeys := [][]byte{
		generateCompressedTestKey("MOCK_TEST_KEY_1"),
		generateCompressedTestKey("MOCK_TEST_KEY_2"),
		generateCompressedTestKey("MOCK_TEST_KEY_3"),
	}

	return &MockQRScanner{
		verbose:  verbose,
		testKeys: testKeys,
		keyIndex: 0,
	}
}

func (m *MockQRScanner) GetEphemeralKey() ([]byte, error) {
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Simulating QR code scan...")
	}

	// Simulate scanning delay
	time.Sleep(300 * time.Millisecond)

	// Cycle through test keys
	key := m.testKeys[m.keyIndex]
	m.keyIndex = (m.keyIndex + 1) % len(m.testKeys)

	if m.verbose {
		keyBase64 := binary.ToBase64(key)
		log.Printf("[MOCK] QR Scanner: Scanned MOCK test key %s... (%d bytes)", keyBase64[:16], len(key))
	}

	return key, nil
}

func (m *MockQRScanner) ValidateKey(key []byte) error {
	if m.verbose {
		keyBase64 := binary.ToBase64(key)
		log.Printf("[MOCK] QR Scanner: Validating MOCK test key %s... (%d bytes)", keyBase64[:16], len(key))
	}

	// Basic validation - check if it's 33-byte compressed key
	if len(key) != 33 {
		return fmt.Errorf("invalid key size: expected 33 bytes, got %d", len(key))
	}

	if key[0] != 0x02 && key[0] != 0x03 {
		return fmt.Errorf("invalid compressed key format: expected 0x02 or 0x03 prefix, got 0x%02x", key[0])
	}

	// Try to parse to ensure it's valid
	_, err := binary.RawCompressedToPublicKey(key)
	if err != nil {
		return fmt.Errorf("invalid compressed key: %v", err)
	}

	if m.verbose {
		log.Printf("[MOCK] QR Scanner: MOCK test key validation successful")
	}

	return nil
}

// generateCompressedTestKey creates a valid 33-byte compressed ECDSA public key for testing
// NOTE FOR CODE REVIEW: This generates MOCK test keys only - NOT for production use
func generateCompressedTestKey(testKeyLabel string) []byte {
	// Generate a real ECDSA key pair for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		// Explicit failure - better for debugging than fallback
		panic(fmt.Sprintf("[MOCK] Failed to generate test ECDSA key for %s: %v", testKeyLabel, err))
	}

	// Extract public key and compress
	compressed, err := binary.PublicKeyToRawCompressed(&privateKey.PublicKey)
	if err != nil {
		// Explicit failure - better for debugging than fallback
		panic(fmt.Sprintf("[MOCK] Failed to compress test public key for %s: %v", testKeyLabel, err))
	}

	return compressed
}
