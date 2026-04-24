package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bane-labs-org/x402-paid-server-go/internal/config"
)

func TestInfoHandler(t *testing.T) {
	cfg := &config.Config{
		Facilitator: config.FacilitatorConfig{
			BaseURL: "http://test-facilitator:3000",
		},
		Payment: config.PaymentConfig{
			Network:        "eip155:84532",
			PayToAddress:   "0xpayto",
			PaidHelloPrice: "$0.01",
			PaidEchoPrice:  "$0.005",
		},
	}

	handler := NewInfoHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/info", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response InfoResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Service != "x402-paid-server-go" {
		t.Errorf("expected service name, got %s", response.Service)
	}
	if response.FacilitatorURL != "http://test-facilitator:3000" {
		t.Errorf("expected facilitator URL, got %s", response.FacilitatorURL)
	}
	if response.Network != "eip155:84532" {
		t.Errorf("expected network eip155:84532, got %s", response.Network)
	}
	if response.Scheme != "exact" {
		t.Errorf("expected scheme exact, got %s", response.Scheme)
	}
	if response.PayTo != "0xpayto" {
		t.Errorf("expected payTo 0xpayto, got %s", response.PayTo)
	}
	if response.Pricing.PaidHello.Price != "$0.01" {
		t.Errorf("expected hello price $0.01, got %s", response.Pricing.PaidHello.Price)
	}
	if response.Pricing.PaidEcho.Price != "$0.005" {
		t.Errorf("expected echo price $0.005, got %s", response.Pricing.PaidEcho.Price)
	}
}
