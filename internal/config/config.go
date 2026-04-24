// Package config handles application configuration via environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server      ServerConfig
	LogLevel    string
	Facilitator FacilitatorConfig
	Network     NetworkConfig
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
	// BaseURL is the facilitator service base URL, e.g. "https://x402.org/facilitator".
	BaseURL string

	// Timeout is the HTTP client timeout for facilitator requests.
	Timeout time.Duration
}

// NetworkConfig holds chain profile settings used by this server.
type NetworkConfig struct {
	// Name is a human-readable profile id, e.g. "neo-x-testnet".
	Name string

	// ChainID is the EVM chain id string, e.g. "12227332".
	ChainID string

	// RPCURL is the RPC endpoint for the selected network.
	RPCURL string

	// ExplorerURL is the block explorer base URL for the selected network.
	ExplorerURL string

	// PaymentAsset is the expected payment asset symbol or address label.
	PaymentAsset string
}

// PaymentConfig holds payment-related configuration.
//
// Values are passed directly to the x402 SDK when building RoutesConfig /
// PaymentOptions. The SDK resolves human-readable USD prices (e.g. "$0.01")
// against the supported asset kinds returned by the facilitator, so this
// service no longer hard-codes chain IDs or asset addresses.
type PaymentConfig struct {
	// Network is the CAIP-2 network identifier passed to the SDK. By default it
	// is derived from CHAIN_ID as eip155:<chain-id>.
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

	// Active default: Base Sepolia (temporary, for go-ethereum v1.16 compatibility).
	// Prepared profile: Neo X Testnet (NETWORK_NAME=neo-x-testnet, CHAIN_ID=12227332,
	//   RPC_URL=https://neoxt4seed1.ngd.network, EXPLORER_URL=https://xt4scan.ngd.network,
	//   PAYMENT_ASSET=USDC). Switch back by setting these env vars once Neo X supports
	//   go-ethereum v1.16.
	cfg.Network.Name = getEnvOrDefault("NETWORK_NAME", "base-sepolia")
	cfg.Network.ChainID = getEnvOrDefault("CHAIN_ID", "84532")
	cfg.Network.RPCURL = getEnvOrDefault("RPC_URL", "https://sepolia.base.org")
	cfg.Network.ExplorerURL = getEnvOrDefault("EXPLORER_URL", "https://sepolia.basescan.org")
	cfg.Network.PaymentAsset = getEnvOrDefault("PAYMENT_ASSET", "USDC")

	cfg.Facilitator.BaseURL = getEnvOrDefault("FACILITATOR_BASE_URL", "https://x402.org/facilitator")
	cfg.Facilitator.Timeout = parseDurationOrDefault("FACILITATOR_TIMEOUT", 30*time.Second)

	cfg.Payment.Network = deriveCAIP2Network(cfg.Network.ChainID)
	if paymentNetwork := os.Getenv("PAYMENT_NETWORK"); paymentNetwork != "" {
		cfg.Payment.Network = paymentNetwork
	}
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
	if c.Network.Name == "" {
		return fmt.Errorf("NETWORK_NAME is required")
	}
	if c.Network.ChainID == "" {
		return fmt.Errorf("CHAIN_ID is required")
	}
	if _, err := strconv.ParseUint(c.Network.ChainID, 10, 64); err != nil {
		return fmt.Errorf("CHAIN_ID must be numeric")
	}
	if c.Network.RPCURL == "" {
		return fmt.Errorf("RPC_URL is required")
	}
	if c.Network.ExplorerURL == "" {
		return fmt.Errorf("EXPLORER_URL is required")
	}
	if c.Network.PaymentAsset == "" {
		return fmt.Errorf("PAYMENT_ASSET is required")
	}
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

func deriveCAIP2Network(chainID string) string {
	trimmed := strings.TrimSpace(chainID)
	if trimmed == "" {
		return ""
	}
	return "eip155:" + trimmed
}
