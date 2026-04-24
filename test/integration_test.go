// Package test contains integration tests that exercise the SDK-backed
// router end-to-end against a mock facilitator.
//
// These tests focus on the integration points this repo owns:
//   - Wiring the SDK middleware onto chi routes.
//   - Returning the SDK-produced 402 (including PAYMENT-REQUIRED header)
//     for unauthenticated paid requests.
//   - Leaving public routes (/healthz, /info, 404) unaffected.
//
// Full end-to-end verify+settle flows require a real EVM signer and a real
// facilitator, which is out of scope for unit-level integration tests.
package test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	x402 "github.com/x402-foundation/x402/go"

	"github.com/bane-labs-org/x402-paid-server-go/internal/config"
	"github.com/bane-labs-org/x402-paid-server-go/internal/httpapi"
	x402wrap "github.com/bane-labs-org/x402-paid-server-go/internal/x402"
)

// newMockFacilitator stands up an httptest.Server that mimics the facilitator
// /verify, /settle and /supported endpoints used by the SDK.
func newMockFacilitator(t *testing.T, network string) *httptest.Server {
	t.Helper()

	handler := http.NewServeMux()

	handler.HandleFunc("/supported", func(w http.ResponseWriter, r *http.Request) {
		v := 2
		resp := map[string]any{
			"kinds": []map[string]any{
				{"x402Version": v, "scheme": "exact", "network": network},
			},
			"extensions": []string{},
			"signers":    map[string][]string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	handler.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(x402.VerifyResponse{IsValid: true, Payer: "0xpayer"})
	})

	handler.HandleFunc("/settle", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(x402.SettleResponse{
			Success:     true,
			Transaction: "0xdeadbeef",
			Network:     x402.Network(network),
			Payer:       "0xpayer",
		})
	})

	return httptest.NewServer(handler)
}

func newTestConfig(facilitatorURL string) *config.Config {
	return &config.Config{
		Server:      config.ServerConfig{Addr: ":0"},
		LogLevel:    "error",
		Facilitator: config.FacilitatorConfig{BaseURL: facilitatorURL},
		Payment: config.PaymentConfig{
			Network:           "eip155:84532",
			PayToAddress:      "0xtest",
			PaidHelloPrice:    "$0.01",
			PaidEchoPrice:     "$0.005",
			MaxTimeoutSeconds: 300,
		},
	}
}

func buildRouter(t *testing.T, cfg *config.Config) http.Handler {
	t.Helper()
	mw, err := x402wrap.Middleware(x402wrap.Config{
		FacilitatorURL:         cfg.Facilitator.BaseURL,
		Network:                httpapi.PaidNetwork(cfg),
		PayTo:                  cfg.Payment.PayToAddress,
		MaxTimeoutSeconds:      cfg.Payment.MaxTimeoutSeconds,
		Routes:                 httpapi.PaidRoutes(cfg),
		SyncFacilitatorOnStart: true,
	})
	if err != nil {
		t.Fatalf("build middleware: %v", err)
	}
	return httpapi.NewRouter(httpapi.RouterConfig{
		Config:         cfg,
		Logger:         slog.Default(),
		X402Middleware: mw,
	})
}

func TestHealthAndInfo_Unauthenticated(t *testing.T) {
	facilitator := newMockFacilitator(t, "eip155:84532")
	defer facilitator.Close()

	cfg := newTestConfig(facilitator.URL)
	router := buildRouter(t, cfg)

	for _, path := range []string{"/healthz", "/info"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d (body=%s)", path, rec.Code, rec.Body.String())
		}
	}
}

func TestPaidHello_Returns402WithoutPaymentHeader(t *testing.T) {
	facilitator := newMockFacilitator(t, "eip155:84532")
	defer facilitator.Close()

	cfg := newTestConfig(facilitator.URL)
	router := buildRouter(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/paid/hello", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("PAYMENT-REQUIRED") == "" {
		t.Error("expected PAYMENT-REQUIRED header on 402 response")
	}
}

func TestPaidEcho_Returns402WithoutPaymentHeader(t *testing.T) {
	facilitator := newMockFacilitator(t, "eip155:84532")
	defer facilitator.Close()

	cfg := newTestConfig(facilitator.URL)
	router := buildRouter(t, cfg)

	req := httptest.NewRequest(http.MethodPost, "/paid/echo", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("PAYMENT-REQUIRED") == "" {
		t.Error("expected PAYMENT-REQUIRED header on 402 response")
	}
}

func TestNotFoundIs404JSON(t *testing.T) {
	facilitator := newMockFacilitator(t, "eip155:84532")
	defer facilitator.Close()

	cfg := newTestConfig(facilitator.URL)
	router := buildRouter(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
