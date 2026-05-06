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
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:      "GET",
					Path:        "/paid/hello",
					Handler:     config.PaidHandlerHello,
					Description: "Paid hello resource",
					Accepts: []config.PaymentAccept{
						{Scheme: "exact", Network: "eip155:84532", Asset: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: "10000", PayTo: "0x1111111111111111111111111111111111111111", MaxTimeoutSeconds: 300},
					},
				},
				{
					Method:      "POST",
					Path:        "/paid/echo",
					Handler:     config.PaidHandlerEcho,
					Description: "Paid echo resource",
					Accepts: []config.PaymentAccept{
						{Scheme: "exact", Network: "eip155:84532", Asset: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: "5000", PayTo: "0x1111111111111111111111111111111111111111", MaxTimeoutSeconds: 300},
					},
				},
			},
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
	if len(response.Pricing.Routes) != 2 {
		t.Fatalf("expected 2 priced routes, got %d", len(response.Pricing.Routes))
	}
	if response.Pricing.Routes[0].Path != "/paid/hello" {
		t.Errorf("expected first route /paid/hello, got %s", response.Pricing.Routes[0].Path)
	}
	if response.Pricing.Routes[0].Handler != config.PaidHandlerHello {
		t.Errorf("expected first route handler %s, got %s", config.PaidHandlerHello, response.Pricing.Routes[0].Handler)
	}
	if len(response.Pricing.Routes[0].Accepts) != 1 {
		t.Fatalf("expected 1 accept on /paid/hello, got %d", len(response.Pricing.Routes[0].Accepts))
	}
	if response.Pricing.Routes[0].Accepts[0].Asset != "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48" {
		t.Errorf("expected first route asset 0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48, got %s", response.Pricing.Routes[0].Accepts[0].Asset)
	}
}

func TestInfoHandler_MultipleOptions(t *testing.T) {
	cfg := &config.Config{
		Facilitator: config.FacilitatorConfig{
			BaseURL: "http://test-facilitator:3000",
		},
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "/paid/hello",
					Handler: config.PaidHandlerHello,
					Accepts: []config.PaymentAccept{
						{Scheme: "exact", Network: "eip155:84532", Asset: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: "10000", PayTo: "0x1111111111111111111111111111111111111111", MaxTimeoutSeconds: 300},
						{Scheme: "exact", Network: "eip155:12227332", Asset: "0xd2a4CfF31913016155e38113C7d8e7F4FC7E63DE", Amount: "1000000000000000000", PayTo: "0x1111111111111111111111111111111111111111", MaxTimeoutSeconds: 300, Extra: map[string]interface{}{"name": "xGAS", "version": "1"}},
					},
				},
				{
					Method:  "POST",
					Path:    "/paid/echo",
					Handler: config.PaidHandlerEcho,
					Accepts: []config.PaymentAccept{
						{Scheme: "exact", Network: "eip155:84532", Asset: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: "5000", PayTo: "0x1111111111111111111111111111111111111111", MaxTimeoutSeconds: 300},
					},
				},
			},
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

	if len(response.Pricing.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(response.Pricing.Routes))
	}
	if len(response.Pricing.Routes[0].Accepts) != 2 {
		t.Fatalf("expected /paid/hello to have 2 accepts, got %d", len(response.Pricing.Routes[0].Accepts))
	}
	if response.Pricing.Routes[0].Accepts[1].Network != "eip155:12227332" {
		t.Errorf("expected second /paid/hello accept on eip155:12227332, got %s", response.Pricing.Routes[0].Accepts[1].Network)
	}
	if len(response.Pricing.Routes[1].Accepts) != 1 {
		t.Fatalf("expected /paid/echo to have 1 accept, got %d", len(response.Pricing.Routes[1].Accepts))
	}
}
