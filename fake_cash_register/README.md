# Fake Cash Register - Receipt Wallet PoC

A Turkish-style cash register (yazarkasa) simulator built with Go and Gin, featuring a realistic physical cash register interface. Part of the receipt-wallet proof-of-concept ecosystem.

## Features

- **Realistic Cash Register Interface**: Physical cash register appearance with glowing screen and 3D buttons
- **Kisim-Based System**: Traditional Turkish cash register "kisim" (sections) for tax categories
- **Modern Web UI**: Beautiful, minimalist design with authentic cash register styling
- **Dual Mode Operation**: 
  - **Standalone Mode**: Offline demo with mock services
  - **Online Mode**: Integration with revenue authority and receipt bank services
- **QR Code Scanner**: Browser camera integration for ephemeral wallet keys
- **Real-time Updates**: Live transaction display on realistic cash register screen
- **Turkish Localization**: Turkish language interface and proper KDV (VAT) calculations

## Architecture

The application follows clean architecture principles with interface-based design:

- **Interfaces**: Abstract all external services for easy testing and swapping
- **Mock Services**: Complete offline functionality for demos
- **Dependency Injection**: Runtime selection of real vs mock implementations
- **Transaction Workflow**: Complete receipt generation and processing pipeline

## Quick Start

1. **Clone and Build**:
   ```bash
   git clone <repository>
   cd fake_cash_register
   go build -o fake-cash-register cmd/main.go
   ```

2. **Run in Standalone Mode**:
   ```bash
   ./fake-cash-register
   ```
   
3. **Open Browser**:
   ```
   http://localhost:8080
   ```

## Configuration

Edit `config.yaml` to customize:

```yaml
server:
  port: 8080
  verbose: true

standalone_mode: true  # Set false for online mode

store:
  vkn: "1234567890"
  name: "Demo Mağazası"
  address: "Örnek Mahalle, Kadıköy/İstanbul"

products:
  - name: "Ekmek"
    price: 3.50
    tax_rate: 10
    category: "Temel Gıda"
  # ... more products
```

## Usage

### Cash Register Interface

1. **Select Kisim**: Choose tax category (Temel Gıda 10% or Yemek 20%)
2. **Enter Price**: Use numeric keypad to input item price
3. **Set Quantity**: Adjust quantity using +/- buttons or direct input
4. **Add Item**: Click "Ekle" to add item to transaction
5. **Manage Items**: 
   - Click items in screen display to select
   - Use "ÜRÜN SİL" to delete selected items
6. **Payment Method**: Choose NAKİT (Cash) or KART (Card)
7. **Checkout**: Click "ÖDEME AL" to process transaction
8. **QR Scan**: (Online mode) Scan wallet QR code for ephemeral key
9. **Complete**: Transaction is processed and receipt is generated

### API Endpoints

- `GET /` - Main cash register interface
- `POST /api/transaction/start` - Start new transaction
- `POST /api/transaction/add-item` - Add item to transaction
- `POST /api/transaction/process` - Process complete transaction
- `GET /api/kisim` - Get kisim (tax category) list
- `POST /webhook` - Receipt bank webhook endpoint
- `GET /health` - Health check

## Testing

Run the test suite:

```bash
go test ./tests/...
```

Tests cover:
- Transaction workflow
- Receipt calculations
- Mock service functionality
- KDV (VAT) tax calculations

## Integration with Sister Services

### Revenue Authority Service
- Sends receipt hashes for ECDSA signing
- Validates signatures for receipt authenticity
- URL configurable in `config.yaml`

### Receipt Bank Service  
- Submits encrypted receipts for wallet delivery
- Receives webhook confirmations
- Handles ephemeral key encryption

### Wallet Integration
- QR code scanning for ephemeral public keys
- Browser camera API integration
- Encrypts receipts with wallet keys

## Development

### Project Structure
```
fake_cash_register/
├── cmd/main.go                 # Application entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── models/                 # Data structures
│   ├── interfaces/             # Service interfaces
│   ├── services/               # Business logic
│   │   ├── mock/              # Mock implementations
│   │   └── real/              # Real service clients
│   ├── crypto/                # Cryptographic functions
│   └── handlers/              # HTTP request handlers
├── web/
│   ├── templates/             # HTML templates
│   └── static/js/             # JavaScript application
├── tests/                     # Test files
└── config.yaml               # Configuration file
```

### Kisim Configuration

The cash register uses two hardcoded "kisim" (tax categories):

```yaml
kisim:
  - id: 1
    name: "Temel Gıda"
    tax_rate: 10
    description: "Ekmek, süt, temel gıda maddeleri"
  - id: 2
    name: "Yemek"
    tax_rate: 20
    description: "Hazır yemek, atıştırmalık, içecek"
```

These correspond to standard Turkish VAT rates and cannot be modified during operation - just like a real cash register.

### Custom Store Configuration

Update store information in `config.yaml`:

```yaml
store:
  vkn: "your_tax_number"
  name: "Your Store Name"  
  address: "Your Store Address"
```

## Turkish Tax Compliance

- **KDV Rates**: Supports 10% and 20% Turkish VAT rates
- **Receipt Format**: Compliant with Turkish fiscal receipt requirements
- **Z Report Numbers**: Sequential daily report numbering
- **Tax Breakdown**: Detailed KDV calculation by rate

## Troubleshooting

### Common Issues

1. **Port Already in Use**:
   Change port in `config.yaml` or kill existing process

2. **QR Scanner Not Working**:
   - Ensure HTTPS (required for camera access) or use localhost
   - Grant camera permissions in browser
   - Fallback to standalone mode for testing

3. **Build Errors**:
   ```bash
   go mod tidy
   go mod download
   ```

### Verbose Logging

Enable detailed logging:
```yaml
server:
  verbose: true
```

This shows:
- HTTP request logging
- Service interaction details  
- Transaction processing steps
- Real-time log panel in UI

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit pull request

## Support

For issues and questions:
- Check the logs with verbose mode enabled
- Review the test suite for usage examples
- Verify configuration file syntax