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

func TestNewRouter_RegistersConfiguredPaidRoutes(t *testing.T) {
	cfg := &config.Config{
		Payment: config.PaymentConfig{
			Routes: []config.PaymentRoute{
				{
					Method:  "GET",
					Path:    "/paid/hello",
					Handler: config.PaidHandlerHello,
				},
				{
					Method:  "POST",
					Path:    "/paid/echo",
					Handler: config.PaidHandlerEcho,
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
				},
			},
		},
	}

	_, err := NewRouter(RouterConfig{Config: cfg})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported handler") {
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
