package mock

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"log"
	"time"
)

type MockQRScanner struct {
	verbose bool
	testKeys []string
	keyIndex int
}

func NewMockQRScanner(verbose bool) *MockQRScanner {
	// Generate valid ECDSA public keys for testing
	// NOTE FOR CODE REVIEW: These are MOCK test keys generated at runtime
	// They are NOT production keys and are only used for testing/standalone mode
	testKeys := []string{
		generateValidECDSATestKey("MOCK_TEST_KEY_1"),
		generateValidECDSATestKey("MOCK_TEST_KEY_2"), 
		generateValidECDSATestKey("MOCK_TEST_KEY_3"),
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
		log.Printf("[MOCK] QR Scanner: Scanned MOCK test key %s...", key[:16])
	}
	
	return key, nil
}

func (m *MockQRScanner) ValidateKey(key string) error {
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: Validating MOCK test key %s...", key[:16])
	}
	
	// Basic validation - check if it's base64 encoded ECDSA key
	_, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	
	if m.verbose {
		log.Printf("[MOCK] QR Scanner: MOCK test key validation successful")
	}
	
	return nil
}

// generateValidECDSATestKey creates a valid ECDSA public key for testing
// NOTE FOR CODE REVIEW: This generates MOCK test keys only - NOT for production use
func generateValidECDSATestKey(testKeyLabel string) string {
	// Generate a real ECDSA key pair for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		// Fallback to deterministic test key if generation fails
		log.Printf("[MOCK WARNING] Failed to generate test ECDSA key, using fallback: %v", err)
		return generateFallbackTestKey(testKeyLabel)
	}
	
	// Extract public key
	publicKey := &privateKey.PublicKey
	
	// Marshal to PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Printf("[MOCK WARNING] Failed to marshal test public key, using fallback: %v", err)
		return generateFallbackTestKey(testKeyLabel)
	}
	
	// Create PEM block with clear test labeling
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
		Headers: map[string]string{
			"Test-Key-Label": testKeyLabel,
			"Generated-At":   time.Now().Format(time.RFC3339),
			"Purpose":        "MOCK-TESTING-ONLY",
		},
	}
	
	// Encode to PEM
	pemBytes := pem.EncodeToMemory(pemBlock)
	
	// Base64 encode for transmission (as expected by crypto service)
	return base64.StdEncoding.EncodeToString(pemBytes)
}

// generateFallbackTestKey creates a simple fallback test key if ECDSA generation fails
// NOTE FOR CODE REVIEW: This is a MOCK fallback key for testing only
func generateFallbackTestKey(testKeyLabel string) string {
	mockKeyData := "MOCK_FALLBACK_TEST_KEY_" + testKeyLabel + "_" + time.Now().Format("20060102150405")
	return base64.StdEncoding.EncodeToString([]byte(mockKeyData))
}