package mock

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"
)

type MockRevenueAuthority struct {
	verbose bool
}

func NewMockRevenueAuthority(verbose bool) *MockRevenueAuthority {
	return &MockRevenueAuthority{
		verbose: verbose,
	}
}

func (m *MockRevenueAuthority) SignHash(binaryHash []byte) ([]byte, error) {
	if m.verbose {
		hashBase64 := base64.StdEncoding.EncodeToString(binaryHash)
		log.Printf("[MOCK] Revenue Authority: Signing hash %s", hashBase64[:8]+"...")
	}

	// Validate hash format (should be 32 bytes for SHA-256)
	if len(binaryHash) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(binaryHash))
	}

	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)

	// Generate a mock 64-byte ECDSA signature (r||s format)
	binarySignature := make([]byte, 64)

	// Fill with deterministic mock data based on hash
	mockSigString := fmt.Sprintf("mock_signature_%d", time.Now().Unix())
	copy(binarySignature[:32], binaryHash)                                       // Use hash as r component
	copy(binarySignature[32:], []byte(fmt.Sprintf("%-32s", mockSigString))[:32]) // Mock s component

	if m.verbose {
		signatureBase64 := base64.StdEncoding.EncodeToString(binarySignature)
		log.Printf("[MOCK] Revenue Authority: Generated signature %s", signatureBase64[:16]+"...")
	}

	return binarySignature, nil
}

func (m *MockRevenueAuthority) GetPublicKey() ([]byte, error) {
	if m.verbose {
		log.Printf("[MOCK] Revenue Authority: Returning mock public key")
	}

	// Return raw mock public key bytes
	mockKey := "mock_public_key_for_verification_purposes_12345"
	return []byte(mockKey), nil
}
