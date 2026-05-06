// Package config handles application configuration via environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server            ServerConfig
	LogLevel          string
	Facilitator       FacilitatorConfig
	PaymentConfigFile string
	Payment           PaymentConfig
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

// PaymentConfig mirrors x402 route accepts configuration.
type PaymentConfig struct {
	Routes []PaymentRoute `json:"routes" yaml:"routes"`
}

// PaymentRoute defines one paid route and its accepts list.
type PaymentRoute struct {
	Method      string          `json:"method" yaml:"method"`
	Path        string          `json:"path" yaml:"path"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Accepts     []PaymentAccept `json:"accepts" yaml:"accepts"`
}

// PaymentAccept mirrors x402 PaymentRequirements fields.
type PaymentAccept struct {
	Scheme            string                 `json:"scheme" yaml:"scheme"`
	Network           string                 `json:"network" yaml:"network"`
	Asset             string                 `json:"asset" yaml:"asset"`
	Amount            string                 `json:"amount" yaml:"amount"`
	PayTo             string                 `json:"payTo" yaml:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds,omitempty" yaml:"maxTimeoutSeconds,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty"`
}

type paymentFileConfig struct {
	Payment PaymentConfig `yaml:"payment"`
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

	cfg.PaymentConfigFile = os.Getenv("PAYMENT_CONFIG_FILE")
	if cfg.PaymentConfigFile == "" {
		return nil, fmt.Errorf("PAYMENT_CONFIG_FILE is required")
	}

	raw, err := os.ReadFile(cfg.PaymentConfigFile)
	if err != nil {
		return nil, fmt.Errorf("PAYMENT_CONFIG_FILE: read %q: %w", cfg.PaymentConfigFile, err)
	}

	var fileCfg paymentFileConfig
	if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
		return nil, fmt.Errorf("PAYMENT_CONFIG_FILE: invalid YAML: %w", err)
	}

	cfg.Payment = fileCfg.Payment
	if len(cfg.Payment.Routes) == 0 {
		var direct PaymentConfig
		if err := yaml.Unmarshal(raw, &direct); err == nil && len(direct.Routes) > 0 {
			cfg.Payment = direct
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Facilitator.BaseURL == "" {
		return fmt.Errorf("FACILITATOR_BASE_URL is required")
	}
	if len(c.Payment.Routes) == 0 {
		return fmt.Errorf("payment.routes must include at least one route")
	}
	for i := range c.Payment.Routes {
		r := &c.Payment.Routes[i]
		if r.Method == "" {
			return fmt.Errorf("payment.routes[%d].method is required", i)
		}
		r.Method = strings.ToUpper(r.Method)
		if r.Path == "" {
			return fmt.Errorf("payment.routes[%d].path is required", i)
		}
		if len(r.Accepts) == 0 {
			return fmt.Errorf("payment.routes[%d].accepts must include at least one option", i)
		}
		for j, accept := range r.Accepts {
			if accept.Scheme == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].scheme is required", i, j)
			}
			if accept.Network == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].network is required", i, j)
			}
			if accept.Asset == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].asset is required", i, j)
			}
			if accept.Amount == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].amount is required", i, j)
			}
			if accept.PayTo == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].payTo is required", i, j)
			}
		}
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
