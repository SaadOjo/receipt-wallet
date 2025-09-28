package binary

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
)

// Core encoding/decoding functions - no wrappers

// ToBase64 encodes binary data to base64 string
func ToBase64(binaryData []byte) string {
	return base64.StdEncoding.EncodeToString(binaryData)
}

// FromBase64 decodes base64 string to binary data
func FromBase64(base64String string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64String)
}

// ECDSA signature encoding/decoding

// EncodeECDSASignature converts ECDSA signature (r, s) to 64-byte binary format
func EncodeECDSASignature(r, s *big.Int) ([]byte, error) {
	binarySignature := make([]byte, SignatureSize)
	
	// Convert r to 32-byte big-endian
	rBytes := r.Bytes()
	if len(rBytes) > 32 {
		return nil, fmt.Errorf("r component too large: %d bytes", len(rBytes))
	}
	copy(binarySignature[32-len(rBytes):32], rBytes)
	
	// Convert s to 32-byte big-endian
	sBytes := s.Bytes()
	if len(sBytes) > 32 {
		return nil, fmt.Errorf("s component too large: %d bytes", len(sBytes))
	}
	copy(binarySignature[64-len(sBytes):64], sBytes)
	
	return binarySignature, nil
}

// DecodeECDSASignature converts 64-byte binary format to ECDSA signature (r, s)
func DecodeECDSASignature(binarySignature []byte) (*big.Int, *big.Int, error) {
	if len(binarySignature) != SignatureSize {
		return nil, nil, fmt.Errorf("invalid signature size: expected %d bytes, got %d", SignatureSize, len(binarySignature))
	}
	
	r := new(big.Int).SetBytes(binarySignature[:32])
	s := new(big.Int).SetBytes(binarySignature[32:64])
	
	return r, s, nil
}

// PEM public key encoding/decoding

// PublicKeyToPEMBase64 encodes ECDSA public key to PEM format, then to base64
func PublicKeyToPEMBase64(publicKey *ecdsa.PublicKey) (string, error) {
	// Marshal to PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}
	
	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	
	// Encode to PEM
	pemBytes := pem.EncodeToMemory(pemBlock)
	
	// Encode to base64
	return ToBase64(pemBytes), nil
}

// PEMBase64ToPublicKey decodes base64 PEM to ECDSA public key
func PEMBase64ToPublicKey(pemBase64 string) (*ecdsa.PublicKey, error) {
	// Decode base64
	pemBytes, err := FromBase64(pemBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	
	// Parse PEM
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}
	
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid PEM block type: expected 'PUBLIC KEY', got '%s'", block.Type)
	}
	
	// Parse public key
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