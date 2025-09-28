package binary

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"fake-cash-register/internal/models"
)

const (
	// Binary receipt format constants
	MagicBytes    = 0x5452 // 'TR' for Turkish Receipt
	FormatVersion = 0x01   // Version 1
	Reserved      = 0x00   // Reserved byte (must be zero)

	// Fixed field sizes
	HeaderSize       = 4
	TimestampSize    = 8
	ZReportSize      = 4
	TransactionSize  = 4
	StoreVKNSize     = 4
	TotalAmountSize  = 4
	ReceiptSerSize   = 4
	ItemCountSize    = 2
	ItemSize         = 13 // KisimID(2) + Quantity(2) + UnitPrice(4) + TotalPrice(4) + TaxRate(1)
	TaxBreakdownSize = 20 // Tax10Base(4) + Tax10Amount(4) + Tax20Base(4) + Tax20Amount(4) + TotalTax(4)

	// ECDSA signature size (P-256: r(32) + s(32))
	SignatureSize = 64
)

// SerializeReceipt converts a models.Receipt to binary format v1
func SerializeReceipt(receipt *models.Receipt) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Header (4 bytes)
	if err := binary.Write(buf, binary.BigEndian, uint16(MagicBytes)); err != nil {
		return nil, fmt.Errorf("failed to write magic bytes: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, uint8(FormatVersion)); err != nil {
		return nil, fmt.Errorf("failed to write version: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, uint8(Reserved)); err != nil {
		return nil, fmt.Errorf("failed to write reserved byte: %v", err)
	}

	// Receipt metadata
	timestamp := uint64(receipt.Timestamp.Unix())
	if err := binary.Write(buf, binary.BigEndian, timestamp); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %v", err)
	}

	// Parse Z-Report number (remove 'Z' prefix)
	zReportNum, err := parseZReportNumber(receipt.ZReportNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Z-Report number: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, zReportNum); err != nil {
		return nil, fmt.Errorf("failed to write Z-Report number: %v", err)
	}

	// Parse Transaction ID (remove 'TX' prefix and date)
	txID, err := parseTransactionID(receipt.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction ID: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, txID); err != nil {
		return nil, fmt.Errorf("failed to write transaction ID: %v", err)
	}

	// Parse Store VKN (string to uint32)
	storeVKN, err := parseVKN(receipt.StoreVKN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse store VKN: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, storeVKN); err != nil {
		return nil, fmt.Errorf("failed to write store VKN: %v", err)
	}

	// Store name (length + UTF-8 bytes)
	storeNameBytes := []byte(receipt.StoreName)
	if err := binary.Write(buf, binary.BigEndian, uint32(len(storeNameBytes))); err != nil {
		return nil, fmt.Errorf("failed to write store name length: %v", err)
	}
	if _, err := buf.Write(storeNameBytes); err != nil {
		return nil, fmt.Errorf("failed to write store name: %v", err)
	}

	// Store address (length + UTF-8 bytes)
	storeAddressBytes := []byte(receipt.StoreAddress)
	if err := binary.Write(buf, binary.BigEndian, uint32(len(storeAddressBytes))); err != nil {
		return nil, fmt.Errorf("failed to write store address length: %v", err)
	}
	if _, err := buf.Write(storeAddressBytes); err != nil {
		return nil, fmt.Errorf("failed to write store address: %v", err)
	}

	// Total amount (convert to kuruş)
	totalKurus := uint32(receipt.TotalAmount * 100)
	if err := binary.Write(buf, binary.BigEndian, totalKurus); err != nil {
		return nil, fmt.Errorf("failed to write total amount: %v", err)
	}

	// Payment method (length + UTF-8 bytes)
	paymentBytes := []byte(receipt.PaymentMethod)
	if err := binary.Write(buf, binary.BigEndian, uint32(len(paymentBytes))); err != nil {
		return nil, fmt.Errorf("failed to write payment method length: %v", err)
	}
	if _, err := buf.Write(paymentBytes); err != nil {
		return nil, fmt.Errorf("failed to write payment method: %v", err)
	}

	// Receipt serial (parse 'F' prefix)
	receiptSerial, err := parseReceiptSerial(receipt.ReceiptSerial)
	if err != nil {
		return nil, fmt.Errorf("failed to parse receipt serial: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, receiptSerial); err != nil {
		return nil, fmt.Errorf("failed to write receipt serial: %v", err)
	}

	// Item count
	itemCount := uint16(len(receipt.Items))
	if err := binary.Write(buf, binary.BigEndian, itemCount); err != nil {
		return nil, fmt.Errorf("failed to write item count: %v", err)
	}

	// Items
	for i, item := range receipt.Items {
		if err := serializeItem(buf, item); err != nil {
			return nil, fmt.Errorf("failed to serialize item %d: %v", i, err)
		}
	}

	// Tax breakdown
	if err := serializeTaxBreakdown(buf, receipt.TaxBreakdown); err != nil {
		return nil, fmt.Errorf("failed to serialize tax breakdown: %v", err)
	}

	return buf.Bytes(), nil
}

// NOTE: DeserializeReceipt and ParseSignedReceipt functions are intentionally NOT implemented.
// This cash register system only ISSUES receipts (serialize → hash → sign → encrypt → submit).
// It does not need to READ back or verify receipts, so deserialization functions would be dead code.
// If receipt retrieval/verification features are needed in the future, implement them then.

// CreateSignedReceipt concatenates binary receipt with ECDSA signature
func CreateSignedReceipt(binaryReceipt []byte, signature []byte) ([]byte, error) {
	if len(signature) != SignatureSize {
		return nil, fmt.Errorf("invalid signature size: expected %d bytes, got %d", SignatureSize, len(signature))
	}

	result := make([]byte, len(binaryReceipt)+SignatureSize)
	copy(result, binaryReceipt)
	copy(result[len(binaryReceipt):], signature)

	return result, nil
}

// Helper functions for parsing string fields to integers

func parseZReportNumber(zReport string) (uint32, error) {
	if len(zReport) < 2 || zReport[0] != 'Z' {
		return 0, fmt.Errorf("invalid Z-Report format: %s", zReport)
	}

	var num uint32
	if _, err := fmt.Sscanf(zReport[1:], "%d", &num); err != nil {
		return 0, fmt.Errorf("failed to parse Z-Report number: %v", err)
	}
	return num, nil
}

func parseTransactionID(txID string) (uint32, error) {
	if len(txID) < 11 || txID[:2] != "TX" {
		return 0, fmt.Errorf("invalid transaction ID format: %s", txID)
	}

	// Extract just the sequential number after TXYYYYMMDD
	var num uint32
	if _, err := fmt.Sscanf(txID[10:], "%d", &num); err != nil {
		return 0, fmt.Errorf("failed to parse transaction ID number: %v", err)
	}
	return num, nil
}

func parseVKN(vkn string) (uint32, error) {
	var num uint32
	if _, err := fmt.Sscanf(vkn, "%d", &num); err != nil {
		return 0, fmt.Errorf("failed to parse VKN: %v", err)
	}
	return num, nil
}

func parseReceiptSerial(serial string) (uint32, error) {
	if len(serial) < 2 || serial[0] != 'F' {
		return 0, fmt.Errorf("invalid receipt serial format: %s", serial)
	}

	var num uint32
	if _, err := fmt.Sscanf(serial[1:], "%d", &num); err != nil {
		return 0, fmt.Errorf("failed to parse receipt serial: %v", err)
	}
	return num, nil
}

func serializeItem(buf *bytes.Buffer, item models.Item) error {
	// KisimID (2 bytes)
	if err := binary.Write(buf, binary.BigEndian, uint16(item.KisimID)); err != nil {
		return fmt.Errorf("failed to write KisimID: %v", err)
	}

	// Quantity (2 bytes)
	if err := binary.Write(buf, binary.BigEndian, uint16(item.Quantity)); err != nil {
		return fmt.Errorf("failed to write quantity: %v", err)
	}

	// Unit price in kuruş (4 bytes)
	unitPriceKurus := uint32(item.UnitPrice * 100)
	if err := binary.Write(buf, binary.BigEndian, unitPriceKurus); err != nil {
		return fmt.Errorf("failed to write unit price: %v", err)
	}

	// Total price in kuruş (4 bytes)
	totalPriceKurus := uint32(item.TotalPrice * 100)
	if err := binary.Write(buf, binary.BigEndian, totalPriceKurus); err != nil {
		return fmt.Errorf("failed to write total price: %v", err)
	}

	// Tax rate (1 byte)
	if err := binary.Write(buf, binary.BigEndian, uint8(item.TaxRate)); err != nil {
		return fmt.Errorf("failed to write tax rate: %v", err)
	}

	return nil
}

func serializeTaxBreakdown(buf *bytes.Buffer, tax models.TaxBreakdown) error {
	// Tax 10% base amount in kuruş
	tax10BaseKurus := uint32(tax.Tax10Percent.TaxableAmount * 100)
	if err := binary.Write(buf, binary.BigEndian, tax10BaseKurus); err != nil {
		return fmt.Errorf("failed to write tax 10 base: %v", err)
	}

	// Tax 10% amount in kuruş
	tax10AmountKurus := uint32(tax.Tax10Percent.TaxAmount * 100)
	if err := binary.Write(buf, binary.BigEndian, tax10AmountKurus); err != nil {
		return fmt.Errorf("failed to write tax 10 amount: %v", err)
	}

	// Tax 20% base amount in kuruş
	tax20BaseKurus := uint32(tax.Tax20Percent.TaxableAmount * 100)
	if err := binary.Write(buf, binary.BigEndian, tax20BaseKurus); err != nil {
		return fmt.Errorf("failed to write tax 20 base: %v", err)
	}

	// Tax 20% amount in kuruş
	tax20AmountKurus := uint32(tax.Tax20Percent.TaxAmount * 100)
	if err := binary.Write(buf, binary.BigEndian, tax20AmountKurus); err != nil {
		return fmt.Errorf("failed to write tax 20 amount: %v", err)
	}

	// Total tax amount in kuruş
	totalTaxKurus := uint32(tax.TotalTax * 100)
	if err := binary.Write(buf, binary.BigEndian, totalTaxKurus); err != nil {
		return fmt.Errorf("failed to write total tax: %v", err)
	}

	return nil
}
