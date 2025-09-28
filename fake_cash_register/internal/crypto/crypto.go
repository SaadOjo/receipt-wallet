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

// EncryptWithUserEphemeralKey encrypts binary data using user's ephemeral public key
// Privacy-preserving: User generates ephemeral keys, cash register encrypts with user's public key
// Strict contract: userEphemeralKeyCompressed must be 33-byte raw compressed ECDSA-P256 key
func (c *CryptoService) EncryptWithUserEphemeralKey(binaryData []byte, userEphemeralKeyCompressed []byte) ([]byte, error) {
	if c.verbose {
		log.Printf("[CRYPTO] Encrypting %d bytes with user's ephemeral key", len(binaryData))
	}

	// Parse the user's ephemeral public key (strict contract - no fallbacks)
	userPublicKey, err := binary.RawCompressedToPublicKey(userEphemeralKeyCompressed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ephemeral key: %v", err)
	}

	// Perform privacy-preserving encryption (no cash register keys involved)
	binaryEncrypted, err := c.encryptWithPublicKey(binaryData, userPublicKey)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %v", err)
	}

	if c.verbose {
		log.Printf("[CRYPTO] Privacy-preserving encryption successful, result size: %d bytes", len(binaryEncrypted))
	}

	return binaryEncrypted, nil
}

// ValidateUserEphemeralKey validates the format and structure of user's ephemeral key
// Strict contract: must be 33-byte raw compressed ECDSA-P256 key
func (c *CryptoService) ValidateUserEphemeralKey(userEphemeralKeyCompressed []byte) error {
	if c.verbose {
		log.Printf("[CRYPTO] Validating user's ephemeral key")
	}

	// Use strict parsing - no fallbacks
	_, err := binary.RawCompressedToPublicKey(userEphemeralKeyCompressed)
	if err != nil {
		return fmt.Errorf("invalid user ephemeral key: %v", err)
	}

	if c.verbose {
		log.Printf("[CRYPTO] User ephemeral key validation successful")
	}

	return nil
}

// encryptWithPublicKey implements privacy-preserving encryption using user's ephemeral public key
// Privacy model: Cash register generates temporary private key, uses ECDH with user's public key
// Returns: nonce || encrypted_data || auth_tag (no keys in output - user already has the ephemeral private key)
func (c *CryptoService) encryptWithPublicKey(binaryData []byte, userEphemeralPublicKey *ecdsa.PublicKey) ([]byte, error) {
	// Privacy-preserving ECDH: Cash register generates random private key for this encryption
	// User can decrypt because they have the corresponding ephemeral private key

	// Step 1: Generate a temporary private key for ECDH (not stored or transmitted)
	tempPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate temporary key: %v", err)
	}

	// Step 2: Perform ECDH using user's ephemeral public key and our temporary private key
	sharedX, _ := userEphemeralPublicKey.Curve.ScalarMult(
		userEphemeralPublicKey.X, userEphemeralPublicKey.Y,
		tempPrivateKey.D.Bytes())
	sharedSecret := sharedX.Bytes()

	// Step 3: Derive encryption key from shared secret
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("Privacy-preserving-ECDH"))
	encryptionKey := make([]byte, 32) // AES-256 key
	if _, err := io.ReadFull(hkdf, encryptionKey); err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %v", err)
	}

	// Step 4: Encrypt with AES-GCM
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Step 5: Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Step 6: Encrypt data
	ciphertext := aesGCM.Seal(nil, nonce, binaryData, nil)

	// Step 7: Include temporary public key in result for user to perform ECDH
	tempPublicKeyBytes := elliptic.Marshal(elliptic.P256(), tempPrivateKey.PublicKey.X, tempPrivateKey.PublicKey.Y)

	// Step 8: Construct result: temp_public_key || nonce || ciphertext
	result := make([]byte, 0, len(tempPublicKeyBytes)+len(nonce)+len(ciphertext))
	result = append(result, tempPublicKeyBytes...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	if c.verbose {
		log.Printf("[CRYPTO] Privacy-preserving ECDH: temp key %d bytes, nonce %d bytes, ciphertext %d bytes",
			len(tempPublicKeyBytes), len(nonce), len(ciphertext))
	}

	// Clear sensitive data
	for i := range encryptionKey {
		encryptionKey[i] = 0
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
