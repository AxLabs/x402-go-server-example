package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AxLabs/x402-go-server-example/internal/config"
)

func testAccept() []config.PaymentAccept {
	return []config.PaymentAccept{
		{
			Scheme:            "exact",
			Network:           "eip155:84532",
			Asset:             "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount:            "10000",
			PayTo:             "0x1111111111111111111111111111111111111111",
			MaxTimeoutSeconds: 300,
		},
	}
}

func TestNewRouter_RegistersConfiguredPaidRoutes(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "/paid/hello",
					Handler: config.PaidHandlerHello,
					Accepts: testAccept(),
				},
				{
					Method:  "POST",
					Path:    "/paid/echo",
					Handler: config.PaidHandlerEcho,
					Accepts: testAccept(),
				},
			},
		},
	}

	paidMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Paid-MW", "1")
			next.ServeHTTP(w, r)
		})
	}

	router, err := NewRouter(RouterConfig{
		Config:         cfg,
		Logger:         slog.Default(),
		X402Middleware: paidMW,
	})
	if err != nil {
		t.Fatalf("unexpected router error: %v", err)
	}

	helloReq := httptest.NewRequest(http.MethodGet, "/paid/hello", nil)
	helloRec := httptest.NewRecorder()
	router.ServeHTTP(helloRec, helloReq)
	if helloRec.Code != http.StatusOK {
		t.Fatalf("expected hello 200, got %d", helloRec.Code)
	}
	if helloRec.Header().Get("X-Paid-MW") != "1" {
		t.Fatalf("expected paid middleware header on /paid/hello")
	}

	echoReq := httptest.NewRequest(http.MethodPost, "/paid/echo", bytes.NewBufferString(`{"message":"hi"}`))
	echoRec := httptest.NewRecorder()
	router.ServeHTTP(echoRec, echoReq)
	if echoRec.Code != http.StatusOK {
		t.Fatalf("expected echo 200, got %d (body=%s)", echoRec.Code, echoRec.Body.String())
	}
	if echoRec.Header().Get("X-Paid-MW") != "1" {
		t.Fatalf("expected paid middleware header on /paid/echo")
	}
}

func TestNewRouter_InvalidHandlerFails(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "/paid/hello",
					Handler: "not_supported",
					Accepts: testAccept(),
				},
			},
		},
	}

	_, err := NewRouter(RouterConfig{Config: cfg})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must be one of: paid_hello, paid_echo") {
		t.Fatalf("expected unsupported handler error, got %v", err)
	}
}

func TestNewRouter_ReservedRouteFails(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "/healthz",
					Handler: config.PaidHandlerHello,
					Accepts: testAccept(),
				},
			},
		},
	}

	_, err := NewRouter(RouterConfig{Config: cfg})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot use reserved public route") {
		t.Fatalf("expected reserved route error, got %v", err)
	}
}

func TestNewRouter_InvalidMethodFails(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "TRACE",
					Path:    "/paid/hello",
					Handler: config.PaidHandlerHello,
					Accepts: testAccept(),
				},
			},
		},
	}

	_, err := NewRouter(RouterConfig{Config: cfg})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "method must be one of") {
		t.Fatalf("expected invalid method error, got %v", err)
	}
}

func TestNewRouter_InvalidPathFails(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "paid/hello",
					Handler: config.PaidHandlerHello,
					Accepts: testAccept(),
				},
			},
		},
	}

	_, err := NewRouter(RouterConfig{Config: cfg})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "path must start with /") {
		t.Fatalf("expected invalid path error, got %v", err)
	}
}
