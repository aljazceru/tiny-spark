package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BreezAPIKey  string
	BreezMnemonic string
	BreezNetwork string
	BreezWorkingDir string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Could not load .env file: %v\n", err)
		fmt.Println("Using environment variables from system")
	}

	config := &Config{
		BreezAPIKey:     getEnv("BREEZ_API_KEY", ""),
		BreezMnemonic:   getEnv("BREEZ_MNEMONIC", ""),
		BreezNetwork:    getEnv("BREEZ_NETWORK", "mainnet"),
		BreezWorkingDir: getEnv("BREEZ_WORKING_DIR", getEnv("BREEZ_DATA_DIR", ".tiny-spark-data")),
	}

	// Validate only required fields
	if config.BreezAPIKey == "" {
		return nil, fmt.Errorf("BREEZ_API_KEY is required")
	}
	if config.BreezMnemonic == "" {
		return nil, fmt.Errorf("BREEZ_MNEMONIC is required")
	}

	return config, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
