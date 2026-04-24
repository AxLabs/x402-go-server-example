package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaidEchoHandler_Success(t *testing.T) {
	body := bytes.NewBufferString(`{"message":"hi there"}`)
	req := httptest.NewRequest(http.MethodPost, "/paid/echo", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	NewPaidEchoHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp PaidEchoResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Echo.OriginalMessage != "hi there" {
		t.Errorf("unexpected echo: %q", resp.Echo.OriginalMessage)
	}
}

func TestPaidEchoHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/paid/echo", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	NewPaidEchoHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPaidEchoHandler_MissingMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/paid/echo", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	NewPaidEchoHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
