# Binary Receipt Format Specification

## Overview

This document defines the binary receipt format used by the Turkish Cash Register system. The format is designed to be deterministic, compact, and version-aware to ensure consistent hash generation across different implementations.

## Design Principles

1. **Deterministic** - Same receipt data always produces identical binary representation
2. **Compact** - Minimal overhead for network transmission and storage
3. **Version-aware** - Format version embedded for future extensibility
4. **Cross-platform** - Works consistently across different architectures
5. **Hash-stable** - Minor format changes don't break existing hash calculations

## Data Encoding

### Byte Order
All multi-byte integers use **Big-Endian** (network byte order) encoding.

### String Encoding
All strings are encoded as UTF-8 with length prefix:
```
[4 bytes length][UTF-8 bytes]
```

### Decimal Encoding
Monetary values are encoded as **fixed-point integers** with 2 decimal places:
- Price ₺12.34 → 1234 (uint32)
- Price ₺0.05 → 5 (uint32)

### Timestamp Encoding
Unix timestamp as 64-bit integer (seconds since epoch).

## Binary Receipt Format v1

### Format Header
```
Offset  Size  Field           Description
------  ----  -----           -----------
0       2     Magic           0x5452 ('TR' for Turkish Receipt)
2       1     Version         0x01 (Format version 1)
3       1     Reserved        0x00 (Must be zero)
```

### Receipt Data Structure
```
Offset  Size  Field                Description
------  ----  -----                -----------
4       8     Timestamp            Unix timestamp (uint64)
12      4     ZReportNumber        Z-Report number (uint32)
16      4     TransactionID        Transaction ID (uint32)
20      4     StoreVKN             Store VKN (uint32, numeric only)
24      4     StoreName Length     UTF-8 byte count
28      N     StoreName            UTF-8 encoded store name
28+N    4     StoreAddress Length  UTF-8 byte count
32+N    M     StoreAddress         UTF-8 encoded store address
32+N+M  4     TotalAmount          Total in kuruş (uint32)
36+N+M  4     PaymentMethod Length UTF-8 byte count
40+N+M  P     PaymentMethod        UTF-8 encoded payment method
40+N+M+P 4    ReceiptSerial        Receipt serial number (uint32)
44+N+M+P 2    ItemCount            Number of items (uint16)
```

### Item Data Structure (repeated ItemCount times)
```
Offset  Size  Field        Description
------  ----  -----        -----------
0       2     KisimID      KISIM identifier (uint16)
2       2     Quantity     Item quantity (uint16)
4       4     UnitPrice    Unit price in kuruş (uint32)
8       4     TotalPrice   Total price in kuruş (uint32)
12      1     TaxRate      Tax rate percentage (uint8)
```
**Item size: 13 bytes per item**

### Tax Breakdown Structure
```
Offset  Size  Field              Description
------  ----  -----              -----------
0       4     Tax10Base          10% tax base amount in kuruş (uint32)
4       4     Tax10Amount        10% tax amount in kuruş (uint32)
8       4     Tax20Base          20% tax base amount in kuruş (uint32)
12      4     Tax20Amount        20% tax amount in kuruş (uint32)
16      4     TotalTax           Total tax amount in kuruş (uint32)
```
**Tax breakdown size: 20 bytes**

## Complete Format Layout

```
┌─────────────────────────────────┐
│ Header (4 bytes)                │
├─────────────────────────────────┤
│ Receipt Metadata (Variable)     │
├─────────────────────────────────┤
│ Item Data (13 × ItemCount)      │
├─────────────────────────────────┤
│ Tax Breakdown (20 bytes)        │
└─────────────────────────────────┘
```

## Example Receipt Binary Layout

For a receipt with:
- Store: "Demo Mağazası" (13 UTF-8 bytes)
- Address: "Kadıköy/İstanbul" (16 UTF-8 bytes)
- 2 items
- Payment: "Nakit" (5 UTF-8 bytes)

