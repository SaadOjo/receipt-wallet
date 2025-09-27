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

func (m *MockRevenueAuthority) SignHash(hash string) (string, error) {
	if m.verbose {
		log.Printf("[MOCK] Revenue Authority: Signing hash %s", hash[:8]+"...")
	}
	
	// Validate hash format (should be 44 characters base64)
	if len(hash) != 44 {
		return "", fmt.Errorf("invalid hash length: expected 44 characters, got %d", len(hash))
	}
	
	_, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoding: %v", err)
	}
	
	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)
	
	// Generate a mock signature (base64 encoded)
	mockSig := fmt.Sprintf("mock_signature_%d_%s", time.Now().Unix(), hash[:8])
	signature := base64.StdEncoding.EncodeToString([]byte(mockSig))
	
	if m.verbose {
		log.Printf("[MOCK] Revenue Authority: Generated signature %s", signature[:16]+"...")
	}
	
	return signature, nil
}

func (m *MockRevenueAuthority) GetPublicKey() (string, error) {
	if m.verbose {
		log.Printf("[MOCK] Revenue Authority: Returning mock public key")
	}
	
	// Return a mock public key (base64 encoded)
	mockKey := "mock_public_key_for_verification_purposes_12345"
	return base64.StdEncoding.EncodeToString([]byte(mockKey)), nil
}