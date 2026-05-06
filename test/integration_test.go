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
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	x402 "github.com/x402-foundation/x402/go"

	"github.com/AxLabs/x402-go-server-example/internal/config"
	"github.com/AxLabs/x402-go-server-example/internal/httpapi"
	x402wrap "github.com/AxLabs/x402-go-server-example/internal/x402"
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
			Routes: []config.PaymentRoute{
				{
					Method:      "GET",
					Path:        "/paid/hello",
					Description: "Paid hello resource",
					Accepts: []config.PaymentAccept{
						{
							Scheme:            "exact",
							Network:           "eip155:84532",
							Asset:             "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Amount:            "10000",
							PayTo:             "0xtest",
							MaxTimeoutSeconds: 300,
						},
					},
				},
				{
					Method:      "POST",
					Path:        "/paid/echo",
					Description: "Paid echo resource",
					Accepts: []config.PaymentAccept{
						{
							Scheme:            "exact",
							Network:           "eip155:84532",
							Asset:             "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							Amount:            "5000",
							PayTo:             "0xtest",
							MaxTimeoutSeconds: 300,
						},
					},
				},
			},
		},
	}
}

func buildRouter(t *testing.T, cfg *config.Config) http.Handler {
	t.Helper()
	mw, err := x402wrap.Middleware(x402wrap.Config{
		FacilitatorURL:         cfg.Facilitator.BaseURL,
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

func TestPaidHello_MultiOption402(t *testing.T) {
	// Mock facilitator needs to support both networks.
	handler := http.NewServeMux()
	handler.HandleFunc("/supported", func(w http.ResponseWriter, r *http.Request) {
		v := 2
		resp := map[string]any{
			"kinds": []map[string]any{
				{"x402Version": v, "scheme": "exact", "network": "eip155:84532"},
				{"x402Version": v, "scheme": "exact", "network": "eip155:47763"},
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
			Success: true, Transaction: "0xdeadbeef", Network: "eip155:84532", Payer: "0xpayer",
		})
	})
	facilitator := httptest.NewServer(handler)
	defer facilitator.Close()

	cfg := newTestConfig(facilitator.URL)
	cfg.Payment.Routes[0].Accepts = append(cfg.Payment.Routes[0].Accepts, config.PaymentAccept{
		Scheme:            "exact",
		Network:           "eip155:47763",
		Asset:             "0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE",
		Amount:            "1000000000000000000",
		PayTo:             "0xtest",
		MaxTimeoutSeconds: 300,
		Extra:             map[string]interface{}{"name": "xGAS", "version": "1", "assetTransferMethod": "eip3009"},
	})

	router := buildRouter(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/paid/hello", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	paymentRequired := rec.Header().Get("PAYMENT-REQUIRED")
	if paymentRequired == "" {
		t.Fatal("expected PAYMENT-REQUIRED header")
	}

	// Parse the PAYMENT-REQUIRED header (base64-encoded JSON) to verify multiple accepts
	decoded, err := base64.StdEncoding.DecodeString(paymentRequired)
	if err != nil {
		t.Fatalf("failed to base64 decode PAYMENT-REQUIRED: %v", err)
	}
	var pr struct {
		X402Version int `json:"x402Version"`
		Accepts     []struct {
			Scheme  string `json:"scheme"`
			Network string `json:"network"`
			Asset   string `json:"asset"`
			Amount  string `json:"amount"`
		} `json:"accepts"`
	}
	if err := json.Unmarshal(decoded, &pr); err != nil {
		t.Fatalf("failed to parse PAYMENT-REQUIRED: %v", err)
	}

	if len(pr.Accepts) < 2 {
		t.Fatalf("expected at least 2 accepts, got %d", len(pr.Accepts))
	}
}
