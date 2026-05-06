package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	paymentFile := writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      description: Paid hello resource
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "1000000000000000000"
          payTo: 0x1234567890abcdef
          maxTimeoutSeconds: 120
          extra:
            name: xGAS
            version: "1"
            assetTransferMethod: eip3009
    - method: post
      path: /paid/echo
      handler: paid_echo
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "500000000000000000"
          payTo: 0x1234567890abcdef
`)

	cleanup := setTestEnv(t, map[string]string{
		"SERVER_ADDR":          ":9090",
		"LOG_LEVEL":            "debug",
		"FACILITATOR_BASE_URL": "http://test-facilitator:3000",
		"FACILITATOR_TIMEOUT":  "45s",
		"PAYMENT_CONFIG_FILE":  paymentFile,
		"READ_TIMEOUT":         "20s",
		"WRITE_TIMEOUT":        "25s",
		"REQUEST_TIMEOUT":      "35s",
		"SHUTDOWN_TIMEOUT":     "60s",
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
	if cfg.Server.RequestTimeout != 35*time.Second {
		t.Errorf("expected request timeout 35s, got %v", cfg.Server.RequestTimeout)
	}
	if cfg.Server.ShutdownTimeout != 60*time.Second {
		t.Errorf("expected shutdown timeout 60s, got %v", cfg.Server.ShutdownTimeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.LogLevel)
	}
	if cfg.Facilitator.BaseURL != "http://test-facilitator:3000" {
		t.Errorf("expected facilitator URL, got %s", cfg.Facilitator.BaseURL)
	}
	if cfg.Facilitator.Timeout != 45*time.Second {
		t.Errorf("expected facilitator timeout 45s, got %v", cfg.Facilitator.Timeout)
	}
	if cfg.PaymentConfigFile != paymentFile {
		t.Errorf("expected payment file %s, got %s", paymentFile, cfg.PaymentConfigFile)
	}
	if len(cfg.Payment.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(cfg.Payment.Routes))
	}
	if cfg.Payment.Routes[0].Method != "GET" {
		t.Errorf("expected first method GET, got %s", cfg.Payment.Routes[0].Method)
	}
	if cfg.Payment.Routes[1].Method != "POST" {
		t.Errorf("expected second method POST after normalization, got %s", cfg.Payment.Routes[1].Method)
	}
	if len(cfg.Payment.Routes[0].Accepts) != 1 {
		t.Fatalf("expected first route to have 1 accept, got %d", len(cfg.Payment.Routes[0].Accepts))
	}
	if cfg.Payment.Routes[0].Accepts[0].PayTo != "0x1234567890abcdef" {
		t.Errorf("expected payTo in first accept, got %s", cfg.Payment.Routes[0].Accepts[0].PayTo)
	}
	if cfg.Payment.Routes[0].Accepts[0].Extra["assetTransferMethod"] != "eip3009" {
		t.Errorf("expected extra.assetTransferMethod=eip3009, got %v", cfg.Payment.Routes[0].Accepts[0].Extra["assetTransferMethod"])
	}
}

func TestLoadDefaults(t *testing.T) {
	paymentFile := writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
          payTo: 0xdefault
`)

	cleanup := setTestEnv(t, map[string]string{
		"PAYMENT_CONFIG_FILE":  paymentFile,
		"FACILITATOR_BASE_URL": "http://default-facilitator:3000",
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
	if cfg.Facilitator.Timeout != 30*time.Second {
		t.Errorf("expected default facilitator timeout 30s, got %v", cfg.Facilitator.Timeout)
	}
}

func TestLoadRequiresPaymentConfigFile(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"FACILITATOR_BASE_URL": "http://facilitator:3000",
	})
	defer cleanup()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "PAYMENT_CONFIG_FILE is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadInvalidPaymentFile(t *testing.T) {
	cleanup := setTestEnv(t, map[string]string{
		"FACILITATOR_BASE_URL": "http://facilitator:3000",
		"PAYMENT_CONFIG_FILE":  filepath.Join(t.TempDir(), "missing.yaml"),
	})
	defer cleanup()

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadValidationErrors(t *testing.T) {
	basePaymentFile := writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
          payTo: 0xabc
`)

	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "missing FACILITATOR_BASE_URL",
			env: map[string]string{
				"PAYMENT_CONFIG_FILE": basePaymentFile,
			},
			wantErr: "FACILITATOR_BASE_URL is required",
		},
		{
			name: "missing route accepts",
			env: map[string]string{
				"FACILITATOR_BASE_URL": "http://facilitator:3000",
				"PAYMENT_CONFIG_FILE": writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      accepts: []
`),
			},
			wantErr: "payment.routes[0].accepts must include at least one option",
		},
		{
			name: "missing route handler",
			env: map[string]string{
				"FACILITATOR_BASE_URL": "http://facilitator:3000",
				"PAYMENT_CONFIG_FILE": writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
          payTo: 0xtest
`),
			},
			wantErr: "payment.routes[0].handler is required",
		},
		{
			name: "invalid route handler",
			env: map[string]string{
				"FACILITATOR_BASE_URL": "http://facilitator:3000",
				"PAYMENT_CONFIG_FILE": writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: unknown_handler
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
          payTo: 0xtest
`),
			},
			wantErr: "payment.routes[0].handler must be one of: paid_hello, paid_echo",
		},
		{
			name: "missing accept payTo",
			env: map[string]string{
				"FACILITATOR_BASE_URL": "http://facilitator:3000",
				"PAYMENT_CONFIG_FILE": writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      accepts:
        - scheme: exact
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
`),
			},
			wantErr: "payment.routes[0].accepts[0].payTo is required",
		},
		{
			name: "non exact scheme",
			env: map[string]string{
				"FACILITATOR_BASE_URL": "http://facilitator:3000",
				"PAYMENT_CONFIG_FILE": writeTestPaymentConfig(t, `payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      accepts:
        - scheme: upto
          network: eip155:47763
          asset: 0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE
          amount: "100"
          payTo: 0xtest
`),
			},
			wantErr: "payment.routes[0].accepts[0].scheme must be exact",
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
				t.Fatalf("expected %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func writeTestPaymentConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "payment.yaml")
	if err := os.WriteFile(file, []byte(contents), 0o644); err != nil {
		t.Fatalf("write payment config: %v", err)
	}
	return file
}

// setTestEnv sets environment variables and returns a cleanup function.
func setTestEnv(t *testing.T, env map[string]string) func() {
	t.Helper()

	envKeys := []string{
		"SERVER_ADDR", "LOG_LEVEL", "FACILITATOR_BASE_URL",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "REQUEST_TIMEOUT", "SHUTDOWN_TIMEOUT",
		"FACILITATOR_TIMEOUT", "PAYMENT_CONFIG_FILE",
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
