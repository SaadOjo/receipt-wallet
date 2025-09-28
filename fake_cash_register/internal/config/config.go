package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port        int    `yaml:"port"`
		Verbose     bool   `yaml:"verbose"`
		WebhookHost string `yaml:"webhook_host"`
		WebhookPort int    `yaml:"webhook_port"`
	} `yaml:"server"`

	StandaloneMode bool `yaml:"standalone_mode"`

	Store struct {
		VKN     string `yaml:"vkn"`
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
	} `yaml:"store"`

	RevenueAuthority struct {
		URL string `yaml:"url"`
	} `yaml:"revenue_authority"`

	ReceiptBank struct {
		URL string `yaml:"url"`
	} `yaml:"receipt_bank"`

	Kisim []Kisim `yaml:"kisim"`
}

type Kisim struct {
	ID          int     `yaml:"id"`
	Name        string  `yaml:"name"`
	TaxRate     int     `yaml:"tax_rate"`
	PresetPrice float64 `yaml:"preset_price"`
}

func Load() *Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return &config
}
