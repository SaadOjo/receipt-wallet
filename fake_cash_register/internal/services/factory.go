package services

import (
	"fake-cash-register/internal/config"
	"fake-cash-register/internal/interfaces"
	"fake-cash-register/internal/services/mock"
	"fake-cash-register/internal/services/real"
)

// CreateServices creates the appropriate service implementations based on configuration
// Returns RevenueAuthorityService, ReceiptBankService, error
func CreateServices(cfg *config.Config) (interfaces.RevenueAuthorityService, interfaces.ReceiptBankService, error) {
	if cfg.StandaloneMode {
		// Standalone mode: use mock services for testing
		revenueAuth := mock.NewMockRevenueAuthority(cfg.Server.Verbose)
		receiptBank := mock.NewMockReceiptBank(cfg.Server.Verbose)

		return revenueAuth, receiptBank, nil
	} else {
		// Online mode: use real HTTP client services
		revenueAuth := real.NewRealRevenueAuthority(cfg.RevenueAuthority.URL, cfg.Server.Verbose)
		receiptBank := real.NewRealReceiptBank(cfg.ReceiptBank.URL, cfg, cfg.Server.Verbose)

		return revenueAuth, receiptBank, nil
	}
}
