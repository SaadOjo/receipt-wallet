package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port    int  `yaml:"port"`
		Verbose bool `yaml:"verbose"`
	} `yaml:"server"`
	Keys struct {
		PrivateKeyPath string `yaml:"private_key_path"`
		PublicKeyPath  string `yaml:"public_key_path"`
	} `yaml:"keys"`
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