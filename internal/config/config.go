// Package config handles application configuration via environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server      ServerConfig
	LogLevel    string
	Facilitator FacilitatorConfig
	Payment     PaymentConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// FacilitatorConfig holds facilitator client configuration.
//
// The facilitator implements the x402 verify/settle/supported API and is
// consumed via the official SDK's HTTPFacilitatorClient.
type FacilitatorConfig struct {
	// BaseURL is the facilitator service base URL.
	BaseURL string

	// Timeout is the HTTP client timeout for facilitator requests.
	Timeout time.Duration
}

// PaymentConfig holds payment-related configuration.
//
// Values are passed directly to the x402 SDK when building RoutesConfig /
// PaymentOptions. The SDK resolves human-readable USD prices (e.g. "$0.01")
// against the supported asset kinds returned by the facilitator, so this
// service no longer hard-codes chain IDs or asset addresses.
type PaymentConfig struct {
	// Network is the CAIP-2 network identifier (e.g. "eip155:84532" for Base Sepolia).
	Network string

	// PayToAddress is the seller's wallet address that receives payments.
	PayToAddress string

	// PaidHelloPrice is the price for GET /paid/hello, expressed as a USD
	// string understood by the SDK (e.g. "$0.01").
	PaidHelloPrice string

	// PaidEchoPrice is the price for POST /paid/echo as a USD string.
	PaidEchoPrice string

	// MaxTimeoutSeconds is the maximum payment timeout advertised per route.
	MaxTimeoutSeconds int
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.Server.Addr = getEnvOrDefault("SERVER_ADDR", ":8080")
	cfg.Server.ReadTimeout = parseDurationOrDefault("READ_TIMEOUT", 15*time.Second)
	cfg.Server.WriteTimeout = parseDurationOrDefault("WRITE_TIMEOUT", 15*time.Second)
	cfg.Server.RequestTimeout = parseDurationOrDefault("REQUEST_TIMEOUT", 30*time.Second)
	cfg.Server.ShutdownTimeout = parseDurationOrDefault("SHUTDOWN_TIMEOUT", 30*time.Second)

	cfg.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")

	cfg.Facilitator.BaseURL = os.Getenv("FACILITATOR_BASE_URL")
	cfg.Facilitator.Timeout = parseDurationOrDefault("FACILITATOR_TIMEOUT", 30*time.Second)

	cfg.Payment.Network = getEnvOrDefault("PAYMENT_NETWORK", "eip155:84532")
	cfg.Payment.PayToAddress = os.Getenv("PAY_TO_ADDRESS")
	cfg.Payment.PaidHelloPrice = getEnvOrDefault("PAID_HELLO_PRICE", "$0.01")
	cfg.Payment.PaidEchoPrice = getEnvOrDefault("PAID_ECHO_PRICE", "$0.005")
	cfg.Payment.MaxTimeoutSeconds = int(parseDurationOrDefault("PAYMENT_MAX_TIMEOUT", 300*time.Second).Seconds())

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Payment.PayToAddress == "" {
		return fmt.Errorf("PAY_TO_ADDRESS is required")
	}
	if c.Payment.Network == "" {
		return fmt.Errorf("PAYMENT_NETWORK is required")
	}
	if c.Payment.PaidHelloPrice == "" {
		return fmt.Errorf("PAID_HELLO_PRICE is required")
	}
	if c.Payment.PaidEchoPrice == "" {
		return fmt.Errorf("PAID_ECHO_PRICE is required")
	}
	if c.Facilitator.BaseURL == "" {
		return fmt.Errorf("FACILITATOR_BASE_URL is required")
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
