package real

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"fake-cash-register/internal/api"
)

type RealRevenueAuthority struct {
	baseURL    string
	httpClient *http.Client
	verbose    bool
}

func NewRealRevenueAuthority(baseURL string, verbose bool) *RealRevenueAuthority {
	return &RealRevenueAuthority{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		verbose: verbose,
	}
}

// SignHash sends binary hash to external revenue authority for signing
func (r *RealRevenueAuthority) SignHash(binaryHash []byte) ([]byte, error) {
	if r.verbose {
		hashBase64 := base64.StdEncoding.EncodeToString(binaryHash)
		log.Printf("[REAL] Revenue Authority: Signing hash %s", hashBase64[:8]+"...")
	}

	// Validate hash format (should be 32 bytes for SHA-256)
	if len(binaryHash) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(binaryHash))
	}

	// Prepare request
	hashBase64 := base64.StdEncoding.EncodeToString(binaryHash)
	signReq := api.SignRequest{
		Hash: hashBase64,
	}

	requestBody, err := json.Marshal(signReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign request: %v", err)
	}

	// Make HTTP request
	url := r.baseURL + "/sign"
	resp, err := r.httpClient.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call revenue authority at %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp api.ErrorResponse
		if json.Unmarshal(responseBody, &errorResp) == nil {
			return nil, fmt.Errorf("revenue authority error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("revenue authority returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	var signResp api.SignResponse
	if err := json.Unmarshal(responseBody, &signResp); err != nil {
		return nil, fmt.Errorf("failed to parse sign response: %v", err)
	}

	// Decode base64 signature to binary
	binarySignature, err := base64.StdEncoding.DecodeString(signResp.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature from base64: %v", err)
	}

	if r.verbose {
		log.Printf("[REAL] Revenue Authority: Received signature %s (%d bytes)",
			signResp.Signature[:16]+"...", len(binarySignature))
	}

	return binarySignature, nil
}

// GetPublicKey fetches the revenue authority's public key
func (r *RealRevenueAuthority) GetPublicKey() ([]byte, error) {
	if r.verbose {
		log.Printf("[REAL] Revenue Authority: Fetching public key")
	}

	// Make HTTP request
	url := r.baseURL + "/public-key"
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call revenue authority at %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp api.ErrorResponse
		if json.Unmarshal(responseBody, &errorResp) == nil {
			return nil, fmt.Errorf("revenue authority error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("revenue authority returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse successful response
	var pubKeyResp api.PublicKeyResponse
	if err := json.Unmarshal(responseBody, &pubKeyResp); err != nil {
		return nil, fmt.Errorf("failed to parse public key response: %v", err)
	}

	// Decode base64 public key to binary
	binaryPublicKey, err := base64.StdEncoding.DecodeString(pubKeyResp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key from base64: %v", err)
	}

	if r.verbose {
		log.Printf("[REAL] Revenue Authority: Received public key (%d bytes)", len(binaryPublicKey))
	}

	return binaryPublicKey, nil
}
