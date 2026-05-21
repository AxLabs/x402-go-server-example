// Package config handles application configuration via environment variables.
package config

import (
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Server             ServerConfig
	LogLevel           string
	Facilitator        FacilitatorConfig
	PaymentConfigFile  string
	Payment            PaymentConfig
	CORSAllowedOrigins []string
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

const (
	// PaidHandlerHello serves the paid hello business response.
	PaidHandlerHello = "paid_hello"
	// PaidHandlerEcho serves the paid echo business response.
	PaidHandlerEcho = "paid_echo"
)

// PaymentRoute defines one paid route and its accepts list.
type PaymentRoute struct {
	Method      string          `json:"method" yaml:"method"`
	Path        string          `json:"path" yaml:"path"`
	Handler     string          `json:"handler" yaml:"handler"`
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

var (
	evmAddressRe      = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	evmCAIP2NetworkRe = regexp.MustCompile(`^eip155:[0-9]+$`)
	allowedMethods    = map[string]struct{}{
		"GET":     {},
		"POST":    {},
		"PUT":     {},
		"PATCH":   {},
		"DELETE":  {},
		"HEAD":    {},
		"OPTIONS": {},
	}
)

const maxAcceptTimeoutSeconds = 3600

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.Server.Addr = getEnvOrDefault("SERVER_ADDR", ":8080")
	cfg.Server.ReadTimeout = parseDurationOrDefault("READ_TIMEOUT", 15*time.Second)
	cfg.Server.WriteTimeout = parseDurationOrDefault("WRITE_TIMEOUT", 15*time.Second)
	cfg.Server.RequestTimeout = parseDurationOrDefault("REQUEST_TIMEOUT", 30*time.Second)
	cfg.Server.ShutdownTimeout = parseDurationOrDefault("SHUTDOWN_TIMEOUT", 30*time.Second)

	cfg.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")

	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if o := strings.TrimSpace(part); o != "" {
				cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
			}
		}
	}

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
	return ValidatePaymentRoutes(c.Payment.Routes)
}

// ValidatePaymentRoutes validates paid route configuration.
//
// It normalizes route methods to uppercase and handler/scheme values to lowercase.
func ValidatePaymentRoutes(routes []PaymentRoute) error {
	if len(routes) == 0 {
		return fmt.Errorf("payment.routes must include at least one route")
	}
	seen := make(map[string]struct{}, len(routes))
	for i := range routes {
		r := &routes[i]
		if r.Method == "" {
			return fmt.Errorf("payment.routes[%d].method is required", i)
		}
		r.Method = strings.ToUpper(r.Method)
		if _, ok := allowedMethods[r.Method]; !ok {
			return fmt.Errorf("payment.routes[%d].method must be one of: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS", i)
		}
		if r.Path == "" {
			return fmt.Errorf("payment.routes[%d].path is required", i)
		}
		if !strings.HasPrefix(r.Path, "/") {
			return fmt.Errorf("payment.routes[%d].path must start with /", i)
		}
		if isReservedPublicRoute(r.Method, r.Path) {
			return fmt.Errorf("payment.routes[%d] cannot use reserved public route: %s %s", i, r.Method, r.Path)
		}
		if r.Handler == "" {
			return fmt.Errorf("payment.routes[%d].handler is required", i)
		}
		r.Handler = strings.ToLower(r.Handler)
		if r.Handler != PaidHandlerHello && r.Handler != PaidHandlerEcho {
			return fmt.Errorf("payment.routes[%d].handler must be one of: %s, %s", i, PaidHandlerHello, PaidHandlerEcho)
		}
		key := r.Method + " " + r.Path
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate payment route: %s", key)
		}
		seen[key] = struct{}{}
		if len(r.Accepts) == 0 {
			return fmt.Errorf("payment.routes[%d].accepts must include at least one option", i)
		}
		for j := range r.Accepts {
			accept := &r.Accepts[j]
			if accept.Scheme == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].scheme is required", i, j)
			}
			accept.Scheme = strings.ToLower(accept.Scheme)
			if accept.Scheme != "exact" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].scheme must be exact", i, j)
			}
			if accept.Network == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].network is required", i, j)
			}
			if !evmCAIP2NetworkRe.MatchString(accept.Network) {
				return fmt.Errorf("payment.routes[%d].accepts[%d].network must match eip155:<chainId>", i, j)
			}
			if accept.Asset == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].asset is required", i, j)
			}
			if !evmAddressRe.MatchString(accept.Asset) {
				return fmt.Errorf("payment.routes[%d].accepts[%d].asset must be a valid EVM address", i, j)
			}
			if accept.Amount == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].amount is required", i, j)
			}
			if !isPositiveBase10Integer(accept.Amount) {
				return fmt.Errorf("payment.routes[%d].accepts[%d].amount must be a positive base-10 integer string", i, j)
			}
			if accept.PayTo == "" {
				return fmt.Errorf("payment.routes[%d].accepts[%d].payTo is required", i, j)
			}
			if !evmAddressRe.MatchString(accept.PayTo) {
				return fmt.Errorf("payment.routes[%d].accepts[%d].payTo must be a valid EVM address", i, j)
			}
			if accept.MaxTimeoutSeconds != 0 {
				if accept.MaxTimeoutSeconds < 0 {
					return fmt.Errorf("payment.routes[%d].accepts[%d].maxTimeoutSeconds must be greater than 0 when set", i, j)
				}
				if accept.MaxTimeoutSeconds > maxAcceptTimeoutSeconds {
					return fmt.Errorf("payment.routes[%d].accepts[%d].maxTimeoutSeconds must be <= %d", i, j, maxAcceptTimeoutSeconds)
				}
			}
		}
	}
	return nil
}

func isReservedPublicRoute(method, path string) bool {
	return (method == "GET" && path == "/healthz") || (method == "GET" && path == "/info")
}

func isPositiveBase10Integer(v string) bool {
	if v == "" {
		return false
	}
	if strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") {
		return false
	}
	n, ok := new(big.Int).SetString(v, 10)
	if !ok {
		return false
	}
	return n.Sign() > 0
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