```
Byte Range    Content
----------    -------
0-3          Header: 0x5452 0x01 0x00
4-11         Timestamp: Unix time
12-15        Z-Report: 0x00000001
16-19        Transaction ID: 0x12345678
20-23        Store VKN: 0x499602D2 (1234567890)
24-27        Store name length: 0x0000000D (13)
28-40        Store name: "Demo Mağazası" (UTF-8)
41-44        Address length: 0x00000010 (16)
45-60        Address: "Kadıköy/İstanbul" (UTF-8)
61-64        Total: 0x00001388 (5000 kuruş = ₺50.00)
65-68        Payment length: 0x00000005 (5)
69-73        Payment: "Nakit" (UTF-8)
74-77        Receipt serial: 0x00000001
78-79        Item count: 0x0002 (2 items)
80-92        Item 1: KisimID=1, Qty=2, Unit=₺10.50, Total=₺21.00, Tax=20%
93-105       Item 2: KisimID=2, Qty=1, Unit=₺29.00, Total=₺29.00, Tax=20%
106-125      Tax breakdown: bases and amounts
```

## Signed Receipt Format

The signed receipt format concatenates the binary receipt with the signature:

```
┌─────────────────────────────────┐
│ Binary Receipt (Variable Size)  │
├─────────────────────────────────┤
│ ECDSA Signature (64 bytes)      │ <- r (32 bytes) + s (32 bytes)
└─────────────────────────────────┘
```

### Signature Format
- **Algorithm**: ECDSA with P-256 curve and SHA-256 hash
- **Encoding**: Raw binary format (r || s)
- **Size**: Fixed 64 bytes
  - r component: 32 bytes (big-endian)
  - s component: 32 bytes (big-endian)

## Encrypted Signed Receipt Format

The final encrypted format is the ECIES encryption of the signed receipt:

```
┌─────────────────────────────────┐
│ Ephemeral Public Key (65 bytes) │ <- Uncompressed P-256 point
├─────────────────────────────────┤
│ Nonce (12 bytes)               │ <- AES-GCM nonce
├─────────────────────────────────┤
│ Encrypted Data + Auth Tag      │ <- AES-GCM output
└─────────────────────────────────┘
```

## Network Transmission

For transmission over HTTP/JSON APIs, the binary data is encoded as Base64:

```json
{
  "encrypted_receipt": "<base64-encoded-binary-data>",
  "ephemeral_key": "<base64-encoded-pem-public-key>"
}
```

## Version Evolution

Future format versions can:
1. Add new fields at the end of structures
2. Modify the version byte in the header
3. Maintain backward compatibility through version detection

### Planned Version 2 Features
- Digital timestamps with nanosecond precision
- Extended KISIM ID space (uint32)
- Customer identification fields
- Multi-currency support

## Implementation Guidelines

### Hash Calculation
1. Serialize receipt to binary format v1
2. Calculate SHA-256 hash of binary data
3. Use hash for signature verification

### Parser Implementation
1. Verify magic bytes (0x5452)
2. Check version byte and route to appropriate parser
3. Validate all length fields before reading
4. Verify that total item count matches actual items
5. Validate tax calculations

### Error Handling
- Invalid magic bytes → "Invalid receipt format"
- Unsupported version → "Unsupported receipt version X"
- Truncated data → "Corrupted receipt data"
- Invalid UTF-8 → "Invalid text encoding"

## Security Considerations

1. **Hash Integrity**: Binary format ensures identical hashes across platforms
2. **Signature Verification**: Fixed 64-byte signature format simplifies validation
3. **Replay Protection**: Timestamps and unique transaction IDs prevent replay attacks
4. **Data Integrity**: ECIES provides authenticated encryption

## Compliance

This format is designed for Turkish Ministry of Treasury and Finance digital receipt requirements and supports:
- KDV (Value Added Tax) calculations
- Z-Report compliance
- Revenue Authority signature verification
- Receipt Bank submission standards

---

**Document Version**: 1.0  
**Last Updated**: 2025-09-28  
**Compatibility**: Turkish Cash Register System v1.0+