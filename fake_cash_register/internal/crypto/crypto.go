package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"

	"fake-cash-register/internal/models"
)

type CryptoService struct {
	verbose bool
}

func NewCryptoService(verbose bool) *CryptoService {
	return &CryptoService{
		verbose: verbose,
	}
}

// GenerateReceiptHash creates a SHA-256 hash of the receipt data
func (c *CryptoService) GenerateReceiptHash(receipt *models.Receipt) (string, error) {
	if c.verbose {
		log.Printf("[CRYPTO] Generating hash for receipt %s", receipt.TransactionID)
	}
	
	// Serialize receipt to JSON
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return "", fmt.Errorf("failed to marshal receipt: %v", err)
	}
	
	// Calculate SHA-256 hash
	hash := sha256.Sum256(receiptJSON)
	
	// Encode to base64 (44 characters for SHA-256)
	hashB64 := base64.StdEncoding.EncodeToString(hash[:])
	
	if c.verbose {
		log.Printf("[CRYPTO] Generated hash: %s", hashB64)
	}
	
	return hashB64, nil
}

// EncryptWithEphemeralKey encrypts data using the ephemeral public key
func (c *CryptoService) EncryptWithEphemeralKey(data []byte, ephemeralKeyPEM string) (string, error) {
	if c.verbose {
		log.Printf("[CRYPTO] Encrypting %d bytes with ephemeral key", len(data))
	}
	
	// Parse the ephemeral public key
	publicKey, err := c.parsePublicKey(ephemeralKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse ephemeral key: %v", err)
	}
	
	// For ECDSA encryption, we'll use a hybrid approach:
	// 1. Generate a random AES key
	// 2. Encrypt data with AES
	// 3. Encrypt AES key with ECDSA (simulate ECIES)
	
	// For this PoC, we'll use a simplified approach
	// In production, you'd use proper ECIES implementation
	encryptedData, err := c.hybridEncrypt(data, publicKey)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %v", err)
	}
	
	// Encode result to base64
	result := base64.StdEncoding.EncodeToString(encryptedData)
	
	if c.verbose {
		log.Printf("[CRYPTO] Encryption successful, result size: %d bytes", len(result))
	}
	
	return result, nil
}

// ValidateEphemeralKey validates the format and structure of an ephemeral key
func (c *CryptoService) ValidateEphemeralKey(keyPEM string) error {
	if c.verbose {
		log.Printf("[CRYPTO] Validating ephemeral key")
	}
	
	_, err := c.parsePublicKey(keyPEM)
	if err != nil {
		return fmt.Errorf("invalid ephemeral key: %v", err)
	}
	
	if c.verbose {
		log.Printf("[CRYPTO] Ephemeral key validation successful")
	}
	
	return nil
}

// parsePublicKey parses a PEM-encoded public key
func (c *CryptoService) parsePublicKey(keyPEM string) (*ecdsa.PublicKey, error) {
	// Decode base64 first (assuming the key comes base64 encoded from QR)
	keyBytes, err := base64.StdEncoding.DecodeString(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %v", err)
	}
	
	// Try to parse as PEM
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		// If not PEM, try direct parsing
		return c.parseRawPublicKey(keyBytes)
	}
	
	// Parse PEM block
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}
	
	ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA public key")
	}
	
	return ecdsaKey, nil
}

// parseRawPublicKey parses raw public key bytes
func (c *CryptoService) parseRawPublicKey(keyBytes []byte) (*ecdsa.PublicKey, error) {
	// For this PoC, we'll create a mock key for validation
	// In production, you'd parse the actual key bytes
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     elliptic.P256().Params().Gx,
		Y:     elliptic.P256().Params().Gy,
	}, nil
}

// hybridEncrypt performs hybrid encryption (simplified for PoC)
func (c *CryptoService) hybridEncrypt(data []byte, publicKey *ecdsa.PublicKey) ([]byte, error) {
	// For this PoC, we'll use a simplified encryption
	// In production, you'd use proper ECIES or similar
	
	// Generate a random "session key"
	sessionKey := make([]byte, 32)
	_, err := rand.Read(sessionKey)
	if err != nil {
		return nil, err
	}
	
	// "Encrypt" data with session key (XOR for simplicity)
	encryptedData := make([]byte, len(data))
	for i, b := range data {
		encryptedData[i] = b ^ sessionKey[i%32]
	}
	
	// "Encrypt" session key with public key (mock for PoC)
	encryptedSessionKey := make([]byte, 64) // Mock encrypted key
	copy(encryptedSessionKey, sessionKey)
	
	// Combine encrypted session key + encrypted data
	result := append(encryptedSessionKey, encryptedData...)
	
	return result, nil
}

// Mock crypto service for testing
type MockCryptoService struct {
	verbose bool
}

func NewMockCryptoService(verbose bool) *MockCryptoService {
	return &MockCryptoService{verbose: verbose}
}

func (m *MockCryptoService) GenerateReceiptHash(receipt *models.Receipt) (string, error) {
	if m.verbose {
		log.Printf("[MOCK CRYPTO] Generating hash for receipt %s", receipt.TransactionID)
	}
	
	// Simple mock hash
	mockHash := fmt.Sprintf("mock_hash_%s", receipt.TransactionID)
	hash := sha256.Sum256([]byte(mockHash))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

func (m *MockCryptoService) EncryptWithEphemeralKey(data []byte, ephemeralKeyPEM string) (string, error) {
	if m.verbose {
		log.Printf("[MOCK CRYPTO] Encrypting %d bytes", len(data))
	}
	
	// Simple mock encryption (just base64 encode)
	mockEncrypted := "mock_encrypted_" + base64.StdEncoding.EncodeToString(data)
	return base64.StdEncoding.EncodeToString([]byte(mockEncrypted)), nil
}

func (m *MockCryptoService) ValidateEphemeralKey(keyPEM string) error {
	if m.verbose {
		log.Printf("[MOCK CRYPTO] Validating ephemeral key")
	}
	return nil // Always valid for mock
}