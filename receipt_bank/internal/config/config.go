package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Port    int  `yaml:"port"`
		Verbose bool `yaml:"verbose"`
	} `yaml:"server"`

	Storage struct {
		CleanupInterval string `yaml:"cleanup_interval"`
		MaxReceiptAge   string `yaml:"max_receipt_age"`
	} `yaml:"storage"`

	Webhooks struct {
		Timeout    string `yaml:"timeout"`
		MaxRetries int    `yaml:"max_retries"`
	} `yaml:"webhooks"`
}

// ParsedConfig contains parsed time.Duration values for easier use
type ParsedConfig struct {
	Config
	CleanupInterval time.Duration
	MaxReceiptAge   time.Duration
	WebhookTimeout  time.Duration
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filepath string) (*ParsedConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Parse duration strings
	cleanupInterval, err := time.ParseDuration(cfg.Storage.CleanupInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid cleanup_interval: %v", err)
	}

	maxReceiptAge, err := time.ParseDuration(cfg.Storage.MaxReceiptAge)
	if err != nil {
		return nil, fmt.Errorf("invalid max_receipt_age: %v", err)
	}

	webhookTimeout, err := time.ParseDuration(cfg.Webhooks.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook timeout: %v", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &ParsedConfig{
		Config:          cfg,
		CleanupInterval: cleanupInterval,
		MaxReceiptAge:   maxReceiptAge,
		WebhookTimeout:  webhookTimeout,
	}, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if cfg.Webhooks.MaxRetries < 0 {
		return fmt.Errorf("webhook max_retries must be non-negative")
	}

	return nil
}
