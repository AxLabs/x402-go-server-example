package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"SERVER_ADDR":          ":9090",
		"LOG_LEVEL":            "debug",
		"FACILITATOR_BASE_URL": "http://test-facilitator:3000",
		"NETWORK_NAME":         "ethereum-mainnet",
		"CHAIN_ID":             "1",
		"RPC_URL":              "https://eth.example",
		"EXPLORER_URL":         "https://etherscan.io",
		"PAYMENT_ASSET":        "USDC",
		"PAY_TO_ADDRESS":       "0x1234567890abcdef",
		"PAID_HELLO_PRICE":     "$0.02",
		"PAID_ECHO_PRICE":      "$0.01",
		"READ_TIMEOUT":         "20s",
		"WRITE_TIMEOUT":        "25s",
		"REQUEST_TIMEOUT":      "45s",
		"SHUTDOWN_TIMEOUT":     "60s",
		"PAYMENT_MAX_TIMEOUT":  "120s",
	})
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Addr != ":9090" {
		t.Errorf("expected addr :9090, got %s", cfg.Server.Addr)
	}
	if cfg.Server.ReadTimeout != 20*time.Second {
		t.Errorf("expected read timeout 20s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 25*time.Second {
		t.Errorf("expected write timeout 25s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.LogLevel)
	}
	if cfg.Facilitator.BaseURL != "http://test-facilitator:3000" {
		t.Errorf("expected facilitator URL, got %s", cfg.Facilitator.BaseURL)
	}
	if cfg.Network.Name != "ethereum-mainnet" {
		t.Errorf("expected network name ethereum-mainnet, got %s", cfg.Network.Name)
	}
	if cfg.Network.ChainID != "1" {
		t.Errorf("expected chain id 1, got %s", cfg.Network.ChainID)
	}
	if cfg.Network.RPCURL != "https://eth.example" {
		t.Errorf("expected rpc url https://eth.example, got %s", cfg.Network.RPCURL)
	}
	if cfg.Network.ExplorerURL != "https://etherscan.io" {
		t.Errorf("expected explorer url https://etherscan.io, got %s", cfg.Network.ExplorerURL)
	}
	if cfg.Network.PaymentAsset != "USDC" {
		t.Errorf("expected payment asset USDC, got %s", cfg.Network.PaymentAsset)
	}
	if cfg.Payment.Network != "eip155:1" {
		t.Errorf("expected network eip155:1, got %s", cfg.Payment.Network)
	}
	if cfg.Payment.PayToAddress != "0x1234567890abcdef" {
		t.Errorf("expected pay to address, got %s", cfg.Payment.PayToAddress)
	}
	if cfg.Payment.PaidHelloPrice != "$0.02" {
		t.Errorf("expected hello price $0.02, got %s", cfg.Payment.PaidHelloPrice)
	}
	if cfg.Payment.PaidEchoPrice != "$0.01" {
		t.Errorf("expected echo price $0.01, got %s", cfg.Payment.PaidEchoPrice)
	}
	if cfg.Payment.MaxTimeoutSeconds != 120 {
		t.Errorf("expected max timeout 120, got %d", cfg.Payment.MaxTimeoutSeconds)
	}
}

func TestLoadDefaults(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"PAY_TO_ADDRESS": "0xdefault",
	})
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Addr != ":8080" {
		t.Errorf("expected default addr :8080, got %s", cfg.Server.Addr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level info, got %s", cfg.LogLevel)
	}
	if cfg.Network.Name != "base-sepolia" {
		t.Errorf("expected default network name base-sepolia, got %s", cfg.Network.Name)
	}
	if cfg.Network.ChainID != "84532" {
		t.Errorf("expected default chain id 84532, got %s", cfg.Network.ChainID)
	}
	if cfg.Payment.Network != "eip155:84532" {
		t.Errorf("expected default network eip155:84532, got %s", cfg.Payment.Network)
	}
	if cfg.Network.RPCURL != "https://sepolia.base.org" {
		t.Errorf("expected default rpc url, got %s", cfg.Network.RPCURL)
	}
	if cfg.Network.ExplorerURL != "https://sepolia.basescan.org" {
		t.Errorf("expected default explorer url, got %s", cfg.Network.ExplorerURL)
	}
	if cfg.Network.PaymentAsset != "USDC" {
		t.Errorf("expected default payment asset USDC, got %s", cfg.Network.PaymentAsset)
	}
	if cfg.Payment.PaidHelloPrice != "$0.01" {
		t.Errorf("expected default hello price $0.01, got %s", cfg.Payment.PaidHelloPrice)
	}
	if cfg.Payment.PaidEchoPrice != "$0.005" {
		t.Errorf("expected default echo price $0.005, got %s", cfg.Payment.PaidEchoPrice)
	}
	if cfg.Facilitator.BaseURL != "https://x402.org/facilitator" {
		t.Errorf("expected default facilitator URL, got %s", cfg.Facilitator.BaseURL)
	}
}

func TestLoadPaymentNetworkOverride(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"PAY_TO_ADDRESS":       "0xdefault",
		"CHAIN_ID":             "12227332",
		"PAYMENT_NETWORK":      "eip155:84532",
		"FACILITATOR_BASE_URL": "http://test-facilitator:3000",
	})
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Payment.Network != "eip155:84532" {
		t.Errorf("expected override network eip155:84532, got %s", cfg.Payment.Network)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "missing PAY_TO_ADDRESS",
			env:     map[string]string{},
			wantErr: "PAY_TO_ADDRESS is required",
		},
		{
			name: "invalid CHAIN_ID",
			env: map[string]string{
				"PAY_TO_ADDRESS": "0xdefault",
				"CHAIN_ID":       "abc",
			},
			wantErr: "CHAIN_ID must be numeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setTestEnv(t, tt.env)
			defer cleanup()

			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// setTestEnv sets environment variables and returns a cleanup function.
func setTestEnv(t *testing.T, env map[string]string) func() {
	t.Helper()

	envKeys := []string{
		"SERVER_ADDR", "LOG_LEVEL", "FACILITATOR_BASE_URL", "FACILITATOR_TIMEOUT",
		"NETWORK_NAME", "CHAIN_ID", "RPC_URL", "EXPLORER_URL", "PAYMENT_ASSET", "PAYMENT_NETWORK",
		"PAY_TO_ADDRESS", "PAID_HELLO_PRICE", "PAID_ECHO_PRICE", "READ_TIMEOUT",
		"WRITE_TIMEOUT", "REQUEST_TIMEOUT", "SHUTDOWN_TIMEOUT", "PAYMENT_MAX_TIMEOUT",
	}

	original := make(map[string]string)
	for _, k := range envKeys {
		original[k] = os.Getenv(k)
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}

	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}

	return func() {
		for k, v := range original {
			if v == "" {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("restore unset %s: %v", k, err)
				}
			} else {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("restore set %s: %v", k, err)
				}
			}
		}
	}
}
