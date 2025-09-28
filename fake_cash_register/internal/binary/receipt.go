package binary

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

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

// DeserializeReceipt converts binary format v1 to models.Receipt
func DeserializeReceipt(data []byte) (*models.Receipt, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("data too short for header")
	}

	buf := bytes.NewReader(data)

	// Verify header
	var magic uint16
	if err := binary.Read(buf, binary.BigEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic bytes: %v", err)
	}
	if magic != MagicBytes {
		return nil, fmt.Errorf("invalid magic bytes: expected 0x%04X, got 0x%04X", MagicBytes, magic)
	}

	var version uint8
	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %v", err)
	}
	if version != FormatVersion {
		return nil, fmt.Errorf("unsupported format version: %d", version)
	}

	var reserved uint8
	if err := binary.Read(buf, binary.BigEndian, &reserved); err != nil {
		return nil, fmt.Errorf("failed to read reserved byte: %v", err)
	}
	if reserved != Reserved {
		return nil, fmt.Errorf("invalid reserved byte: expected 0x%02X, got 0x%02X", Reserved, reserved)
	}

	receipt := &models.Receipt{}

	// Timestamp
	var timestamp uint64
	if err := binary.Read(buf, binary.BigEndian, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %v", err)
	}
	receipt.Timestamp = time.Unix(int64(timestamp), 0)

	// Z-Report number
	var zReportNum uint32
	if err := binary.Read(buf, binary.BigEndian, &zReportNum); err != nil {
		return nil, fmt.Errorf("failed to read Z-Report number: %v", err)
	}
	receipt.ZReportNumber = fmt.Sprintf("Z%04d", zReportNum)

	// Transaction ID
	var txID uint32
	if err := binary.Read(buf, binary.BigEndian, &txID); err != nil {
		return nil, fmt.Errorf("failed to read transaction ID: %v", err)
	}
	receipt.TransactionID = fmt.Sprintf("TX%s%04d", receipt.Timestamp.Format("20060102"), txID)

	// Store VKN
	var storeVKN uint32
	if err := binary.Read(buf, binary.BigEndian, &storeVKN); err != nil {
		return nil, fmt.Errorf("failed to read store VKN: %v", err)
	}
	receipt.StoreVKN = fmt.Sprintf("%010d", storeVKN)

	// Store name
	var storeNameLen uint32
	if err := binary.Read(buf, binary.BigEndian, &storeNameLen); err != nil {
		return nil, fmt.Errorf("failed to read store name length: %v", err)
	}
	storeNameBytes := make([]byte, storeNameLen)
	if _, err := buf.Read(storeNameBytes); err != nil {
		return nil, fmt.Errorf("failed to read store name: %v", err)
	}
	receipt.StoreName = string(storeNameBytes)

	// Store address
	var storeAddressLen uint32
	if err := binary.Read(buf, binary.BigEndian, &storeAddressLen); err != nil {
		return nil, fmt.Errorf("failed to read store address length: %v", err)
	}
	storeAddressBytes := make([]byte, storeAddressLen)
	if _, err := buf.Read(storeAddressBytes); err != nil {
		return nil, fmt.Errorf("failed to read store address: %v", err)
	}
	receipt.StoreAddress = string(storeAddressBytes)

	// Total amount
	var totalKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &totalKurus); err != nil {
		return nil, fmt.Errorf("failed to read total amount: %v", err)
	}
	receipt.TotalAmount = float64(totalKurus) / 100.0

	// Payment method
	var paymentLen uint32
	if err := binary.Read(buf, binary.BigEndian, &paymentLen); err != nil {
		return nil, fmt.Errorf("failed to read payment method length: %v", err)
	}
	paymentBytes := make([]byte, paymentLen)
	if _, err := buf.Read(paymentBytes); err != nil {
		return nil, fmt.Errorf("failed to read payment method: %v", err)
	}
	receipt.PaymentMethod = string(paymentBytes)

	// Receipt serial
	var receiptSerial uint32
	if err := binary.Read(buf, binary.BigEndian, &receiptSerial); err != nil {
		return nil, fmt.Errorf("failed to read receipt serial: %v", err)
	}
	receipt.ReceiptSerial = fmt.Sprintf("F%04d", receiptSerial)

	// Item count
	var itemCount uint16
	if err := binary.Read(buf, binary.BigEndian, &itemCount); err != nil {
		return nil, fmt.Errorf("failed to read item count: %v", err)
	}

	// Items
	receipt.Items = make([]models.Item, itemCount)
	for i := uint16(0); i < itemCount; i++ {
		if err := deserializeItem(buf, &receipt.Items[i]); err != nil {
			return nil, fmt.Errorf("failed to deserialize item %d: %v", i, err)
		}
	}

	// Tax breakdown
	if err := deserializeTaxBreakdown(buf, &receipt.TaxBreakdown); err != nil {
		return nil, fmt.Errorf("failed to deserialize tax breakdown: %v", err)
	}

	return receipt, nil
}

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

