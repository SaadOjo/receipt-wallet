package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/hkdf"

	"fake-cash-register/internal/binary"
)

type CryptoService struct {
	verbose bool
}

func NewCryptoService(verbose bool) *CryptoService {
	return &CryptoService{
		verbose: verbose,
	}
}

// GenerateReceiptHash creates a SHA-256 hash of the binary receipt data
func (c *CryptoService) GenerateReceiptHash(binaryReceipt []byte) []byte {
	if c.verbose {
		log.Printf("[CRYPTO] Generating hash for %d byte binary receipt", len(binaryReceipt))
	}
	
	// Calculate SHA-256 hash of binary data
	binaryHash := sha256.Sum256(binaryReceipt)
	
	if c.verbose {
		hashBase64 := binary.ToBase64(binaryHash[:])
		log.Printf("[CRYPTO] Generated hash: %s", hashBase64)
	}
	
	return binaryHash[:]
}

// EncryptWithEphemeralKey encrypts binary data using ECIES with the ephemeral public key
// Strict contract: ephemeralKeyPEMBase64 must be base64(PEM("PUBLIC KEY", ECDSA-P256-PublicKey))
func (c *CryptoService) EncryptWithEphemeralKey(binaryData []byte, ephemeralKeyPEMBase64 string) ([]byte, error) {
	if c.verbose {
		log.Printf("[CRYPTO] Encrypting %d bytes with ephemeral key", len(binaryData))
	}
	
	// Parse the ephemeral public key (strict contract - no fallbacks)
	recipientPublicKey, err := binary.PEMBase64ToPublicKey(ephemeralKeyPEMBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ephemeral key: %v", err)
	}
	
	// Perform ECIES encryption
	binaryEncrypted, err := c.eciesEncrypt(binaryData, recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECIES encryption failed: %v", err)
	}
	
	if c.verbose {
		log.Printf("[CRYPTO] ECIES encryption successful, result size: %d bytes", len(binaryEncrypted))
	}
	
	return binaryEncrypted, nil
}

// ValidateEphemeralKey validates the format and structure of an ephemeral key
// Strict contract: must be base64(PEM("PUBLIC KEY", ECDSA-P256-PublicKey))
func (c *CryptoService) ValidateEphemeralKey(ephemeralKeyPEMBase64 string) error {
	if c.verbose {
		log.Printf("[CRYPTO] Validating ephemeral key")
	}
	
	// Use strict parsing - no fallbacks
	_, err := binary.PEMBase64ToPublicKey(ephemeralKeyPEMBase64)
	if err != nil {
		return fmt.Errorf("invalid ephemeral key: %v", err)
	}
	
	if c.verbose {
		log.Printf("[CRYPTO] Ephemeral key validation successful")
	}
	
	return nil
}


// eciesEncrypt implements proper ECIES encryption
// Returns: ephemeral_public_key || encrypted_data || auth_tag
func (c *CryptoService) eciesEncrypt(data []byte, recipientPublicKey *ecdsa.PublicKey) ([]byte, error) {
	// Step 1: Generate ephemeral key pair
	ephemeralPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %v", err)
	}
	
	// Step 2: Compute ECDH shared secret
	sharedX, _ := recipientPublicKey.Curve.ScalarMult(recipientPublicKey.X, recipientPublicKey.Y, ephemeralPrivateKey.D.Bytes())
	sharedSecret := sharedX.Bytes()
	
	// Step 3: Derive encryption and MAC keys using HKDF
	// We need 32 bytes for AES-256 + 32 bytes for HMAC key
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("ECIES-encryption"))
	keyMaterial := make([]byte, 64)
	if _, err := io.ReadFull(hkdf, keyMaterial); err != nil {
		return nil, fmt.Errorf("failed to derive keys: %v", err)
	}
	
	encryptionKey := keyMaterial[:32]  // AES-256 key
	macKey := keyMaterial[32:]        // HMAC key
	
	// Step 4: Encrypt with AES-GCM (provides both encryption and authentication)
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}
	
	// Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}
	
	// Encrypt data
	ciphertext := aesGCM.Seal(nil, nonce, data, nil)
	
	// Step 5: Serialize ephemeral public key
	ephemeralPublicKeyBytes := elliptic.Marshal(elliptic.P256(), ephemeralPrivateKey.PublicKey.X, ephemeralPrivateKey.PublicKey.Y)
	
	// Step 6: Construct result: ephemeral_public_key || nonce || ciphertext
	result := make([]byte, 0, len(ephemeralPublicKeyBytes)+len(nonce)+len(ciphertext))
	result = append(result, ephemeralPublicKeyBytes...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	
	if c.verbose {
		log.Printf("[CRYPTO] ECIES encryption: ephemeral key %d bytes, nonce %d bytes, ciphertext %d bytes", 
			len(ephemeralPublicKeyBytes), len(nonce), len(ciphertext))
	}
	
	// Clear sensitive data
	for i := range encryptionKey {
		encryptionKey[i] = 0
	}
	for i := range macKey {
		macKey[i] = 0
	}
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}
	
	return result, nil
}

// eciesDecrypt implements proper ECIES decryption (for completeness/testing)
// This would be used by the recipient to decrypt the data
func (c *CryptoService) eciesDecrypt(encryptedData []byte, recipientPrivateKey *ecdsa.PrivateKey) ([]byte, error) {
	curve := elliptic.P256()
	keySize := (curve.Params().BitSize + 7) / 8
	
	// Parse components: ephemeral_public_key || nonce || ciphertext
	if len(encryptedData) < 2*keySize+1+12 { // min size: uncompressed point + 12-byte nonce + some ciphertext
		return nil, fmt.Errorf("encrypted data too short")
	}
	
	// Extract ephemeral public key (uncompressed point: 0x04 + 32 + 32 bytes)
	ephemeralPubKeyBytes := encryptedData[:2*keySize+1]
	x, y := elliptic.Unmarshal(curve, ephemeralPubKeyBytes)
	if x == nil {
		return nil, fmt.Errorf("invalid ephemeral public key")
	}
	
	ephemeralPublicKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	
	// Compute shared secret
	sharedX, _ := curve.ScalarMult(ephemeralPublicKey.X, ephemeralPublicKey.Y, recipientPrivateKey.D.Bytes())
	sharedSecret := sharedX.Bytes()
	
	// Derive keys
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("ECIES-encryption"))
	keyMaterial := make([]byte, 64)
	if _, err := io.ReadFull(hkdf, keyMaterial); err != nil {
		return nil, fmt.Errorf("failed to derive keys: %v", err)
	}
	
	encryptionKey := keyMaterial[:32]
	
	// Extract nonce and ciphertext
	remaining := encryptedData[2*keySize+1:]
	if len(remaining) < 12 {
		return nil, fmt.Errorf("missing nonce")
	}
	
	nonce := remaining[:12]
	ciphertext := remaining[12:]
	
	// Decrypt
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}
	
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}
	
	// Clear sensitive data
	for i := range encryptionKey {
		encryptionKey[i] = 0
	}
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}
	
	return plaintext, nil
}