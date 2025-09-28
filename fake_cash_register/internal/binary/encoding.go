package binary

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
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

// NOTE: ECDSA signature encoding/decoding functions are intentionally NOT implemented.
// This cash register system receives signatures as pre-formatted 64-byte binary data from
// the Revenue Authority service and treats them as opaque blobs. The signatures are simply
// concatenated to receipt data and encrypted - no encoding/decoding of (r,s) components needed.
// Signature encoding would only be needed if we were implementing ECDSA signing ourselves
// or verifying signatures (which requires access to r,s components).

// ECDSA public key encoding/decoding (raw compressed format for QR codes)

// RawCompressedToPublicKey converts 33-byte compressed ECDSA key to public key object
func RawCompressedToPublicKey(compressed []byte) (*ecdsa.PublicKey, error) {
	if len(compressed) != 33 {
		return nil, fmt.Errorf("invalid compressed key size: expected 33 bytes, got %d", len(compressed))
	}

	// Decompress the point manually
	curve := elliptic.P256()
	x, y := decompressPoint(curve, compressed)
	if x == nil {
		return nil, fmt.Errorf("failed to decompress public key point")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// PublicKeyToRawCompressed converts ECDSA public key to 33-byte compressed format
func PublicKeyToRawCompressed(publicKey *ecdsa.PublicKey) ([]byte, error) {
	// Compress the point manually
	return compressPoint(publicKey.Curve, publicKey.X, publicKey.Y), nil
}

// compressPoint compresses an elliptic curve point to 33 bytes
func compressPoint(curve elliptic.Curve, x, y *big.Int) []byte {
	compressed := make([]byte, 33)

	// X coordinate (32 bytes, big-endian)
	xBytes := x.Bytes()
	copy(compressed[33-len(xBytes):], xBytes)

	// Y parity bit: 0x02 if Y is even, 0x03 if Y is odd
	if y.Bit(0) == 0 {
		compressed[0] = 0x02
	} else {
		compressed[0] = 0x03
	}

	return compressed
}

// decompressPoint decompresses a 33-byte compressed point
func decompressPoint(curve elliptic.Curve, compressed []byte) (*big.Int, *big.Int) {
	if len(compressed) != 33 || (compressed[0] != 0x02 && compressed[0] != 0x03) {
		return nil, nil
	}

	// Extract X coordinate
	x := new(big.Int).SetBytes(compressed[1:])

	// Calculate Y coordinate using curve equation: y² = x³ - 3x + b
	p := curve.Params().P

	// x³
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)

	// 3x
	threeX := new(big.Int).Mul(x, big.NewInt(3))

	// x³ - 3x + b
	ySquared := new(big.Int).Sub(x3, threeX)
	ySquared.Add(ySquared, curve.Params().B)
	ySquared.Mod(ySquared, p)

	// Calculate square root mod p
	y := new(big.Int).ModSqrt(ySquared, p)
	if y == nil {
		return nil, nil
	}

	// Choose correct root based on parity bit
	if y.Bit(0) != uint(compressed[0]&1) {
		y.Sub(p, y)
	}

	// Verify point is on curve
	if !curve.IsOnCurve(x, y) {
		return nil, nil
	}

	return x, y
}
