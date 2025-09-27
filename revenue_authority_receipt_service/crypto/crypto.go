package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

type CryptoService struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

func NewCryptoService(privateKeyPath, publicKeyPath string) *CryptoService {
	privateKey := loadPrivateKey(privateKeyPath)
	publicKey := loadPublicKey(publicKeyPath)
	
	return &CryptoService{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

func (c *CryptoService) SignHash(hashBase64 string) (string, error) {
	if len(hashBase64) != 44 {
		return "", fmt.Errorf("invalid hash length: expected 44 characters, got %d", len(hashBase64))
	}

	hashBytes, err := base64.StdEncoding.DecodeString(hashBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoding: %v", err)
	}

	if len(hashBytes) != 32 {
		return "", fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hashBytes))
	}

	r, s, err := ecdsa.Sign(rand.Reader, c.privateKey, hashBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign hash: %v", err)
	}

	signature := append(r.Bytes(), s.Bytes()...)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (c *CryptoService) GetPublicKeyBase64() (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(c.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}
	
	return base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}

func loadPrivateKey(path string) *ecdsa.PrivateKey {
	keyData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read private key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatalf("Failed to decode PEM block for private key")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	return privateKey
}

func loadPublicKey(path string) *ecdsa.PublicKey {
	keyData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read public key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatalf("Failed to decode PEM block for public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse public key: %v", err)
	}

	ecdsaPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("Public key is not ECDSA")
	}

	return ecdsaPublicKey
}