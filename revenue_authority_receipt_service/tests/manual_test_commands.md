# Manual Testing Commands

## Prerequisites
1. Start the service: `go run main.go`
2. Service should be running on `http://localhost:4406`

## Test 1: Get Public Key
```bash
curl -X GET http://localhost:4406/public-key
```
**Expected**: JSON response with base64-encoded public key

## Test 2: Valid Hash Signing
```bash
# Generate a test hash (SHA-256 of "test message" in base64)
# "test message" -> SHA-256 -> base64 = "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="

curl -X POST http://localhost:4406/sign \
  -H "Content-Type: application/json" \
  -d '{"hash": "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ="}'
```
**Expected**: JSON response with base64-encoded signature

## Test 3: Invalid Hash Length (Too Short)
```bash
curl -X POST http://localhost:4406/sign \
  -H "Content-Type: application/json" \
  -d '{"hash": "invalid"}'
```
**Expected**: 400 error with "invalid hash length" message

## Test 4: Invalid Base64 Format
```bash
curl -X POST http://localhost:4406/sign \
  -H "Content-Type: application/json" \
  -d '{"hash": "!@#$%^&*()1234567890!@#$%^&*()1234567890ab"}'
```
**Expected**: 400 error with "invalid base64 encoding" message

## Test 5: Missing Hash Field
```bash
curl -X POST http://localhost:4406/sign \
  -H "Content-Type: application/json" \
  -d '{}'
```
**Expected**: 400 error with "Invalid request format" message

## Generate Test Hashes
To generate your own test hashes:
```bash
# Using openssl
echo -n "your message" | openssl dgst -sha256 -binary | base64

# Using Python
python3 -c "import hashlib, base64; print(base64.b64encode(hashlib.sha256(b'your message').digest()).decode())"
```