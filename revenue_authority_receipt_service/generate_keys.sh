#!/bin/bash

# Revenue Authority Receipt Service - Key Generation Script
# Generates ECDSA P-256 key pair using OpenSSL

PRIVATE_KEY_PATH="keys/private_key.pem"
PUBLIC_KEY_PATH="keys/public_key.pem"

# Create keys directory if it doesn't exist
mkdir -p keys

# Check if keys already exist
if [ -f "$PRIVATE_KEY_PATH" ] || [ -f "$PUBLIC_KEY_PATH" ]; then
    echo "Warning: Key files already exist:"
    [ -f "$PRIVATE_KEY_PATH" ] && echo "  - $PRIVATE_KEY_PATH"
    [ -f "$PUBLIC_KEY_PATH" ] && echo "  - $PUBLIC_KEY_PATH"
    echo
    read -p "Do you want to override the existing keys? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Key generation cancelled."
        exit 0
    fi
fi

# Generate ECDSA private key (P-256 curve)
echo "Generating ECDSA private key..."
if ! openssl ecparam -genkey -name prime256v1 -noout -out "$PRIVATE_KEY_PATH"; then
    echo "Error: Failed to generate private key"
    exit 1
fi

# Extract public key from private key
echo "Extracting public key..."
if ! openssl ec -in "$PRIVATE_KEY_PATH" -pubout -out "$PUBLIC_KEY_PATH" 2>/dev/null; then
    echo "Error: Failed to extract public key"
    exit 1
fi

echo "ECDSA key pair generated successfully:"
echo "  - Private key: $PRIVATE_KEY_PATH"
echo "  - Public key: $PUBLIC_KEY_PATH"
