package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AxLabs/x402-go-server-example/internal/config"
)

func TestInfoHandler(t *testing.T) {
	cfg := &config.Config{
		Facilitator: config.FacilitatorConfig{
			BaseURL: "http://test-facilitator:3000",
		},
		Network: config.NetworkConfig{
			Name:         "neox-testnet",
			ChainID:      "12227332",
			RPCURL:       "https://neoxt4seed1.ngd.network",
			ExplorerURL:  "https://xt4scan.ngd.network",
			PaymentAsset: "USDC",
		},
		Payment: config.PaymentConfig{
			Network:        "eip155:12227332",
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

	if response.Service != "x402-go-server-example" {
		t.Errorf("expected service name, got %s", response.Service)
	}
	if response.FacilitatorURL != "http://test-facilitator:3000" {
		t.Errorf("expected facilitator URL, got %s", response.FacilitatorURL)
	}
	if response.Network != "eip155:12227332" {
		t.Errorf("expected network eip155:12227332, got %s", response.Network)
	}
	if response.NetworkName != "neox-testnet" {
		t.Errorf("expected network name neox-testnet, got %s", response.NetworkName)
	}
	if response.ChainID != "12227332" {
		t.Errorf("expected chain id 12227332, got %s", response.ChainID)
	}
	if response.RPCURL != "https://neoxt4seed1.ngd.network" {
		t.Errorf("expected rpc url https://neoxt4seed1.ngd.network, got %s", response.RPCURL)
	}
	if response.ExplorerURL != "https://xt4scan.ngd.network" {
		t.Errorf("expected explorer url https://xt4scan.ngd.network, got %s", response.ExplorerURL)
	}
	if response.PaymentAsset != "USDC" {
		t.Errorf("expected payment asset USDC, got %s", response.PaymentAsset)
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