// ParseSignedReceipt extracts binary receipt and signature
func ParseSignedReceipt(signedData []byte) (binaryReceipt []byte, signature []byte, err error) {
	if len(signedData) < SignatureSize {
		return nil, nil, fmt.Errorf("signed data too short: minimum %d bytes required", SignatureSize)
	}

	receiptLen := len(signedData) - SignatureSize
	binaryReceipt = make([]byte, receiptLen)
	signature = make([]byte, SignatureSize)

	copy(binaryReceipt, signedData[:receiptLen])
	copy(signature, signedData[receiptLen:])

	return binaryReceipt, signature, nil
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

func deserializeItem(buf *bytes.Reader, item *models.Item) error {
	// KisimID
	var kisimID uint16
	if err := binary.Read(buf, binary.BigEndian, &kisimID); err != nil {
		return fmt.Errorf("failed to read KisimID: %v", err)
	}
	item.KisimID = int(kisimID)

	// Quantity
	var quantity uint16
	if err := binary.Read(buf, binary.BigEndian, &quantity); err != nil {
		return fmt.Errorf("failed to read quantity: %v", err)
	}
	item.Quantity = int(quantity)

	// Unit price
	var unitPriceKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &unitPriceKurus); err != nil {
		return fmt.Errorf("failed to read unit price: %v", err)
	}
	item.UnitPrice = float64(unitPriceKurus) / 100.0

	// Total price
	var totalPriceKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &totalPriceKurus); err != nil {
		return fmt.Errorf("failed to read total price: %v", err)
	}
	item.TotalPrice = float64(totalPriceKurus) / 100.0

	// Tax rate
	var taxRate uint8
	if err := binary.Read(buf, binary.BigEndian, &taxRate); err != nil {
		return fmt.Errorf("failed to read tax rate: %v", err)
	}
	item.TaxRate = int(taxRate)

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

func deserializeTaxBreakdown(buf *bytes.Reader, tax *models.TaxBreakdown) error {
	// Tax 10% base
	var tax10BaseKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &tax10BaseKurus); err != nil {
		return fmt.Errorf("failed to read tax 10 base: %v", err)
	}
	tax.Tax10Percent.TaxableAmount = float64(tax10BaseKurus) / 100.0

	// Tax 10% amount
	var tax10AmountKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &tax10AmountKurus); err != nil {
		return fmt.Errorf("failed to read tax 10 amount: %v", err)
	}
	tax.Tax10Percent.TaxAmount = float64(tax10AmountKurus) / 100.0

	// Tax 20% base
	var tax20BaseKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &tax20BaseKurus); err != nil {
		return fmt.Errorf("failed to read tax 20 base: %v", err)
	}
	tax.Tax20Percent.TaxableAmount = float64(tax20BaseKurus) / 100.0

	// Tax 20% amount
	var tax20AmountKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &tax20AmountKurus); err != nil {
		return fmt.Errorf("failed to read tax 20 amount: %v", err)
	}
	tax.Tax20Percent.TaxAmount = float64(tax20AmountKurus) / 100.0

	// Total tax
	var totalTaxKurus uint32
	if err := binary.Read(buf, binary.BigEndian, &totalTaxKurus); err != nil {
		return fmt.Errorf("failed to read total tax: %v", err)
	}
	tax.TotalTax = float64(totalTaxKurus) / 100.0

	return nil
}